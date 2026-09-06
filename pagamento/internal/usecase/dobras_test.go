package usecase

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"testing"
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

type repoFalso struct {
	mu            sync.Mutex
	porReserva    map[string]transacao.Transacao
	erroCriar     error
	erroFinalizar error
	erroMarcar    error
	Finalizacoes  int
	Marcacoes     int
	Escolhas      int
}

func novoRepo() *repoFalso {
	return &repoFalso{porReserva: map[string]transacao.Transacao{}}
}

func (r *repoFalso) semear(t transacao.Transacao) { r.porReserva[t.ReservaID] = t }

func (r *repoFalso) RegistrarEscolha(_ context.Context, t transacao.Transacao) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	atual, ok := r.porReserva[t.ReservaID]
	// A escolha só pega quando a linha ainda espera por ela: é o mesmo `WHERE
	// status = 'AGUARDANDO_FORMA'` do adaptador real.
	if !ok || atual.Status != transacao.AguardandoForma {
		return ErrJaFinalizada
	}
	r.porReserva[t.ReservaID] = t
	r.Escolhas++
	return nil
}

func (r *repoFalso) AnunciosPendentes(_ context.Context, limite int) ([]transacao.Transacao, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var lista []transacao.Transacao
	for _, t := range r.porReserva {
		if t.Status.Anunciavel() && !t.ResultadoAnunciado && len(lista) < limite {
			lista = append(lista, t)
		}
	}
	return lista, nil
}

func (r *repoFalso) AguardandoCobranca(_ context.Context, limite int) ([]transacao.Transacao, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var lista []transacao.Transacao
	for _, t := range r.porReserva {
		if t.Status == transacao.Processando && !t.CobrancaEmitida && len(lista) < limite {
			lista = append(lista, t)
		}
	}
	return lista, nil
}

func (r *repoFalso) CancelarEsperasVencidas(_ context.Context, agora time.Time, limite int) ([]transacao.Transacao, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var lista []transacao.Transacao
	for id, t := range r.porReserva {
		if t.Status != transacao.AguardandoForma || !transacao.Expirada(t.ExpiraEm, agora) || len(lista) >= limite {
			continue
		}
		t.Status = transacao.Cancelado
		t.MotivoFalha = transacao.MotivoReservaExpirada
		t.AtualizadoEm = agora
		r.porReserva[id] = t
		lista = append(lista, t)
	}
	return lista, nil
}

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
	atual.Status = t.Status
	atual.CodigoTransacaoGateway = t.CodigoTransacaoGateway
	atual.MotivoFalha = t.MotivoFalha
	atual.PagoEm = t.PagoEm
	atual.AtualizadoEm = t.AtualizadoEm
	r.porReserva[t.ReservaID] = atual
	r.Finalizacoes++
	return nil
}

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

func logDescartado() *slog.Logger { return slog.New(slog.DiscardHandler) }

func cenario(res ResultadoCobranca) (ProcessarPagamento, *repoFalso, *adquirenteFalso, *publicadorFalso) {
	repo, adq, pub := novoRepo(), &adquirenteFalso{resultado: res}, &publicadorFalso{}
	uc := ProcessarPagamento{
		Repo: repo, Adquirente: adq, Publicador: pub,
		Relogio: relogioFixo{instante}, IDs: idsFixos{"t-1"},
	}
	return uc, repo, adq, pub
}

// O fluxo inteiro, do anúncio da reserva até o desfecho da cobrança, como ele
// acontece em produção: o consumo registra a intenção, quem paga escolhe a
// forma, e o varredor cobra. Os testes de cobrança exercitam os três de uma vez
// porque o que eles verificam é o desfecho, não o caminho.
//
// Devolve o desfecho da cobrança — é o que os testes inspecionavam quando isso
// era uma chamada só.
func fluxoCompleto(t *testing.T, uc ProcessarPagamento, repo *repoFalso, i Intencao) (Desfecho, error) {
	t.Helper()
	ctx := context.Background()

	registrar := RegistrarIntencao{Repo: repo, Relogio: uc.Relogio, IDs: uc.IDs}
	if d, err := registrar.Executar(ctx, i); err != nil {
		return d, err
	}

	escolher := EscolherForma{Repo: repo, Relogio: uc.Relogio}
	tr, err := escolher.Executar(ctx, i.ReservaID, i.UsuarioID, transacao.PIX)

	// A reserva já vencida recusa a escolha. Quem cancela e anuncia é a
	// varredura, e é ela que o teste precisa exercitar para chegar ao desfecho.
	if errors.Is(err, transacao.ErrReservaExpirada) {
		varrer := VarrerCobrancas{Repo: repo, Cobranca: uc, Relogio: uc.Relogio, Log: logDescartado()}
		if err := varrer.Executar(ctx); err != nil {
			return Requeue, err
		}
		return Confirmar, nil
	}
	if err != nil {
		return Requeue, err
	}
	return uc.Cobrar(ctx, tr)
}

func intencaoValida() Intencao {
	return Intencao{
		Evento: "RESERVA_CRIADA", ReservaID: "r-1", UsuarioID: "u-1",
		ValorTotal: "84.00",
		ExpiraEm:   prazoOK.Format(time.RFC3339),
	}
}

type adquirenteLento struct{}

func (adquirenteLento) Cobrar(ctx context.Context, _ Cobranca) (ResultadoCobranca, error) {
	<-ctx.Done()
	return ResultadoCobranca{}, ctx.Err()
}
