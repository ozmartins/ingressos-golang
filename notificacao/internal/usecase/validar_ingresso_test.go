package usecase

import (
	"context"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/oseias/ingressos-golang/notificacao/internal/domain/ingresso"
)

func validador(repo *ingressosFalsos) ValidarIngresso {
	return ValidarIngresso{
		Ingressos: repo, Assinador: assinadorFalso{}, Relogio: relogioFixo{},
		Log: slog.New(slog.NewJSONHandler(io.Discard, nil)),
	}
}

func semear(t *testing.T, repo *ingressosFalsos, status ingresso.Status) ingresso.Ingresso {
	t.Helper()
	i, err := ingresso.Novo("ing-1", reserva1, usuario1, "CIN1.ing-1.assinatura", instanteFixo)
	if err != nil {
		t.Fatalf("preparação: %v", err)
	}
	switch status {
	case ingresso.Utilizado:
		i, _ = i.Utilizar(instanteFixo.Add(time.Hour))
	case ingresso.Cancelado:
		i, _ = i.Cancelar()
	}
	repo.semear(i)
	return i
}

func TestValidarAutorizaEDaBaixa(t *testing.T) {
	repo := novosIngressos()
	i := semear(t, repo, ingresso.Valido)

	r, err := validador(repo).Executar(context.Background(), i.CodigoQR)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if r.Veredito != Autorizada {
		t.Errorf("veredito = %v, queria Autorizada", r.Veredito)
	}
	if r.Ingresso.Status != ingresso.Utilizado {
		t.Errorf("status = %q, queria UTILIZADO", r.Ingresso.Status)
	}
	if r.Ingresso.UtilizadoEm == nil {
		t.Error("baixa sem instante de utilização")
	}
}

func TestSegundaLeituraENegadaSemAlterarInstante(t *testing.T) {
	repo := novosIngressos()
	i := semear(t, repo, ingresso.Valido)
	u := validador(repo)
	ctx := context.Background()

	primeira, err := u.Executar(ctx, i.CodigoQR)
	if err != nil {
		t.Fatalf("primeira leitura: %v", err)
	}
	segunda, err := u.Executar(ctx, i.CodigoQR)
	if err != nil {
		t.Fatalf("segunda leitura: %v", err)
	}
	if segunda.Veredito != Reuso {
		t.Errorf("veredito = %v, queria Reuso", segunda.Veredito)
	}
	if !segunda.Ingresso.UtilizadoEm.Equal(*primeira.Ingresso.UtilizadoEm) {
		t.Error("o instante de utilização original foi alterado (FR-008)")
	}
}

func TestIngressoCanceladoENegado(t *testing.T) {
	repo := novosIngressos()
	i := semear(t, repo, ingresso.Cancelado)

	r, err := validador(repo).Executar(context.Background(), i.CodigoQR)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if r.Veredito != NaoValido {
		t.Errorf("veredito = %v, queria NaoValido", r.Veredito)
	}
	if r.Ingresso.Status != ingresso.Cancelado {
		t.Errorf("o ingresso deixou de estar cancelado: %q", r.Ingresso.Status)
	}
}

func TestAssinaturaInvalidaNaoConsultaOAcervo(t *testing.T) {
	repo := novosIngressos()
	semear(t, repo, ingresso.Valido)

	r, err := validador(repo).Executar(context.Background(), "CIN1.ing-1.forjada")
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if r.Veredito != NaoEncontrado {
		t.Errorf("veredito = %v, queria NaoEncontrado", r.Veredito)
	}
	if repo.chamouBuscar {
		t.Error("o repositório foi consultado apesar de a assinatura ser inválida (FR-010)")
	}
}

func TestCodigoMalformadoEInexistenteDaoOMesmoVeredito(t *testing.T) {
	repo := novosIngressos()
	u := validador(repo)
	ctx := context.Background()

	casos := map[string]string{
		"malformado":  "lixo",
		"vazio":       "",
		"inexistente": "CIN1.ing-nao-existe.assinatura",
	}
	for nome, c := range casos {
		t.Run(nome, func(t *testing.T) {
			r, err := u.Executar(ctx, c)
			if err != nil {
				t.Fatalf("erro inesperado: %v", err)
			}
			if r.Veredito != NaoEncontrado {
				t.Errorf("veredito = %v, queria NaoEncontrado", r.Veredito)
			}
			if r.TemIgresso {
				t.Error("recusa não pode devolver ingresso")
			}
		})
	}
}

func TestLeiturasSimultaneasAutorizamUmaSo(t *testing.T) {
	repo := novosIngressos()
	i := semear(t, repo, ingresso.Valido)
	u := validador(repo)

	const n = 16
	var wg sync.WaitGroup
	autorizadas := make(chan struct{}, n)
	for k := 0; k < n; k++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r, err := u.Executar(context.Background(), i.CodigoQR)
			if err == nil && r.Veredito == Autorizada {
				autorizadas <- struct{}{}
			}
		}()
	}
	wg.Wait()
	close(autorizadas)
	if got := len(autorizadas); got != 1 {
		t.Errorf("%d autorizações, queria exatamente 1 (FR-011)", got)
	}
}
