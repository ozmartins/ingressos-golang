package http

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/oseias/ingressos-golang/catalogo/internal/adapter/identidade"
)

const idDeTeste = "11111111-2222-3333-4444-555555555555"

type verificadorFalso struct{ aceita bool }

func (v verificadorFalso) Verificar(context.Context, string) (identidade.Identidade, error) {
	if !v.aceita {
		return identidade.Identidade{}, context.Canceled
	}
	return identidade.Identidade{UsuarioID: "u1"}, nil
}

func tabelaFalsa(visto *[]string) []Rota {
	falsa := Rotas()
	for i := range falsa {
		rota := falsa[i]
		falsa[i].handler = simples(func(Dependencias) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				*visto = append(*visto, rota.Metodo+" "+rota.Caminho+" id="+r.PathValue("id"))
				w.WriteHeader(http.StatusTeapot)
			}
		})
	}
	return falsa
}

func caminhoConcreto(caminho string) string {
	return strings.ReplaceAll(caminho, "{id}", idDeTeste)
}

func TestCadaRotaDaTabelaChegaAoSeuHandlerComOId(t *testing.T) {
	for _, rota := range Rotas() {
		t.Run(rota.Metodo+" "+rota.Caminho, func(t *testing.T) {
			var visto []string
			router := novoRouter(tabelaFalsa(&visto), Dependencias{Verificador: verificadorFalso{aceita: true}})

			req := httptest.NewRequest(rota.Metodo, caminhoConcreto(rota.Caminho), nil)
			req.Header.Set("Authorization", "Bearer token")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != http.StatusTeapot {
				t.Fatalf("status = %d, o handler da rota não foi atingido", w.Code)
			}
			esperado := rota.Metodo + " " + rota.Caminho + " id="
			if strings.Contains(rota.Caminho, "{id}") {
				esperado += idDeTeste
			}
			if len(visto) != 1 || visto[0] != esperado {
				t.Fatalf("visto = %v, esperado [%s]", visto, esperado)
			}
		})
	}
}

func TestRotaProtegidaExigeCredencialEPublicaNao(t *testing.T) {
	for _, rota := range Rotas() {
		t.Run(rota.Metodo+" "+rota.Caminho, func(t *testing.T) {
			var visto []string
			router := novoRouter(tabelaFalsa(&visto), Dependencias{Verificador: verificadorFalso{aceita: false}})

			w := httptest.NewRecorder()
			router.ServeHTTP(w, httptest.NewRequest(rota.Metodo, caminhoConcreto(rota.Caminho), nil))

			if rota.Protegida {
				if w.Code != http.StatusUnauthorized || len(visto) != 0 {
					t.Fatalf("protegida sem credencial: status %d, chamadas %v; esperado 401 sem chamar o handler", w.Code, visto)
				}
				return
			}
			if w.Code != http.StatusTeapot {
				t.Fatalf("pública sem credencial: status %d, esperado o handler atingido", w.Code)
			}
		})
	}
}

func TestLogRegistraARotaComMetodoESemOIdentificador(t *testing.T) {
	var saida bytes.Buffer
	anterior := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&saida, nil)))
	t.Cleanup(func() { slog.SetDefault(anterior) })

	var visto []string
	router := novoRouter(tabelaFalsa(&visto), Dependencias{Verificador: verificadorFalso{aceita: true}})
	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/v1/filmes/"+idDeTeste, nil))

	log := saida.String()
	if !strings.Contains(log, `rota="GET /api/v1/filmes/{id}"`) {
		t.Fatalf("o log deveria ter rota=\"GET /api/v1/filmes/{id}\"; foi:\n%s", log)
	}
	if strings.Contains(log, idDeTeste) {
		t.Fatalf("o identificador vazou para o rótulo da rota:\n%s", log)
	}
}

func TestBordasDeRoteamento(t *testing.T) {
	var visto []string
	router := novoRouter(tabelaFalsa(&visto), Dependencias{Verificador: verificadorFalso{aceita: true}})

	do := func(metodo, caminho string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(metodo, caminho, nil)
		req.Header.Set("Authorization", "Bearer token")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		return w
	}

	t.Run("caminho inexistente", func(t *testing.T) {
		w := do(http.MethodGet, "/nada")
		if w.Code != http.StatusNotFound || w.Body.String() != "404 page not found\n" {
			t.Fatalf("status %d corpo %q", w.Code, w.Body.String())
		}
	})
	t.Run("método não permitido", func(t *testing.T) {
		w := do(http.MethodPost, "/api/v1/filmes/"+idDeTeste)
		if w.Code != http.StatusMethodNotAllowed {
			t.Fatalf("status %d, esperado 405", w.Code)
		}
		if got := w.Header().Get("Allow"); got != "DELETE, GET, HEAD, PUT" {
			t.Fatalf("Allow = %q", got)
		}
		if w.Body.String() != "Method Not Allowed\n" {
			t.Fatalf("corpo = %q", w.Body.String())
		}
	})
	t.Run("HEAD cai na rota GET", func(t *testing.T) {
		visto = nil
		w := do(http.MethodHead, "/api/v1/filmes")
		if w.Code != http.StatusTeapot || len(visto) != 1 {
			t.Fatalf("status %d, chamadas %v; esperado o handler do GET", w.Code, visto)
		}
	})
	t.Run("documentação com e sem barra final", func(t *testing.T) {
		for _, c := range []string{"/docs", "/docs/", "/openapi.yaml"} {
			if w := do(http.MethodGet, c); w.Code != http.StatusTeapot {
				t.Errorf("%s: status %d, esperado o handler atingido", c, w.Code)
			}
		}
	})
}

func TestMiddlewaresEnvolvemTambemAsRequisicoesSemRota(t *testing.T) {
	var saida bytes.Buffer
	anterior := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&saida, nil)))
	t.Cleanup(func() { slog.SetDefault(anterior) })

	var visto []string
	router := novoRouter(tabelaFalsa(&visto), Dependencias{Verificador: verificadorFalso{aceita: true}})
	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/nada", nil))
	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/api/v1/filmes/"+idDeTeste, nil))

	log := saida.String()
	for _, esperado := range []string{"status=404", "status=405"} {
		if !strings.Contains(log, esperado) {
			t.Errorf("o log não registrou %s:\n%s", esperado, log)
		}
	}
}

func TestIdComBarraCodificadaDaAMesmaRecusaQueIdInvalido(t *testing.T) {
	router := NovoRouter(Dependencias{})

	pedir := func(caminho string) (int, string) {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, caminho, nil))
		return w.Code, w.Body.String()
	}

	codigoSimples, corpoSimples := pedir("/api/v1/filmes/abc")
	if codigoSimples != http.StatusBadRequest {
		t.Fatalf("id inválido simples: status %d, esperado 400", codigoSimples)
	}
	codigo, corpo := pedir("/api/v1/filmes/x%2Fy")
	if codigo != codigoSimples {
		t.Fatalf("id com %%2F: status %d, esperado %d (corpo %s; referência %s)", codigo, codigoSimples, corpo, corpoSimples)
	}
}

func TestCaminhoNaoCanonicoResponde404(t *testing.T) {
	router := NovoRouter(Dependencias{})
	for _, caminho := range []string{"/api/v1//filmes", "/api/v1/filmes/../cinemas"} {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, caminho, nil))
		if w.Code != http.StatusNotFound {
			t.Errorf("%s: status %d, esperado 404", caminho, w.Code)
		}
	}
}
