package grpc

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"os"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"

	pb "github.com/oseias/ingressos-golang/estoque/gen/pb/estoque"
	"github.com/oseias/ingressos-golang/estoque/internal/platform/config"
	"github.com/oseias/ingressos-golang/estoque/internal/platform/observability"
)

type Servidor struct {
	pb.UnimplementedServicoEstoqueServer

	bloqueio CasoDeUsoBloqueio
	mapa     CasoDeUsoMapa
	obs      *observability.Observabilidade
	metricas *metricas
	limite   int
	grpc     *grpc.Server
}

type Opcoes struct {
	Bloqueio CasoDeUsoBloqueio
	Mapa     CasoDeUsoMapa
	Obs      *observability.Observabilidade
	Config   *config.Config
}

func NovoServidor(opts Opcoes) (*Servidor, error) {
	m, err := novasMetricas(opts.Obs)
	if err != nil {
		return nil, err
	}

	s := &Servidor{
		bloqueio: opts.Bloqueio,
		mapa:     opts.Mapa,
		obs:      opts.Obs,
		metricas: m,
		limite:   opts.Config.PoltronasMaxPorBloqueio,
	}

	servidorOpts := []grpc.ServerOption{
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
	}

	if opts.Config.TLSClientAuth == config.TLSExigido {
		cred, err := credenciaisMTLS(opts.Config, opts.Obs)
		if err != nil {
			return nil, err
		}
		servidorOpts = append(servidorOpts, grpc.Creds(cred))
	} else {
		opts.Obs.Log.Warn("canal síncrono sem autenticação de serviço",
			"modo", string(opts.Config.TLSClientAuth),
			"aviso", "aceitável apenas em desenvolvimento (FR-037)")
	}

	s.grpc = grpc.NewServer(servidorOpts...)
	pb.RegisterServicoEstoqueServer(s.grpc, s)
	return s, nil
}

func credenciaisMTLS(cfg *config.Config, obs *observability.Observabilidade) (credentials.TransportCredentials, error) {
	par, err := tls.LoadX509KeyPair(cfg.TLSCertFile, cfg.TLSKeyFile)
	if err != nil {
		return nil, fmt.Errorf("carregar par de certificado do servidor: %w", err)
	}
	pemCA, err := os.ReadFile(cfg.TLSClientCAFile)
	if err != nil {
		return nil, fmt.Errorf("carregar âncora de confiança: %w", err)
	}
	ancoras := x509.NewCertPool()
	if !ancoras.AppendCertsFromPEM(pemCA) {
		return nil, fmt.Errorf("âncora de confiança em %s não contém certificado válido", cfg.TLSClientCAFile)
	}

	conf := &tls.Config{
		Certificates: []tls.Certificate{par},
		ClientCAs:    ancoras,
		ClientAuth:   tls.RequireAndVerifyClientCert,
		MinVersion:   tls.VersionTLS13,
		VerifyConnection: func(cs tls.ConnectionState) error {
			if len(cs.PeerCertificates) > 0 {
				obs.Log.Debug("chamador autenticado", "subject", cs.PeerCertificates[0].Subject.CommonName)
			}
			return nil
		},
	}

	return &credenciaisAuditadas{
		TransportCredentials: credentials.NewTLS(conf),
		obs:                  obs,
	}, nil
}

type credenciaisAuditadas struct {
	credentials.TransportCredentials
	obs *observability.Observabilidade
}

func (c *credenciaisAuditadas) ServerHandshake(conn net.Conn) (net.Conn, credentials.AuthInfo, error) {
	envolvida, info, err := c.TransportCredentials.ServerHandshake(conn)
	if err != nil {
		origem := "desconhecida"
		if conn.RemoteAddr() != nil {
			origem = conn.RemoteAddr().String()
		}
		c.obs.Log.Warn("conexão recusada por falha de autenticação de serviço",
			"origem", origem,
			"motivo", err.Error(),
			"instante", time.Now().UTC().Format(time.RFC3339))
		return nil, nil, err
	}
	if tlsInfo, ok := info.(credentials.TLSInfo); ok && len(tlsInfo.State.PeerCertificates) > 0 {
		c.obs.Log.Debug("handshake concluído",
			"subject", tlsInfo.State.PeerCertificates[0].Subject.CommonName)
	}
	return envolvida, info, nil
}

func (c *credenciaisAuditadas) Clone() credentials.TransportCredentials {
	return &credenciaisAuditadas{TransportCredentials: c.TransportCredentials.Clone(), obs: c.obs}
}

func (s *Servidor) Servir(addr string) error {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("escutar em %s: %w", addr, err)
	}
	s.obs.Log.Info("servidor gRPC no ar", "endereco", addr)
	return s.grpc.Serve(lis)
}

func (s *Servidor) ServirEm(lis net.Listener) error { return s.grpc.Serve(lis) }

func (s *Servidor) Encerrar() { s.grpc.GracefulStop() }

func subjectDoChamador(ctx context.Context) string {
	p, ok := peer.FromContext(ctx)
	if !ok {
		return ""
	}
	tlsInfo, ok := p.AuthInfo.(credentials.TLSInfo)
	if !ok || len(tlsInfo.State.PeerCertificates) == 0 {
		return ""
	}
	return tlsInfo.State.PeerCertificates[0].Subject.CommonName
}
