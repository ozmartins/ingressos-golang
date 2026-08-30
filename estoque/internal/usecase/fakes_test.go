package usecase

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/oseias/ingressos-golang/estoque/internal/domain/poltrona"
	"github.com/oseias/ingressos-golang/estoque/internal/domain/reserva"
	"github.com/oseias/ingressos-golang/estoque/internal/domain/shared"
)

// estoqueFalso implementa RepositorioReservas e RepositorioPoltronas em memória,
// com a MESMA semântica de guarda de estado do adaptador real: transição só a
// partir de PENDENTE, idempotência por (fila, message_id).
type estoqueFalso struct {
	mu          sync.Mutex
	poltronas   map[string]map[string]*poltrona.Poltrona // sessão → rótulo → poltrona
	reservas    map[string]*reserva.Reserva
	vinculos    map[string][]string // reserva → rótulos
	sessaoDe    map[string]string   // reserva → sessão
	processadas map[string]bool
	fatos       []FatoPendente

	erroForcado error
}

func novoEstoqueFalso() *estoqueFalso {
	return &estoqueFalso{
		poltronas:   map[string]map[string]*poltrona.Poltrona{},
		reservas:    map[string]*reserva.Reserva{},
		vinculos:    map[string][]string{},
		sessaoDe:    map[string]string{},
		processadas: map[string]bool{},
	}
}

func (e *estoqueFalso) provisionar(sessaoID string, rotulos ...string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.poltronas[sessaoID] == nil {
		e.poltronas[sessaoID] = map[string]*poltrona.Poltrona{}
	}
	for _, r := range rotulos {
		fileira, numero, err := poltrona.LerRotulo(r)
		if err != nil {
			panic(err)
		}
		p, err := poltrona.Nova(sessaoID, fileira, numero, poltrona.Normal)
		if err != nil {
			panic(err)
		}
		e.poltronas[sessaoID][p.Rotulo] = &p
	}
}

func (e *estoqueFalso) statusDe(sessaoID, rotulo string) poltrona.Status {
	e.mu.Lock()
	defer e.mu.Unlock()
	if p, ok := e.poltronas[sessaoID][rotulo]; ok {
		return p.Status
	}
	return ""
}

func (e *estoqueFalso) statusReserva(id string) reserva.Status {
	e.mu.Lock()
	defer e.mu.Unlock()
	if r, ok := e.reservas[id]; ok {
		return r.Status
	}
	return ""
}

func (e *estoqueFalso) Conceder(_ context.Context, sol reserva.Solicitacao, r reserva.Reserva, fato FatoPendente) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.erroForcado != nil {
		return e.erroForcado
	}

	daSessao, existe := e.poltronas[sol.SessaoID]
	if !existe || len(daSessao) == 0 {
		return fmt.Errorf("%w: sessão %s", shared.ErrSessaoNaoProvisionada, sol.SessaoID)
	}

	alvos := make([]*poltrona.Poltrona, 0, len(sol.Rotulos))
	for _, rotulo := range sol.Rotulos {
		p, ok := daSessao[rotulo]
		if !ok {
			return fmt.Errorf("%w: %s", shared.ErrPoltronaInexistente, rotulo)
		}
		if !p.PodeSerBloqueada() {
			return fmt.Errorf("%w: %s está %s", shared.ErrPoltronasIndisponiveis, rotulo, p.Status)
		}
		alvos = append(alvos, p)
	}

	for _, p := range alvos {
		atualizada, err := p.Transicionar(poltrona.Reservada)
		if err != nil {
			return err
		}
		*p = atualizada
	}

	copia := r
	e.reservas[r.ID] = &copia
	e.vinculos[r.ID] = sol.Rotulos
	e.sessaoDe[r.ID] = sol.SessaoID
	e.fatos = append(e.fatos, fato)
	return nil
}

func (e *estoqueFalso) aplicar(fila, messageID, reservaID string, agora time.Time,
	novoReserva reserva.Status, novoPoltrona poltrona.Status) (ResultadoTransicao, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.erroForcado != nil {
		return TransicaoIgnoradaInexistente, e.erroForcado
	}

	if messageID != "" {
		chave := fila + "|" + messageID
		if e.processadas[chave] {
			return TransicaoIgnoradaDuplicata, nil
		}
		e.processadas[chave] = true
	}

	r, existe := e.reservas[reservaID]
	if !existe {
		return TransicaoIgnoradaInexistente, nil
	}
	atualizada, err := r.Transicionar(novoReserva)
	if err != nil {
		return TransicaoIgnoradaEstadoFinal, nil
	}
	*r = atualizada

	sessao := e.sessaoDe[reservaID]
	for _, rotulo := range e.vinculos[reservaID] {
		if p, ok := e.poltronas[sessao][rotulo]; ok {
			nova, err := p.Transicionar(novoPoltrona)
			if err != nil {
				return TransicaoIgnoradaEstadoFinal, err
			}
			*p = nova
		}
	}
	_ = agora
	return TransicaoAplicada, nil
}

