package obs

import "github.com/prometheus/client_golang/prometheus/promauto"
import "github.com/prometheus/client_golang/prometheus"

var (
	reservasExpiradas = promauto.With(Registro).NewCounter(prometheus.CounterOpts{
		Name: "estoque_reservas_expiradas_total", Help: "Reservas não pagas liberadas pela varredura de expiração.",
	})

	fallbackCatalogo = promauto.With(Registro).NewCounter(prometheus.CounterOpts{
		Name: "catalogo_fallback_total", Help: "Reservas atendidas com a sessão em cache porque o catálogo estava indisponível.",
	})

	estadoBreaker = promauto.With(Registro).NewGaugeVec(prometheus.GaugeOpts{
		Name: "circuit_breaker_state", Help: "Estado do disjuntor: 1 no estado atual, 0 nos demais.",
	}, []string{"breaker", "state"})
)

func init() {
	for _, e := range []string{"closed", "half-open", "open"} {
		v := 0.0
		if e == "closed" {
			v = 1
		}
		estadoBreaker.WithLabelValues("catalogo.sessao", e).Set(v)
	}
}

// ReservasExpiradas soma n reservas liberadas por expiração.
func ReservasExpiradas(n int) { reservasExpiradas.Add(float64(n)) }

// FallbackCatalogo conta uma consulta atendida pelo cache de sessões.
func FallbackCatalogo() { fallbackCatalogo.Inc() }

// EstadoDoBreaker registra a transição do disjuntor para o estado `atual`.
func EstadoDoBreaker(nome, atual string) {
	for _, e := range []string{"closed", "half-open", "open"} {
		v := 0.0
		if e == atual {
			v = 1
		}
		estadoBreaker.WithLabelValues(nome, e).Set(v)
	}
}
