package contract

import (
	"context"
	"net"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"

	pb "github.com/oseias/ingressos-golang/estoque/gen/pb/estoque"
	adaptadorgrpc "github.com/oseias/ingressos-golang/estoque/internal/adapter/grpc"
	"github.com/oseias/ingressos-golang/estoque/internal/domain/shared"
	"github.com/oseias/ingressos-golang/estoque/internal/platform/config"
	"github.com/oseias/ingressos-golang/estoque/internal/platform/observability"
	"github.com/oseias/ingressos-golang/estoque/internal/usecase"
)

func servidorComMTLS(t *testing.T, estoque *estoqueDeTeste, material materialTLS) string {
	t.Helper()

	obs, err := observability.Iniciar(context.Background(), "error", "")
	if err != nil {
		t.Fatalf("observabilidade: %v", err)
	}

	cfg := &config.Config{
		PoltronasMaxPorBloqueio: 10,
		ReservaTTL:              10 * time.Minute,
		TLSClientAuth:           config.TLSExigido,
		TLSCertFile:             material.certServidor,
		TLSKeyFile:              material.chaveServidor,
		TLSClientCAFile:         material.dir + "/ca.pem",
	}

	servidor, err := adaptadorgrpc.NovoServidor(adaptadorgrpc.Opcoes{
		Bloqueio: usecase.BloquearPoltronas{
			Reservas: estoque, Relogio: shared.RelogioReal{}, Log: logSilencioso{},
			TTL: cfg.ReservaTTL, Limite: cfg.PoltronasMaxPorBloqueio,
		},
		Mapa: usecase.ConsultarMapa{Poltronas: estoque}, Obs: obs, Config: cfg,
	})
	if err != nil {
		t.Fatalf("montar servidor: %v", err)
	}

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listener: %v", err)
	}
	go func() { _ = servidor.ServirEm(lis) }()
	t.Cleanup(servidor.Encerrar)

	return lis.Addr().String()
}

func tentarBloqueio(t *testing.T, endereco string, cred credentials.TransportCredentials) error {
	t.Helper()

	conn, err := grpc.NewClient(endereco, grpc.WithTransportCredentials(cred))
	if err != nil {
		return err
	}
	defer conn.Close()

	ctx, cancelar := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelar()

	_, err = pb.NewServicoEstoqueClient(conn).BloquearPoltronas(ctx, &pb.SolicitacaoBloqueio{
		SessaoId: sessaoProvisionada, PoltronasIds: []string{"A1"}, UsuarioId: usuario,
	})
	return err
}

func TestMTLSRecusaChamadorSemIdentidadeValida(t *testing.T) {
	material := gerarMaterialTLS(t)

	casos := map[string]credentials.TransportCredentials{
		"sem certificado de cliente":     credentials.NewTLS(nil),
		"certificado expirado":           material.credExpirado,
		"certificado de CA desconhecida": material.credOutraCA,
		"sem TLS nenhum":                 insecure.NewCredentials(),
	}

	for nome, cred := range casos {
		t.Run(nome, func(t *testing.T) {
			estoque := novoEstoqueDeTeste()
			endereco := servidorComMTLS(t, estoque, material)

			if err := tentarBloqueio(t, endereco, cred); err == nil {
				t.Fatal("esperava recusa do chamador")
			}
			for rotulo, status := range estoque.poltronas {
				if status != "LIVRE" {
					t.Errorf("chamador recusado alterou %s para %s", rotulo, status)
				}
			}
		})
	}
}

func TestMTLSAceitaChamadorComIdentidadeValida(t *testing.T) {
	material := gerarMaterialTLS(t)
	estoque := novoEstoqueDeTeste()
	endereco := servidorComMTLS(t, estoque, material)

	if err := tentarBloqueio(t, endereco, material.credCliente); err != nil {
		t.Fatalf("chamador com certificado válido devia ser atendido: %v", err)
	}
	if estoque.poltronas["A1"] != "RESERVADA" {
		t.Errorf("A1 = %s, esperado RESERVADA", estoque.poltronas["A1"])
	}
}
