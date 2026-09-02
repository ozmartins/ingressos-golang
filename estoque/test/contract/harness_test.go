package contract

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"

	pb "github.com/oseias/ingressos-golang/estoque/gen/pb/estoque"
	adaptadorgrpc "github.com/oseias/ingressos-golang/estoque/internal/adapter/grpc"
	"github.com/oseias/ingressos-golang/estoque/internal/domain/poltrona"
	"github.com/oseias/ingressos-golang/estoque/internal/domain/reserva"
	"github.com/oseias/ingressos-golang/estoque/internal/domain/shared"
	"github.com/oseias/ingressos-golang/estoque/internal/platform/config"
	"github.com/oseias/ingressos-golang/estoque/internal/platform/observability"
	"github.com/oseias/ingressos-golang/estoque/internal/usecase"
)

const (
	sessaoProvisionada = "f781a9b2-11e2-4f81-a901-8890bc123456"
	usuario            = "c394c8b3-76a1-4328-b803-02f5923b7a15"
)

type estoqueDeTeste struct {
	mu        sync.Mutex
	poltronas map[string]poltrona.Status
	erro      error
}

func novoEstoqueDeTeste() *estoqueDeTeste {
	return &estoqueDeTeste{poltronas: map[string]poltrona.Status{
		"A1": poltrona.Livre, "A2": poltrona.Livre, "A3": poltrona.Livre,
		"B1": poltrona.Livre, "B2": poltrona.Livre,
	}}
}

func (e *estoqueDeTeste) Conceder(_ context.Context, sol reserva.Solicitacao, _ reserva.Reserva, _ usecase.FatoPendente) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.erro != nil {
		return e.erro
	}
	if sol.SessaoID != sessaoProvisionada {
		return fmt.Errorf("%w: %s", shared.ErrSessaoNaoProvisionada, sol.SessaoID)
	}
	for _, rotulo := range sol.Rotulos {
		status, existe := e.poltronas[rotulo]
		if !existe {
			return fmt.Errorf("%w: %s", shared.ErrPoltronaInexistente, rotulo)
		}
		if status != poltrona.Livre {
			return fmt.Errorf("%w: %s", shared.ErrPoltronasIndisponiveis, rotulo)
		}
	}
	for _, rotulo := range sol.Rotulos {
		e.poltronas[rotulo] = poltrona.Reservada
	}
	return nil
}

func (e *estoqueDeTeste) Confirmar(context.Context, string, string, string, time.Time) (usecase.ResultadoTransicao, error) {
	return usecase.TransicaoAplicada, nil
}
func (e *estoqueDeTeste) Cancelar(context.Context, string, string, string, time.Time) (usecase.ResultadoTransicao, error) {
	return usecase.TransicaoAplicada, nil
}
func (e *estoqueDeTeste) ExpirarVencidas(context.Context, time.Time, int) ([]string, error) {
	return nil, nil
}
func (e *estoqueDeTeste) ExpirarUma(context.Context, string, time.Time) (usecase.ResultadoTransicao, error) {
	return usecase.TransicaoAplicada, nil
}

func (e *estoqueDeTeste) MapaDaSessao(_ context.Context, sessaoID string) ([]poltrona.Poltrona, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.erro != nil {
		return nil, e.erro
	}
	if sessaoID != sessaoProvisionada {
		return nil, nil
	}
	var mapa []poltrona.Poltrona
	for rotulo, status := range e.poltronas {
		fileira, numero, _ := poltrona.LerRotulo(rotulo)
		mapa = append(mapa, poltrona.Poltrona{
			SessaoID: sessaoID, Fileira: fileira, Numero: numero,
			Rotulo: rotulo, Tipo: poltrona.Normal, Status: status,
		})
	}
	return mapa, nil
}

func (e *estoqueDeTeste) ProvisionarMatriz(context.Context, string, string, string, []poltrona.Poltrona) (usecase.ResultadoTransicao, error) {
	return usecase.TransicaoAplicada, nil
}

type logSilencioso struct{}

func (logSilencioso) Info(string, ...any)  {}
func (logSilencioso) Warn(string, ...any)  {}
func (logSilencioso) Error(string, ...any) {}

