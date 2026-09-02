package http

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/oseias/ingressos-golang/pagamento/internal/domain/transacao"
	"github.com/oseias/ingressos-golang/pagamento/internal/platform/health"
	"github.com/oseias/ingressos-golang/pagamento/internal/usecase"
)

const (
	issuer   = "https://keycloak.teste/realms/cinema"
	audience = "servico-pagamento"
	dona     = "11111111-1111-4111-8111-111111111111"
	terceiro = "22222222-2222-4222-8222-222222222222"
)

var segredo = []byte("chave-de-teste")

type repoStub struct {
	t   transacao.Transacao
	err error
}

func (r repoStub) CriarSeAusente(context.Context, transacao.Transacao) (bool, transacao.Transacao, error) {
	return false, transacao.Transacao{}, nil
}
func (r repoStub) BuscarPorReserva(context.Context, string) (transacao.Transacao, error) {
	if r.err != nil {
		return transacao.Transacao{}, r.err
	}
	return r.t, nil
}
func (r repoStub) Finalizar(context.Context, transacao.Transacao) error { return nil }
func (r repoStub) ReivindicarCobranca(context.Context, string, time.Time) (bool, error) {
	return true, nil
}
func (r repoStub) LiberarCobranca(context.Context, string, time.Time) error { return nil }
func (r repoStub) MarcarAnunciado(context.Context, string, time.Time) error { return nil }

func token(t *testing.T, sub string, ajustar func(jwt.MapClaims)) string {
	t.Helper()
	c := jwt.MapClaims{
		"sub": sub, "iss": issuer, "aud": audience,
		"exp": time.Now().Add(time.Hour).Unix(), "iat": time.Now().Unix(),
	}
	if ajustar != nil {
		ajustar(c)
	}
	s, err := jwt.NewWithClaims(jwt.SigningMethodHS256, c).SignedString(segredo)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func apiCom(repo usecase.Repositorio) *API {
	kf := func(*jwt.Token) (any, error) { return segredo, nil }
	p := health.NovaProntidao()
	p.Registrar("banco", func(context.Context) error { return nil })
	return &API{
		Consulta:  usecase.ConsultarPagamento{Repo: repo},
		Auth:      NovoAutenticadorComChave(kf, issuer, audience),
		Prontidao: p,
		Log:       slog.New(slog.DiscardHandler),
	}
}

func transacaoDe(status transacao.Status, reservaID string) transacao.Transacao {
	tr := transacao.Nova("t-1", reservaID, dona, "84.00", transacao.PIX, time.Now().UTC())
	tr.Status = status
	return tr
}

func chamar(t *testing.T, api *API, reservaID, bearer string) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/pagamentos/reserva/"+reservaID, nil)
	if bearer != "" {
		r.Header.Set("Authorization", "Bearer "+bearer)
	}
	w := httptest.NewRecorder()
	api.Rotas().ServeHTTP(w, r)
	return w
}

func TestConsultaDaDonaEmCadaEstado(t *testing.T) {
	estados := []transacao.Status{
		transacao.Processando, transacao.Pago, transacao.Recusado,
		transacao.Cancelado, transacao.PendenteVerificacao,
	}
	for _, s := range estados {
		t.Run(string(s), func(t *testing.T) {
			reserva := uuid.NewString()
			api := apiCom(repoStub{t: transacaoDe(s, reserva)})
			w := chamar(t, api, reserva, token(t, dona, nil))

			if w.Code != http.StatusOK {
				t.Fatalf("esperava 200, veio %d: %s", w.Code, w.Body)
			}
			var corpo map[string]any
			if err := json.Unmarshal(w.Body.Bytes(), &corpo); err != nil {
				t.Fatal(err)
			}
			if corpo["status"] != string(s) {
				t.Fatalf("estado errado: %v", corpo["status"])
			}
			for _, proibido := range []string{"motivo_falha", "codigo_transacao_gateway"} {
				if _, presente := corpo[proibido]; presente {
					t.Fatalf("%q não deve ser exposto na API", proibido)
				}
			}
		})
	}
}

func TestTerceiroEInexistenteSaoIndistinguiveis(t *testing.T) {
	estados := []transacao.Status{
		transacao.Processando, transacao.Pago, transacao.Recusado,
		transacao.Cancelado, transacao.PendenteVerificacao,
	}

	reservaAusente := uuid.NewString()
	apiAusente := apiCom(repoStub{err: usecase.ErrNaoEncontrada})
	wAusente := chamar(t, apiAusente, reservaAusente, token(t, terceiro, nil))
	if wAusente.Code != http.StatusNotFound {
		t.Fatalf("reserva inexistente devia dar 404, veio %d", wAusente.Code)
	}
	corpoAusente, _ := io.ReadAll(wAusente.Body)

	for _, s := range estados {
		t.Run(string(s), func(t *testing.T) {
			reserva := uuid.NewString()
			api := apiCom(repoStub{t: transacaoDe(s, reserva)})
			w := chamar(t, api, reserva, token(t, terceiro, nil))

			if w.Code != http.StatusNotFound {
				t.Fatalf("terceiro devia receber 404, veio %d", w.Code)
			}
			corpo, _ := io.ReadAll(w.Body)
			if string(corpo) != string(corpoAusente) {
				t.Fatalf("corpos diferentes revelam existência da reserva:\n terceiro: %s\ninexistente: %s",
					corpo, corpoAusente)
			}
		})
	}
}

