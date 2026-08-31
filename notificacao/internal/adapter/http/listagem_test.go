package http

import (
	"bytes"
	"net/http"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/oseias/ingressos-golang/notificacao/internal/domain/ingresso"
)

// Uma asserção por linha de contracts/erros.md §2.

const rota = "/api/v1/ingressos/meus-ingressos"

func comToken(t *testing.T, a *ambiente, sub string) map[string]string {
	t.Helper()
	return map[string]string{"Authorization": "Bearer " + a.token(t, sub)}
}

func semearTres(t *testing.T, a *ambiente) {
	t.Helper()
	a.repo.semear(t, "ing-antigo", usuario1, ingresso.Valido, instanteFixo)
	a.repo.semear(t, "ing-meio", usuario1, ingresso.Utilizado, instanteFixo.Add(time.Minute))
	a.repo.semear(t, "ing-novo", usuario1, ingresso.Cancelado, instanteFixo.Add(2*time.Minute))
	a.repo.semear(t, "ing-de-outra", usuario2, ingresso.Valido, instanteFixo.Add(3*time.Minute))
}

func TestListagemDevolveTodosOsEstadosOrdenados(t *testing.T) {
	a := montarAmbiente(t)
	semearTres(t, a)

	res, corpo := a.get(t, rota, comToken(t, a, usuario1))
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, queria 200 (corpo: %s)", res.StatusCode, corpo)
	}
	lista := decodificarLista(t, corpo)
	if len(lista) != 3 {
		t.Fatalf("%d ingressos, queria 3", len(lista))
	}
	// FR-023: do mais recente para o mais antigo.
	querida := []string{"ing-novo", "ing-meio", "ing-antigo"}
	for k, esperado := range querida {
		if lista[k]["ingresso_id"] != esperado {
			t.Errorf("posição %d = %v, queria %s", k, lista[k]["ingresso_id"], esperado)
		}
	}
	// Campos do contrato.
	for _, campo := range []string{"ingresso_id", "reserva_id", "codigo_qr", "status", "criado_em"} {
		if _, ok := lista[0][campo]; !ok {
			t.Errorf("campo %q ausente na resposta", campo)
		}
	}
}

// FR-014: nenhum parâmetro alcança ingresso de terceiro.
func TestListagemNaoRevelaIngressoDeTerceiro(t *testing.T) {
	a := montarAmbiente(t)
	semearTres(t, a)

	_, corpo := a.get(t, rota, comToken(t, a, usuario1))
	if bytes.Contains(corpo, []byte("ing-de-outra")) {
		t.Error("ingresso de outra pessoa apareceu na listagem (FR-014)")
	}

	// E a pessoa dona daquele ingresso vê só o dela.
	_, corpo2 := a.get(t, rota, comToken(t, a, usuario2))
	lista := decodificarLista(t, corpo2)
	if len(lista) != 1 || lista[0]["ingresso_id"] != "ing-de-outra" {
		t.Errorf("a outra pessoa recebeu %v", lista)
	}
}

func TestListagemFiltraPorEstado(t *testing.T) {
	a := montarAmbiente(t)
	semearTres(t, a)

	casos := map[string]string{"VALIDO": "ing-antigo", "UTILIZADO": "ing-meio", "CANCELADO": "ing-novo"}
	for filtro, esperado := range casos {
		t.Run(filtro, func(t *testing.T) {
			res, corpo := a.get(t, rota+"?status="+filtro, comToken(t, a, usuario1))
			if res.StatusCode != http.StatusOK {
				t.Fatalf("status = %d, queria 200", res.StatusCode)
			}
			lista := decodificarLista(t, corpo)
			if len(lista) != 1 || lista[0]["ingresso_id"] != esperado {
				t.Errorf("filtro %s devolveu %v", filtro, lista)
			}
		})
	}
}

// FR-024: estado desconhecido é 400, nunca a lista inteira.
func TestFiltroDesconhecidoEQuatrocentos(t *testing.T) {
	a := montarAmbiente(t)
	semearTres(t, a)

	for _, filtro := range []string{"INVENTADO", "valido", "VALIDO,UTILIZADO", "1"} {
		t.Run(filtro, func(t *testing.T) {
			res, corpo := a.get(t, rota+"?status="+filtro, comToken(t, a, usuario1))
			if res.StatusCode != http.StatusBadRequest {
				t.Errorf("status = %d, queria 400 (corpo: %s)", res.StatusCode, corpo)
			}
			if bytes.Contains(corpo, []byte("ing-antigo")) {
				t.Error("o filtro inválido devolveu a lista em vez de recusar")
			}
		})
	}
}

func TestPessoaSemIngressosRecebeListaVaziaENao404(t *testing.T) {
	a := montarAmbiente(t)
	res, corpo := a.get(t, rota, comToken(t, a, usuario1))
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, queria 200", res.StatusCode)
	}
	if string(bytes.TrimSpace(corpo)) != "[]" {
		t.Errorf("corpo = %s, queria []", corpo)
	}
}

// FR-015: toda recusa de credencial é o mesmo 401.
func TestListagemRecusaCredencial(t *testing.T) {
	a := montarAmbiente(t)
	semearTres(t, a)

	casos := map[string]map[string]string{
		"sem authorization":          {},
		"esquema errado":             {"Authorization": "Basic abc"},
		"token lixo":                 {"Authorization": "Bearer nao-e-um-token"},
		"chave de portaria no lugar": {"X-API-Key": chaveAPI},
		"expirado": {"Authorization": "Bearer " + a.token(t, usuario1, func(c jwt.MapClaims) {
			c["exp"] = time.Now().Add(-time.Hour).Unix()
		})},
		"emissor errado": {"Authorization": "Bearer " + a.token(t, usuario1, func(c jwt.MapClaims) {
			c["iss"] = "http://impostor"
		})},
		"publico errado": {"Authorization": "Bearer " + a.token(t, usuario1, func(c jwt.MapClaims) {
			c["aud"] = "outro-app"
		})},
		"sem sub": {"Authorization": "Bearer " + a.token(t, usuario1, func(c jwt.MapClaims) {
			delete(c, "sub")
		})},
	}
	var refCorpo []byte
	for nome, cab := range casos {
		res, corpo := a.get(t, rota, cab)
		if res.StatusCode != http.StatusUnauthorized {
			t.Errorf("%s: status = %d, queria 401 (corpo: %s)", nome, res.StatusCode, corpo)
			continue
		}
		if bytes.Contains(corpo, []byte("ing-antigo")) {
			t.Errorf("%s: a recusa vazou ingresso", nome)
		}
		if refCorpo == nil {
			refCorpo = corpo
		} else if !bytes.Equal(corpo, refCorpo) {
			t.Errorf("%s: corpo de 401 difere dos demais — vaza qual é o motivo", nome)
		}
	}
}
