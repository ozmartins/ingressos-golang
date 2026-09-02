package shared

import "time"

type Relogio interface {
	Agora() time.Time
}

type RelogioReal struct{}

func (RelogioReal) Agora() time.Time { return time.Now().UTC() }

type RelogioFixo struct{ instante time.Time }

func NovoRelogioFixo(t time.Time) *RelogioFixo { return &RelogioFixo{instante: t.UTC()} }

func (r *RelogioFixo) Agora() time.Time { return r.instante }

func (r *RelogioFixo) Avancar(d time.Duration) { r.instante = r.instante.Add(d) }