func TestCredencialInvalida(t *testing.T) {
	reserva := uuid.NewString()
	api := apiCom(repoStub{t: transacaoDe(transacao.Pago, reserva)})

	casos := map[string]string{
		"sem token":  "",
		"token lixo": "nao-e-um-jwt",
		"assinatura errada": func() string {
			s, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
				"sub": dona, "iss": issuer, "aud": audience,
				"exp": time.Now().Add(time.Hour).Unix(),
			}).SignedString([]byte("outra-chave"))
			return s
		}(),
		"expirado": token(t, dona, func(c jwt.MapClaims) {
			c["exp"] = time.Now().Add(-time.Hour).Unix()
		}),
		"emissor errado": token(t, dona, func(c jwt.MapClaims) { c["iss"] = "https://intruso" }),
		"publico errado": token(t, dona, func(c jwt.MapClaims) { c["aud"] = "outro-servico" }),
		"sem sub":        token(t, dona, func(c jwt.MapClaims) { delete(c, "sub") }),
	}

	for nome, bearer := range casos {
		t.Run(nome, func(t *testing.T) {
			w := chamar(t, api, reserva, bearer)
			if w.Code != http.StatusUnauthorized {
				t.Fatalf("esperava 401, veio %d: %s", w.Code, w.Body)
			}
			var e erroResposta
			_ = json.Unmarshal(w.Body.Bytes(), &e)
			if e.Codigo != CodCredencialInvalida {
				t.Fatalf("código errado: %q", e.Codigo)
			}
		})
	}
}

func TestCredencialVerificadaAntesDeLerDados(t *testing.T) {
	api := apiCom(repoStub{err: usecase.ErrNaoEncontrada})
	w := chamar(t, api, "nem-e-uuid", "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("sem credencial, nem o formato do id deve ser avaliado; veio %d", w.Code)
	}
}

func TestReservaIDMalformado(t *testing.T) {
	api := apiCom(repoStub{t: transacaoDe(transacao.Pago, "x")})
	for _, id := range []string{"abc", "123", "nao-uuid-nenhum"} {
		w := chamar(t, api, id, token(t, dona, nil))
		if w.Code != http.StatusBadRequest {
			t.Fatalf("%q devia dar 400, veio %d", id, w.Code)
		}
		var e erroResposta
		_ = json.Unmarshal(w.Body.Bytes(), &e)
		if e.Codigo != CodReservaIDInvalido {
			t.Fatalf("código errado: %q", e.Codigo)
		}
	}
}

func TestFalhaDeArmazenamentoDa503(t *testing.T) {
	reserva := uuid.NewString()
	api := apiCom(repoStub{err: context.DeadlineExceeded})
	w := chamar(t, api, reserva, token(t, dona, nil))
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("esperava 503, veio %d", w.Code)
	}
	var e erroResposta
	_ = json.Unmarshal(w.Body.Bytes(), &e)
	if e.Codigo != CodIndisponivel {
		t.Fatalf("código errado: %q", e.Codigo)
	}
}

func TestSondasDeSaude(t *testing.T) {
	api := apiCom(repoStub{})
	for _, rota := range []string{"/api/v1/health/live", "/api/v1/health/ready"} {
		w := httptest.NewRecorder()
		api.Rotas().ServeHTTP(w, httptest.NewRequest(http.MethodGet, rota, nil))
		if w.Code != http.StatusOK {
			t.Fatalf("%s devia dar 200, veio %d", rota, w.Code)
		}
	}
}

func TestProntidaoFalhaQuandoDependenciaCai(t *testing.T) {
	api := apiCom(repoStub{})
	api.Prontidao = health.NovaProntidao()
	api.Prontidao.Registrar("banco", func(context.Context) error { return context.DeadlineExceeded })

	w := httptest.NewRecorder()
	api.Rotas().ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/health/ready", nil))
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("prontidão devia falhar, veio %d", w.Code)
	}
}

func chamarURL(t *testing.T, api *API, url, bearer string) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest(http.MethodGet, url, nil)
	if bearer != "" {
		r.Header.Set("Authorization", "Bearer "+bearer)
	}
	w := httptest.NewRecorder()
	api.Rotas().ServeHTTP(w, r)
	return w
}
