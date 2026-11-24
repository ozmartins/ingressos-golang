// Package ratelimit limita a taxa de requisições por chave (usuário ou IP)
// com um balde de fichas (golang.org/x/time/rate) por chave.
//
// É deliberadamente simples e em memória: protege cada réplica contra abuso e
// força bruta, mas não é um limite global entre réplicas.
package ratelimit

import (
	"net"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

const (
	ociosoAte   = 5 * time.Minute // entrada sem uso por esse tempo é descartada
	limpezaCada = time.Minute
)

type entrada struct {
	lim   *rate.Limiter
	visto time.Time
}

// Limitador mantém um balde por chave. O zero value não é usável: use Novo.
type Limitador struct {
	rps   rate.Limit
	burst int

	mu       sync.Mutex
	entradas map[string]*entrada
	limpoEm  time.Time
	agora    func() time.Time
}

// Novo cria um limitador que repõe rps fichas por segundo e acumula até burst.
func Novo(rps float64, burst int) *Limitador {
	return &Limitador{
		rps: rate.Limit(rps), burst: burst,
		entradas: map[string]*entrada{}, agora: time.Now,
	}
}

// Permitir consome uma ficha da chave e diz se a requisição pode passar.
// Um *Limitador nil não limita nada (limite desligado).
func (l *Limitador) Permitir(chave string) bool {
	if l == nil {
		return true
	}
	agora := l.agora()

	l.mu.Lock()
	defer l.mu.Unlock()

	if agora.Sub(l.limpoEm) >= limpezaCada {
		for k, e := range l.entradas {
			if agora.Sub(e.visto) > ociosoAte {
				delete(l.entradas, k)
			}
		}
		l.limpoEm = agora
	}

	e, ok := l.entradas[chave]
	if !ok {
		e = &entrada{lim: rate.NewLimiter(l.rps, l.burst)}
		l.entradas[chave] = e
	}
	e.visto = agora
	return e.lim.AllowN(agora, 1)
}

// IPDe devolve o IP de origem da conexão. Não confia em X-Forwarded-For, que
// o cliente pode forjar para fugir do limite.
func IPDe(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
