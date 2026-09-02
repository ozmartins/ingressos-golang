package usecase

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/oseias/ingressos-golang/notificacao/internal/domain/aviso"
	"github.com/oseias/ingressos-golang/notificacao/internal/domain/ingresso"
)

var instanteFixo = time.Date(2026, 8, 29, 21, 35, 12, 0, time.UTC)

type relogioFixo struct{ t time.Time }

func (r relogioFixo) Agora() time.Time {
	if r.t.IsZero() {
		return instanteFixo
	}
	return r.t
}

type idsSequenciais struct {
	mu sync.Mutex
	n  int
}

func (g *idsSequenciais) Novo() string {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.n++
	return canonico(g.n)
}

func canonico(n int) string {
	hex := "0123456789abcdef"
	b := []byte("00000000-0000-4000-8000-000000000000")
	b[len(b)-1] = hex[n%16]
	b[len(b)-2] = hex[(n/16)%16]
	return string(b)
}

type assinadorFalso struct{}

func (assinadorFalso) Gerar(id string) string { return "CIN1." + id + ".assinatura" }
func (assinadorFalso) Verificar(c string) (string, error) {
	if len(c) < 6 || c[:5] != "CIN1." {
		return "", errors.New("inválido")
	}
	resto := c[5:]
	for i := 0; i < len(resto); i++ {
		if resto[i] == '.' {
			if resto[i+1:] != "assinatura" {
				return "", errors.New("inválido")
			}
			return resto[:i], nil
		}
	}
	return "", errors.New("inválido")
}

var errBanco = errors.New("banco indisponível")

type ingressosFalsos struct {
	mu           sync.Mutex
	porReserva   map[string]ingresso.Ingresso
	porID        map[string]ingresso.Ingresso
	falharCriar  bool
	chamouBuscar bool
}

func novosIngressos() *ingressosFalsos {
	return &ingressosFalsos{
		porReserva: map[string]ingresso.Ingresso{},
		porID:      map[string]ingresso.Ingresso{},
	}
}

func (r *ingressosFalsos) CriarSeAusente(_ context.Context, i ingresso.Ingresso) (bool, ingresso.Ingresso, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.falharCriar {
		return false, ingresso.Ingresso{}, errBanco
	}
	if existente, ok := r.porReserva[i.ReservaID]; ok {
		return false, existente, nil
	}
	r.porReserva[i.ReservaID] = i
	r.porID[i.ID] = i
	return true, i, nil
}

func (r *ingressosFalsos) Utilizar(_ context.Context, id string, agora time.Time) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	i, ok := r.porID[id]
	if !ok || i.Status != ingresso.Valido {
		return false, nil
	}
	u, err := i.Utilizar(agora)
	if err != nil {
		return false, nil
	}
	r.porID[id] = u
	r.porReserva[u.ReservaID] = u
	return true, nil
}

func (r *ingressosFalsos) BuscarPorID(_ context.Context, id string) (ingresso.Ingresso, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.chamouBuscar = true
	i, ok := r.porID[id]
	if !ok {
		return ingresso.Ingresso{}, ErrNaoEncontrado
	}
	return i, nil
}

func (r *ingressosFalsos) ListarPorUsuario(_ context.Context, usuarioID string, filtro ingresso.Status) ([]ingresso.Ingresso, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []ingresso.Ingresso
	for _, i := range r.porID {
		if i.UsuarioID != usuarioID {
			continue
		}
		if filtro != "" && i.Status != filtro {
			continue
		}
		out = append(out, i)
	}
	return out, nil
}

func (r *ingressosFalsos) semear(i ingresso.Ingresso) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.porID[i.ID] = i
	r.porReserva[i.ReservaID] = i
}

type avisosFalsos struct {
	mu         sync.Mutex
	registros  []aviso.Registro
	falharGrav bool
}

func (a *avisosFalsos) Registrar(_ context.Context, r aviso.Registro) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.falharGrav {
		return errBanco
	}
	a.registros = append(a.registros, r)
	return nil
}

func (a *avisosFalsos) todos() []aviso.Registro {
	a.mu.Lock()
	defer a.mu.Unlock()
	return append([]aviso.Registro(nil), a.registros...)
}

var errCanal = errors.New("servidor de e-mail recusou a conexão")

type notificadorFalso struct {
	mu      sync.Mutex
	falhar  bool
	chamado int
}

func (n *notificadorFalso) Canal() aviso.Canal { return aviso.Email }

func (n *notificadorFalso) Avisar(context.Context, ingresso.Ingresso) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.chamado++
	if n.falhar {
		return errCanal
	}
	return nil
}

func (n *notificadorFalso) vezes() int {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.chamado
}
