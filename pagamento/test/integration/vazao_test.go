//go:build integration

package integration

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	adapthttp "github.com/oseias/ingressos-golang/pagamento/internal/adapter/http"
	"github.com/oseias/ingressos-golang/pagamento/internal/platform/health"
	"github.com/oseias/ingressos-golang/pagamento/internal/usecase"
)

const (
	quantidade = 1000
	teto       = 10
)

func TestRajadaRespeitaTetoEMantemConsultasRapidas(t *testing.T) {
	a := subirAmbiente(t)
	adq := novoAdquirente(usecase.ResultadoCobranca{Desfecho: usecase.Aprovada, Codigo: "gw"})
	adq.demora = 5 * time.Millisecond
	consumidor, parar := a.consumidorDe(t, adq, teto)
	defer parar()

	reservas := make([]string, quantidade)
	inicio := time.Now()
	for i := range reservas {
		reservas[i] = uuid.NewString()
		a.publicarIntencao(t, intencao(reservas[i], "84.00", "PIX", 30*time.Minute))
	}

	latencias := medirConsultasDuranteOPico(t, a, reservas)

	limite := time.Now().Add(3 * time.Minute)
	var processadas int
	for time.Now().Before(limite) {
		if err := a.Pool.QueryRow(context.Background(),
			"SELECT count(*) FROM transacoes_pagamento WHERE status='PAGO'").Scan(&processadas); err != nil {
			t.Fatal(err)
		}
		if processadas >= quantidade {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
	decorrido := time.Since(inicio)

	if processadas != quantidade {
		t.Fatalf("SC-004 violado: %d de %d intenções processadas em %s", processadas, quantidade, decorrido)
	}
	if n := consumidor.EmAndamento.Maximo(); n > teto {
		t.Fatalf("FR-019 violado: %d cobranças simultâneas, teto é %d", n, teto)
	}
	t.Logf("rajada: %d intenções em %s, pico de concorrência %d",
		processadas, decorrido.Round(time.Millisecond), consumidor.EmAndamento.Maximo())

	sort.Slice(latencias, func(i, j int) bool { return latencias[i] < latencias[j] })
	p95 := latencias[int(float64(len(latencias))*0.95)]
	t.Logf("consultas durante o pico: n=%d p95=%s", len(latencias), p95.Round(time.Millisecond))
	if p95 > time.Second {
		t.Fatalf("SC-005 violado: p95 das consultas foi %s durante o pico", p95)
	}
}

func medirConsultasDuranteOPico(t *testing.T, a *ambiente, reservas []string) []time.Duration {
	t.Helper()

	segredo := []byte("chave-de-teste")
	const iss, aud = "https://keycloak.teste/realms/cinema", "servico-pagamento"
	kf := func(*jwt.Token) (any, error) { return segredo, nil }

	prontidao := health.NovaProntidao()
	prontidao.Registrar("banco", a.Repo.Ping)
	api := &adapthttp.API{
		Consulta:  usecase.ConsultarPagamento{Repo: a.Repo},
		Auth:      adapthttp.NovoAutenticadorComChave(kf, iss, aud),
		Prontidao: prontidao,
		Log:       slog.New(slog.DiscardHandler),
	}
	rotas := api.Rotas()

	var (
		mu        sync.Mutex
		latencias []time.Duration
		wg        sync.WaitGroup
	)
	fim := time.Now().Add(20 * time.Second)

	for c := 0; c < 8; c++ {
		wg.Add(1)
		go func(semente int) {
			defer wg.Done()
			i := semente
			for time.Now().Before(fim) {
				reserva := reservas[i%len(reservas)]
				i += 7

				sub := uuid.NewString()
				if tr, err := a.Repo.BuscarPorReserva(context.Background(), reserva); err == nil {
					sub = tr.UsuarioID
				}
				tok, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
					"sub": sub, "iss": iss, "aud": aud,
					"exp": time.Now().Add(time.Hour).Unix(),
				}).SignedString(segredo)
				if err != nil {
					return
				}

				req := httptest.NewRequest(http.MethodGet, "/api/v1/pagamentos/reserva/"+reserva, nil)
				req.Header.Set("Authorization", "Bearer "+tok)
				w := httptest.NewRecorder()

				antes := time.Now()
				rotas.ServeHTTP(w, req)
				d := time.Since(antes)

				if w.Code != http.StatusOK && w.Code != http.StatusNotFound {
					t.Errorf("consulta devolveu %d durante o pico: %s", w.Code, w.Body)
					return
				}
				mu.Lock()
				latencias = append(latencias, d)
				mu.Unlock()
			}
		}(c)
	}
	wg.Wait()

	if len(latencias) == 0 {
		t.Fatal("nenhuma consulta foi medida durante o pico")
	}
	return latencias
}
