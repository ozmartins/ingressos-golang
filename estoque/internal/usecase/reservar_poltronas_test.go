package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	dominiosessao "github.com/oseias/ingressos-golang/estoque/internal/domain/sessao"
	"github.com/oseias/ingressos-golang/estoque/internal/domain/shared"
)

type catalogoFalso struct {
	sessao   dominiosessao.Sessao
	err      error
	chamadas int
}

func (c *catalogoFalso) BuscarSessao(context.Context, string) (dominiosessao.Sessao, error) {
	c.chamadas++
	return c.sessao, c.err
}

func sessaoAgendada() dominiosessao.Sessao {
	return dominiosessao.Sessao{
		ID: sessao, Status: dominiosessao.Agendada, PrecoBase: "42.00", Inicio: agora.Add(time.Hour),
	}
}

func montarReserva(c *catalogoFalso, e *estoqueFalso) ReservarPoltronas {
	return ReservarPoltronas{
		Sessoes:  c,
		Bloqueio: montarBloqueio(e, novoPrazoFalso(), &logFalso{}),
		Relogio:  shared.NovoRelogioFixo(agora),
		Log:      &logFalso{},
	}
}

func TestReservaCalculaOValorEBloqueia(t *testing.T) {
	estoque := novoEstoqueFalso()
	estoque.provisionar(sessao, "A1", "A2")
	catalogo := &catalogoFalso{sessao: sessaoAgendada()}

	resultado, err := montarReserva(catalogo, estoque).Executar(context.Background(), sessao, usuario, []string{"A1", "A2"})
	if err != nil || !resultado.Concedido {
		t.Fatalf("resultado = %+v, err = %v; esperava reserva concedida", resultado, err)
	}
	if got := resultado.Reserva.ValorTotal; got != "84.00" {
		t.Errorf("valor_total = %q, esperado 84.00 (2 × 42.00)", got)
	}
}

func TestReservaRecusaSessaoNaoReservavel(t *testing.T) {
	cancelada := sessaoAgendada()
	cancelada.Status = "CANCELADA"
	jaComecou := sessaoAgendada()
	jaComecou.Inicio = agora.Add(-time.Minute)

	for nome, s := range map[string]dominiosessao.Sessao{"cancelada": cancelada, "já começou": jaComecou} {
		t.Run(nome, func(t *testing.T) {
			estoque := novoEstoqueFalso()
			estoque.provisionar(sessao, "A1")

			_, err := montarReserva(&catalogoFalso{sessao: s}, estoque).Executar(context.Background(), sessao, usuario, []string{"A1"})
			if !errors.Is(err, shared.ErrSessaoNaoReservavel) {
				t.Fatalf("err = %v, esperado ErrSessaoNaoReservavel", err)
			}
			if got := estoque.statusDe(sessao, "A1"); got != "LIVRE" {
				t.Errorf("A1 = %s; a poltrona não deveria ser tocada", got)
			}
		})
	}
}

func TestReservaPropagaErrosDoCatalogo(t *testing.T) {
	for _, e := range []error{shared.ErrSessaoNaoEncontrada, shared.ErrCatalogoIndisponivel, shared.ErrRespostaInvalidaDoCatalogo} {
		_, err := montarReserva(&catalogoFalso{err: e}, novoEstoqueFalso()).Executar(context.Background(), sessao, usuario, []string{"A1"})
		if !errors.Is(err, e) {
			t.Errorf("err = %v, esperado %v", err, e)
		}
	}
}

func TestReservaInvalidaNaoConsultaOCatalogo(t *testing.T) {
	catalogo := &catalogoFalso{sessao: sessaoAgendada()}

	for nome, rotulos := range map[string][]string{"sem poltronas": nil} {
		_, err := montarReserva(catalogo, novoEstoqueFalso()).Executar(context.Background(), sessao, usuario, rotulos)
		if !errors.Is(err, shared.ErrSolicitacaoInvalida) {
			t.Errorf("%s: err = %v, esperado ErrSolicitacaoInvalida", nome, err)
		}
	}
	if catalogo.chamadas != 0 {
		t.Errorf("catálogo consultado %d vez(es) para pedido inválido", catalogo.chamadas)
	}
}
