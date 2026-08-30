package shared

import "time"

// Relogio é a porta de tempo do núcleo. Existe para que expiração seja testável
// sem espera real: os testes injetam um relógio controlável.
type Relogio interface {
	Agora() time.Time
}

// RelogioReal lê o relógio do sistema, em UTC.
type RelogioReal struct{}

// Agora devolve o instante atual em UTC.
func (RelogioReal) Agora() time.Time { return time.Now().UTC() }

// RelogioFixo é o relógio controlável usado em teste.
type RelogioFixo struct{ instante time.Time }

// NovoRelogioFixo cria um relógio parado em t.
func NovoRelogioFixo(t time.Time) *RelogioFixo { return &RelogioFixo{instante: t.UTC()} }

// Agora devolve o instante configurado.
func (r *RelogioFixo) Agora() time.Time { return r.instante }

// Avancar move o relógio para frente.
func (r *RelogioFixo) Avancar(d time.Duration) { r.instante = r.instante.Add(d) }