func servidorEmMemoria(t *testing.T, estoque *estoqueDeTeste) pb.ServicoEstoqueClient {
	t.Helper()

	obs, err := observability.Iniciar(context.Background(), "error", "")
	if err != nil {
		t.Fatalf("observabilidade: %v", err)
	}

	cfg := &config.Config{PoltronasMaxPorBloqueio: 10, ReservaTTL: 10 * time.Minute, TLSClientAuth: config.TLSDesligado}
	servidor, err := adaptadorgrpc.NovoServidor(adaptadorgrpc.Opcoes{
		Bloqueio: usecase.BloquearPoltronas{
			Reservas: estoque, Relogio: shared.RelogioReal{}, Log: logSilencioso{},
			TTL: cfg.ReservaTTL, Limite: cfg.PoltronasMaxPorBloqueio,
		},
		Mapa:   usecase.ConsultarMapa{Poltronas: estoque},
		Obs:    obs,
		Config: cfg,
	})
	if err != nil {
		t.Fatalf("montar servidor: %v", err)
	}

	lis := bufconn.Listen(1024 * 1024)
	go func() { _ = servidor.ServirEm(lis) }()
	t.Cleanup(servidor.Encerrar)

	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) { return lis.DialContext(ctx) }),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("cliente: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	return pb.NewServicoEstoqueClient(conn)
}

type materialTLS struct {
	dir                         string
	caPEM                       []byte
	certServidor, chaveServidor string
	credCliente, credExpirado   credentials.TransportCredentials
	credOutraCA, credSemCliente credentials.TransportCredentials
}

func gerarMaterialTLS(t *testing.T) materialTLS {
	t.Helper()
	dir := t.TempDir()

	caCert, caChave := gerarCA(t, "estoque-teste-ca")
	certServidor, chaveServidor := emitir(t, caCert, caChave, "estoque", time.Hour)
	certCliente, chaveCliente := emitir(t, caCert, caChave, "servico-catalogo", time.Hour)
	certExpirado, chaveExpirada := emitir(t, caCert, caChave, "servico-catalogo", -time.Hour)

	outraCA, outraChave := gerarCA(t, "ca-desconhecida")
	certIntruso, chaveIntruso := emitir(t, outraCA, outraChave, "intruso", time.Hour)

	caPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caCert.Raw})
	ancoras := x509.NewCertPool()
	ancoras.AppendCertsFromPEM(caPEM)

	escrever := func(nome string, conteudo []byte) string {
		caminho := filepath.Join(dir, nome)
		if err := os.WriteFile(caminho, conteudo, 0o600); err != nil {
			t.Fatalf("escrever %s: %v", nome, err)
		}
		return caminho
	}

	m := materialTLS{
		dir:           dir,
		caPEM:         caPEM,
		certServidor:  escrever("servidor.pem", certServidor),
		chaveServidor: escrever("servidor-key.pem", chaveServidor),
	}
	escrever("ca.pem", caPEM)

	credDe := func(certPEM, chavePEM []byte) credentials.TransportCredentials {
		par, err := tls.X509KeyPair(certPEM, chavePEM)
		if err != nil {
			t.Fatalf("par de cliente: %v", err)
		}
		return credentials.NewTLS(&tls.Config{
			Certificates: []tls.Certificate{par},
			RootCAs:      ancoras,
			ServerName:   "estoque",
			MinVersion:   tls.VersionTLS13,
		})
	}
	m.credSemCliente = credentials.NewTLS(&tls.Config{
		RootCAs: ancoras, ServerName: "estoque", MinVersion: tls.VersionTLS13,
	})
	m.credCliente = credDe(certCliente, chaveCliente)
	m.credExpirado = credDe(certExpirado, chaveExpirada)
	m.credOutraCA = credDe(certIntruso, chaveIntruso)
	return m
}

func gerarCA(t *testing.T, nome string) (*x509.Certificate, *ecdsa.PrivateKey) {
	t.Helper()
	chave, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	modelo := &x509.Certificate{
		SerialNumber:          big.NewInt(time.Now().UnixNano()),
		Subject:               pkix.Name{CommonName: nome},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, modelo, modelo, &chave.PublicKey, chave)
	if err != nil {
		t.Fatal(err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}
	return cert, chave
}

func emitir(t *testing.T, ca *x509.Certificate, caChave *ecdsa.PrivateKey, cn string, validade time.Duration) (certPEM, chavePEM []byte) {
	t.Helper()
	chave, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	inicio := time.Now().Add(-2 * time.Hour)
	modelo := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		Subject:      pkix.Name{CommonName: cn},
		NotBefore:    inicio,
		NotAfter:     time.Now().Add(validade),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		DNSNames:     []string{"estoque", "localhost"},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
	}
	der, err := x509.CreateCertificate(rand.Reader, modelo, ca, &chave.PublicKey, caChave)
	if err != nil {
		t.Fatal(err)
	}
	derChave, err := x509.MarshalECPrivateKey(chave)
	if err != nil {
		t.Fatal(err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}),
		pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: derChave})
}