func (e *estoqueFalso) Confirmar(_ context.Context, fila, messageID, reservaID string, agora time.Time) (ResultadoTransicao, error) {
	return e.aplicar(fila, messageID, reservaID, agora, reserva.Confirmada, poltrona.Ocupada)
}

func (e *estoqueFalso) Cancelar(_ context.Context, fila, messageID, reservaID string, agora time.Time) (ResultadoTransicao, error) {
	return e.aplicar(fila, messageID, reservaID, agora, reserva.Cancelada, poltrona.Livre)
}

func (e *estoqueFalso) ExpirarVencidas(_ context.Context, agora time.Time, limite int) ([]string, error) {
	e.mu.Lock()
	ids := []string{}
	for id, r := range e.reservas {
		if r.Expirou(agora) && len(ids) < limite {
			ids = append(ids, id)
		}
	}
	e.mu.Unlock()

	afetadas := make([]string, 0, len(ids))
	for _, id := range ids {
		res, err := e.aplicar("", "", id, agora, reserva.Expirada, poltrona.Livre)
		if err != nil {
			return nil, err
		}
		if res == TransicaoAplicada {
			afetadas = append(afetadas, id)
		}
	}
	return afetadas, nil
}

func (e *estoqueFalso) ExpirarUma(_ context.Context, reservaID string, agora time.Time) (ResultadoTransicao, error) {
	e.mu.Lock()
	r, existe := e.reservas[reservaID]
	vencida := existe && r.Expirou(agora)
	e.mu.Unlock()

	if !existe {
		return TransicaoIgnoradaInexistente, nil
	}
	if !vencida {
		return TransicaoIgnoradaEstadoFinal, nil
	}
	return e.aplicar("", "", reservaID, agora, reserva.Expirada, poltrona.Livre)
}

func (e *estoqueFalso) MapaDaSessao(_ context.Context, sessaoID string) ([]poltrona.Poltrona, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.erroForcado != nil {
		return nil, e.erroForcado
	}
	var mapa []poltrona.Poltrona
	for _, p := range e.poltronas[sessaoID] {
		mapa = append(mapa, *p)
	}
	return mapa, nil
}

func (e *estoqueFalso) ProvisionarMatriz(_ context.Context, fila, messageID, sessaoID string, matriz []poltrona.Poltrona) (ResultadoTransicao, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.erroForcado != nil {
		return TransicaoIgnoradaInexistente, e.erroForcado
	}

	chave := fila + "|" + messageID
	if messageID != "" && e.processadas[chave] {
		return TransicaoIgnoradaDuplicata, nil
	}
	e.processadas[chave] = true

	if e.poltronas[sessaoID] == nil {
		e.poltronas[sessaoID] = map[string]*poltrona.Poltrona{}
	}
	criadas := 0
	for _, item := range matriz {
		if _, existe := e.poltronas[sessaoID][item.Rotulo]; existe {
			continue // nunca reinicia o estado de poltrona existente
		}
		copia := item
		e.poltronas[sessaoID][item.Rotulo] = &copia
		criadas++
	}
	if criadas == 0 {
		return TransicaoIgnoradaDuplicata, nil
	}
	return TransicaoAplicada, nil
}

// prazoFalso registra as marcações do índice de prazo.
type prazoFalso struct {
	mu        sync.Mutex
	marcados  map[string]time.Time
	liberados map[string]bool
	erro      error
}

func novoPrazoFalso() *prazoFalso {
	return &prazoFalso{marcados: map[string]time.Time{}, liberados: map[string]bool{}}
}

func (p *prazoFalso) Marcar(_ context.Context, reservaID string, expiraEm time.Time) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.erro != nil {
		return p.erro
	}
	p.marcados[reservaID] = expiraEm
	return nil
}

func (p *prazoFalso) Liberar(_ context.Context, reservaID string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.erro != nil {
		return p.erro
	}
	p.liberados[reservaID] = true
	return nil
}

// logFalso captura o que foi registrado, para verificar auditoria de divergência.
type logFalso struct {
	mu     sync.Mutex
	avisos []string
}

func (l *logFalso) Info(string, ...any) {}
func (l *logFalso) Warn(msg string, _ ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.avisos = append(l.avisos, msg)
}
func (l *logFalso) Error(string, ...any) {}

func (l *logFalso) avisou() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.avisos) > 0
}
