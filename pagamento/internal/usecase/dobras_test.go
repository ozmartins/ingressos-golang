package usecase

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/oseias/ingressos-golang/pagamento/internal/domain/transacao"
)

var (
	instante = time.Date(2026, 8, 30, 12, 0, 0, 0, time.UTC)
	prazoOK  = instante.Add(10 * time.Minute)
	prazoIdo = instante.Add(-time.Minute)
)

type relogioFixo struct{ t time.Time }

func (r relogioFixo) Agora() time.Time { return r.t }

type idsFixos struct{ id string }

func (g idsFixos) Novo() string { return g.id }

// repoFalso guarda uma transação por reserva e registra as chamadas, para que os
// testes possam afirmar o que NÃO aconteceu.
type repoFalso struct {
	mu            sync.Mutex
	porReserva    map[string]transacao.Transacao
	erroCriar     error
	erroFinalizar error
	erroMarcar    error
	Finalizacoes  int
	Marcacoes     int
}

func novoRepo() *repoFalso {
	return &repoFalso{porReserva: map[string]transacao.Transacao{}}
}

func (r *repoFalso) semear(t transacao.Transacao) { r.porReserva[t.ReservaID] = t }

func (r *repoFalso) CriarSeAusente(_ context.Context, t transacao.Transacao) (bool, transacao.Transacao, error) {
	if r.erroCriar != nil {
		return false, transacao.Transacao{}, r.erroCriar
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if existente, ok := r.porReserva[t.ReservaID]; ok {
		return false, existente, nil
	}
	r.porReserva[t.ReservaID] = t
	return true, t, nil
}

func (r *repoFalso) BuscarPorReserva(_ context.Context, reservaID string) (transacao.Transacao, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	t, ok := r.porReserva[reservaID]
	if !ok {
		return transacao.Transacao{}, ErrNaoEncontrada
	}
	return t, nil
}

func (r *repoFalso) Finalizar(_ context.Context, t transacao.Transacao) error {
	if r.erroFinalizar != nil {
		return r.erroFinalizar
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	atual, ok := r.porReserva[t.ReservaID]
	if !ok {
		return ErrNaoEncontrada
	}
	if atual.Status.Final() {
		return ErrJaFinalizada
	}
	// Espelha o conjunto de colunas do UPDATE real (postgres.Finalizar): status,
	// código, motivo, pago_em e atualizado_em. cobranca_emitida e
	// resultado_anunciado NÃO são tocados aqui — têm operações próprias. Uma
	// dobra que substituísse a linha inteira mentiria sobre a persistência.
	atual.Status = t.Status
	atual.CodigoTransacaoGateway = t.CodigoTransacaoGateway
	atual.MotivoFalha = t.MotivoFalha
	atual.PagoEm = t.PagoEm
	atual.AtualizadoEm = t.AtualizadoEm
	r.porReserva[t.ReservaID] = atual
	r.Finalizacoes++
	return nil
}

// ReivindicarCobranca replica a atomicidade do UPDATE condicionado do banco.
func (r *repoFalso) ReivindicarCobranca(_ context.Context, id string, agora time.Time) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for k, t := range r.porReserva {
		if t.ID == id && t.Status == transacao.Processando && !t.CobrancaEmitida {
			t.CobrancaEmitida = true
			t.AtualizadoEm = agora
			r.porReserva[k] = t
			return true, nil
		}
	}
	return false, nil
}

func (r *repoFalso) LiberarCobranca(_ context.Context, id string, agora time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for k, t := range r.porReserva {
		if t.ID == id && t.Status == transacao.Processando {
			t.CobrancaEmitida = false
			t.AtualizadoEm = agora
			r.porReserva[k] = t
			return nil
		}
	}
	return nil
}

func (r *repoFalso) MarcarAnunciado(_ context.Context, id string, agora time.Time) error {
	if r.erroMarcar != nil {
		return r.erroMarcar
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for k, t := range r.porReserva {
		if t.ID == id {
			t.ResultadoAnunciado = true
			t.AtualizadoEm = agora
			r.porReserva[k] = t
			r.Marcacoes++
			return nil
		}
	}
	return ErrNaoEncontrada
}

// adquirenteFalso conta cobranças — é assim que os testes provam que ninguém foi
// cobrado duas vezes.
type adquirenteFalso struct {
	mu        sync.Mutex
	resultado ResultadoCobranca
	erro      error
	Cobrancas int
}

func (a *adquirenteFalso) Cobrar(context.Context, Cobranca) (ResultadoCobranca, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.Cobrancas++
	if a.erro != nil {
		return ResultadoCobranca{}, a.erro
	}
	return a.resultado, nil
}

type publicadorFalso struct {
	mu         sync.Mutex
	erro       error
	Publicados []Fato
}

func (p *publicadorFalso) Publicar(_ context.Context, f Fato) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.erro != nil {
		return p.erro
	}
	p.Publicados = append(p.Publicados, f)
	return nil
}

func (p *publicadorFalso) rotas() []string {
	var r []string
	for _, f := range p.Publicados {
		r = append(r, f.RoutingKey)
	}
	return r
}

var errInfra = errors.New("infra fora do ar")

// cenario monta o caso de uso com dobras e devolve tudo para inspeção.
func cenario(res ResultadoCobranca) (ProcessarPagamento, *repoFalso, *adquirenteFalso, *publicadorFalso) {
	repo, adq, pub := novoRepo(), &adquirenteFalso{resultado: res}, &publicadorFalso{}
	uc := ProcessarPagamento{
		Repo: repo, Adquirente: adq, Publicador: pub,
		Relogio: relogioFixo{instante}, IDs: idsFixos{"t-1"},
	}
	return uc, repo, adq, pub
}

func intencaoValida() Intencao {
	return Intencao{
		Evento: "RESERVA_CRIADA", ReservaID: "r-1", UsuarioID: "u-1",
		ValorTotal: "84.00", FormaPagamento: "PIX",
		ExpiraEm: prazoOK.Format(time.RFC3339),
	}
}

// adquirenteLento nunca responde por conta própria: só devolve quando o contexto
// do chamador expira. É o que permite testar o prazo de verdade, em vez de
// injetar o desfecho pronto.
type adquirenteLento struct{}

func (adquirenteLento) Cobrar(ctx context.Context, _ Cobranca) (ResultadoCobranca, error) {
	<-ctx.Done()
	return ResultadoCobranca{}, ctx.Err()
}
