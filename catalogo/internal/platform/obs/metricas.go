package obs

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Registro é o registry do processo. Rótulos são sempre de baixa cardinalidade:
// método, padrão da rota (nunca a URL com ids), status, fila, desfecho.
var Registro = prometheus.NewRegistry()

var (
	requisicoes = promauto.With(Registro).NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total", Help: "Total de requisições HTTP atendidas.",
	}, []string{"method", "route", "status"})

	duracaoHTTP = promauto.With(Registro).NewHistogramVec(prometheus.HistogramOpts{
		Name: "http_request_duration_seconds", Help: "Duração das requisições HTTP.",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "route"})

	emVoo = promauto.With(Registro).NewGauge(prometheus.GaugeOpts{
		Name: "http_requests_in_flight", Help: "Requisições HTTP em andamento.",
	})

	consumidas = promauto.With(Registro).NewCounterVec(prometheus.CounterOpts{
		Name: "amqp_messages_consumed_total", Help: "Mensagens consumidas, por fila e desfecho.",
	}, []string{"queue", "outcome"})

	duracaoConsumo = promauto.With(Registro).NewHistogramVec(prometheus.HistogramOpts{
		Name: "amqp_message_processing_seconds", Help: "Tempo de tratamento de uma mensagem.",
		Buckets: prometheus.DefBuckets,
	}, []string{"queue"})

	publicadas = promauto.With(Registro).NewCounterVec(prometheus.CounterOpts{
		Name: "amqp_messages_published_total", Help: "Mensagens publicadas, por routing key e resultado.",
	}, []string{"routing_key", "result"})
)

func init() {
	Registro.MustRegister(collectors.NewGoCollector(), collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
}

// RegistrarConsumo contabiliza uma mensagem tratada.
func RegistrarConsumo(fila, desfecho string, d time.Duration) {
	consumidas.WithLabelValues(fila, desfecho).Inc()
	duracaoConsumo.WithLabelValues(fila).Observe(d.Seconds())
}

// RegistrarPublicacao contabiliza uma publicação; erro nil conta como "ok".
func RegistrarPublicacao(routingKey string, err error) {
	resultado := "ok"
	if err != nil {
		resultado = "erro"
	}
	publicadas.WithLabelValues(routingKey, resultado).Inc()
}
