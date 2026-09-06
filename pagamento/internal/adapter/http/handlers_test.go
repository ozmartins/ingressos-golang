package http

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
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

func (r repoStub) RegistrarEscolha(context.Context, transacao.Transacao) error { return nil }

// Guarda a transação para que a escolha seja observável, e devolve o mesmo erro
// em qualquer operação — é o que os casos de indisponibilidade precisam.
type repoEscolha struct {
	repoStub
	t   transacao.Transacao
	err error
}

func (r *repoEscolha) BuscarPorReserva(context.Context, string) (transacao.Transacao, error) {
	if r.err != nil {
		return transacao.Transacao{}, r.err
	}
	return r.t, nil
}

func (r *repoEscolha) RegistrarEscolha(_ context.Context, t transacao.Transacao) error {
	r.t = t
	return nil
}
func (r repoStub) AguardandoCobranca(context.Context, int) ([]transacao.Transacao, error) {
	return nil, nil
}
func (r repoStub) AnunciosPendentes(context.Context, int) ([]transacao.Transacao, error) {
	return nil, nil
}
func (r repoStub) CancelarEsperasVencidas(context.Context, time.Time, int) ([]transacao.Transacao, error) {
	return nil, nil
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
		Escolha:   usecase.EscolherForma{Repo: repo, Relogio: relogioReal{}},
		Auth:      NovoAutenticadorComChave(kf, issuer, audience),
		Prontidao: p,
		Log:       slog.New(slog.DiscardHandler),
	}
}

type relogioReal struct{}

func (relogioReal) Agora() time.Time { return time.Now().UTC() }

// Uma transação em qualquer estado posterior à escolha da forma — que é a
// maioria dos casos que a API responde, e o único em que todos os campos do
// contrato estão preenchidos. Para o estado que ainda espera a escolha, use
// `transacao.Nova` direto.
func transacaoDe(status transacao.Status, reservaID string) transacao.Transacao {
	agora := time.Now().UTC()
	tr := transacao.Nova("t-1", reservaID, dona, "84.00", agora.Add(10*time.Minute), agora)
	tr.Status = status
	tr.FormaPagamento = transacao.PIX
	return tr
}

func escolher(t *testing.T, api *API, reservaID, bearer, corpo string) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/pagamentos/reserva/"+reservaID,
		strings.NewReader(corpo))
	if bearer != "" {
		r.Header.Set("Authorization", "Bearer "+bearer)
	}
	w := httptest.NewRecorder()
	api.Rotas().ServeHTTP(w, r)
	return w
}

// Uma transação que ainda espera a escolha da forma — o único estado em que a
// escolha é aceita.
func aguardandoForma(reservaID string) transacao.Transacao {
	agora := time.Now().UTC()
	return transacao.Nova("t-1", reservaID, dona, "84.00", agora.Add(10*time.Minute), agora)
}

// O 202 é deliberado: a escolha foi aceita e a cobrança acontece fora da
// requisição, então quem paga não fica esperando o adquirente.
func TestEscolhaDaFormaAceita(t *testing.T) {
	reserva := uuid.NewString()
	api := apiCom(&repoEscolha{t: aguardandoForma(reserva)})
	w := escolher(t, api, reserva, token(t, dona, nil), `{"forma_pagamento":"PIX"}`)

	if w.Code != http.StatusAccepted {
		t.Fatalf("esperava 202, veio %d: %s", w.Code, w.Body)
	}
	var corpo map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &corpo); err != nil {
		t.Fatal(err)
	}
	if corpo["status"] != string(transacao.Processando) || corpo["forma_pagamento"] != "PIX" {
		t.Fatalf("corpo inesperado: %v", corpo)
	}
}

func TestEscolhaDaFormaRecusaCadaCategoriaDeErro(t *testing.T) {
	reserva := uuid.NewString()
	agora := time.Now().UTC()

	vencida := aguardandoForma(reserva)
	vencida.ExpiraEm = agora.Add(-time.Minute)

	jaEscolhida := transacaoDe(transacao.Processando, reserva)

	casos := map[string]struct {
		repo    usecase.Repositorio
		bearer  string
		corpo   string
		status  int
		codigo  string
		reserva string
	}{
		"sem token": {
			repo: &repoEscolha{t: aguardandoForma(reserva)}, bearer: "",
			corpo: `{"forma_pagamento":"PIX"}`, status: http.StatusUnauthorized,
			codigo: CodCredencialInvalida,
		},
		"reserva_id não é UUID": {
			repo: &repoEscolha{t: aguardandoForma(reserva)}, bearer: token(t, dona, nil),
			corpo: `{"forma_pagamento":"PIX"}`, status: http.StatusBadRequest,
			codigo: CodReservaIDInvalido, reserva: "nao-e-uuid",
		},
		"corpo não é JSON": {
			repo: &repoEscolha{t: aguardandoForma(reserva)}, bearer: token(t, dona, nil),
			corpo: `{`, status: http.StatusBadRequest, codigo: CodCorpoInvalido,
		},
		"forma desconhecida": {
			repo: &repoEscolha{t: aguardandoForma(reserva)}, bearer: token(t, dona, nil),
			corpo: `{"forma_pagamento":"BOLETO"}`, status: http.StatusBadRequest,
			codigo: CodFormaDesconhecida,
		},
		"forma ausente": {
			repo: &repoEscolha{t: aguardandoForma(reserva)}, bearer: token(t, dona, nil),
			corpo: `{}`, status: http.StatusBadRequest, codigo: CodFormaDesconhecida,
		},
		"reserva de outra pessoa": {
			repo: &repoEscolha{t: aguardandoForma(reserva)}, bearer: token(t, "outra-pessoa", nil),
			corpo: `{"forma_pagamento":"PIX"}`, status: http.StatusNotFound,
			codigo: CodNaoEncontrado,
		},
		"reserva inexistente": {
			repo: &repoEscolha{err: usecase.ErrNaoEncontrada}, bearer: token(t, dona, nil),
			corpo: `{"forma_pagamento":"PIX"}`, status: http.StatusNotFound,
			codigo: CodNaoEncontrado,
		},
		"forma já escolhida": {
			repo: &repoEscolha{t: jaEscolhida}, bearer: token(t, dona, nil),
			corpo: `{"forma_pagamento":"CARTAO_CREDITO"}`, status: http.StatusConflict,
			codigo: CodFormaJaEscolhida,
		},
		"prazo da reserva vencido": {
			repo: &repoEscolha{t: vencida}, bearer: token(t, dona, nil),
			corpo: `{"forma_pagamento":"PIX"}`, status: http.StatusConflict,
			codigo: CodReservaExpirada,
		},
		"armazenamento indisponível": {
			repo: &repoEscolha{err: errors.New("banco fora do ar")}, bearer: token(t, dona, nil),
			corpo: `{"forma_pagamento":"PIX"}`, status: http.StatusServiceUnavailable,
			codigo: CodIndisponivel,
		},
	}

	for nome, c := range casos {
		t.Run(nome, func(t *testing.T) {
			alvo := c.reserva
			if alvo == "" {
				alvo = reserva
			}
			w := escolher(t, apiCom(c.repo), alvo, c.bearer, c.corpo)
			if w.Code != c.status {
				t.Fatalf("esperava %d, veio %d: %s", c.status, w.Code, w.Body)
			}
			var e erroResposta
			if err := json.Unmarshal(w.Body.Bytes(), &e); err != nil {
				t.Fatal(err)
			}
			if e.Codigo != c.codigo {
				t.Fatalf("codigo = %q, esperado %q", e.Codigo, c.codigo)
			}
		})
	}
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
