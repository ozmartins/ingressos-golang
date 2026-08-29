package contract

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/oseias/ingressos-golang/catalogo/internal/domain/catalogo"
	"github.com/oseias/ingressos-golang/catalogo/internal/domain/reserva"
	"github.com/oseias/ingressos-golang/catalogo/internal/domain/shared"
)

const sessaoID = "f781a9b2-11e2-4f81-a901-8890bc123456"

func sessaoReservavel() catalogo.Sessao {
	return catalogo.Sessao{
		ID:             sessaoID,
		Status:         catalogo.SessaoAgendada,
		DataHoraInicio: agora().Add(10 * time.Hour),
	}
}

func reservar(t *testing.T, amb *ambiente, token, sessao string, corpo any) (*http.Response, []byte) {
	t.Helper()
	b, _ := json.Marshal(corpo)
	req, err := http.NewRequest(http.MethodPost, amb.servidor.URL+"/api/v1/sessoes/"+sessao+"/reservar", bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := amb.servidor.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	dados, _ := io.ReadAll(resp.Body)
	return resp, dados
}

// US3, cenário 1.
func TestReservarConfirmada(t *testing.T) {
	expira := agora().Add(10 * time.Minute)
	amb := montar(t, func(a *ambiente) {
		a.sessoes.sessao = sessaoReservavel()
		a.estoque.resultado = reserva.ResultadoReserva{ReservaID: "9982a1b3-44c1-4221-a123-902183120192", ExpiraEm: expira}
	})

	resp, corpo := reservar(t, amb, "token-bom", sessaoID, map[string]any{"poltronas_ids": []string{"A1", "A2"}})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("esperava 201, obteve %d (%s)", resp.StatusCode, corpo)
	}

	var confirmada struct {
		ReservaID string `json:"reserva_id"`
		ExpiraEm  string `json:"expira_em"`
	}
	if err := json.Unmarshal(corpo, &confirmada); err != nil {
		t.Fatalf("corpo inválido: %v", err)
	}
	if confirmada.ReservaID == "" || confirmada.ExpiraEm == "" {
		t.Fatalf("201 sem os dados obrigatórios: %+v", confirmada)
	}
	if _, err := time.Parse(time.RFC3339, confirmada.ExpiraEm); err != nil {
		t.Errorf("expira_em não é RFC3339: %s", confirmada.ExpiraEm)
	}
}

// US3, cenário 2.
func TestReservarPoltronaOcupadaDevolve409(t *testing.T) {
	amb := montar(t, func(a *ambiente) {
		a.sessoes.sessao = sessaoReservavel()
		a.estoque.erro = shared.ErrPoltronasIndisponiveis
	})
	resp, corpo := reservar(t, amb, "token-bom", sessaoID, map[string]any{"poltronas_ids": []string{"B1"}})

	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("esperava 409, obteve %d", resp.StatusCode)
	}
	p := decodificarProblem(t, resp, corpo)
	if p.Type != "https://cinema.example/errors/poltronas-indisponiveis" {
		t.Fatalf("type inesperado: %s", p.Type)
	}
}

// US3, cenário 3 — e a prova de que o estoque não é contatado.
func TestReservarSemCredencialNaoContataOEstoque(t *testing.T) {
	for _, token := range []string{"", "token-ruim"} {
		amb := montar(t, func(a *ambiente) { a.sessoes.sessao = sessaoReservavel() })
		resp, corpo := reservar(t, amb, token, sessaoID, map[string]any{"poltronas_ids": []string{"A1"}})

		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("token %q: esperava 401, obteve %d", token, resp.StatusCode)
		}
		p := decodificarProblem(t, resp, corpo)
		if p.Type != "https://cinema.example/errors/nao-autenticado" {
			t.Errorf("type inesperado: %s", p.Type)
		}
		if amb.estoque.chamadas != 0 {
			t.Errorf("token %q: o estoque foi contatado %d vez(es)", token, amb.estoque.chamadas)
		}
	}
}

// US3, cenário 4 — indisponibilidade temporária, sem vazar o motivo.
func TestReservarEstoqueIndisponivelDevolve503(t *testing.T) {
	amb := montar(t, func(a *ambiente) {
		a.sessoes.sessao = sessaoReservavel()
		a.estoque.erro = shared.ErrEstoqueIndisponivel
	})
	resp, corpo := reservar(t, amb, "token-bom", sessaoID, map[string]any{"poltronas_ids": []string{"A1"}})

	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("esperava 503, obteve %d", resp.StatusCode)
	}
	p := decodificarProblem(t, resp, corpo)
	if p.Type != "https://cinema.example/errors/estoque-indisponivel" {
		t.Fatalf("type inesperado: %s", p.Type)
	}
	for _, vazamento := range []string{"grpc", "gobreaker", "dial", "connection", "50051"} {
		if contemString(p.Detail, vazamento) {
			t.Errorf("detail vazou detalhe interno %q: %s", vazamento, p.Detail)
		}
	}
}

// US3, cenário 5 — sessão inexistente, sem contatar o estoque.
func TestReservarSessaoInexistenteNaoContataOEstoque(t *testing.T) {
	amb := montar(t, func(a *ambiente) { a.sessoes.erro = shared.ErrNaoEncontrado })
	resp, corpo := reservar(t, amb, "token-bom", "00000000-0000-0000-0000-000000000000",
		map[string]any{"poltronas_ids": []string{"A1"}})

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("esperava 404, obteve %d", resp.StatusCode)
	}
	p := decodificarProblem(t, resp, corpo)
	if p.Type != "https://cinema.example/errors/sessao-nao-encontrada" {
		t.Fatalf("type inesperado: %s", p.Type)
	}
	if amb.estoque.chamadas != 0 {
		t.Errorf("o estoque foi contatado %d vez(es) para sessão inexistente", amb.estoque.chamadas)
	}
}

// US3, cenário 6.
func TestReservarCorpoInvalido(t *testing.T) {
	casos := map[string]any{
		"lista vazia":    map[string]any{"poltronas_ids": []string{}},
		"lista ausente":  map[string]any{},
		"duplicatas":     map[string]any{"poltronas_ids": []string{"A1", "A1"}},
		"campo estranho": map[string]any{"poltronas_ids": []string{"A1"}, "desconto": true},
	}
	for nome, corpo := range casos {
		t.Run(nome, func(t *testing.T) {
			amb := montar(t, func(a *ambiente) { a.sessoes.sessao = sessaoReservavel() })
			resp, dados := reservar(t, amb, "token-bom", sessaoID, corpo)
			if resp.StatusCode != http.StatusBadRequest {
				t.Fatalf("esperava 400, obteve %d (%s)", resp.StatusCode, dados)
			}
			p := decodificarProblem(t, resp, dados)
			if p.Type != "https://cinema.example/errors/corpo-invalido" {
				t.Errorf("type inesperado: %s", p.Type)
			}
			if amb.estoque.chamadas != 0 {
				t.Errorf("o estoque foi contatado com corpo inválido")
			}
		})
	}
}

// Sessão que já começou: existe, mas não aceita mais reservas.
func TestReservarSessaoNaoReservavelDevolve422(t *testing.T) {
	amb := montar(t, func(a *ambiente) {
		a.sessoes.sessao = catalogo.Sessao{ID: sessaoID, Status: catalogo.SessaoAgendada, DataHoraInicio: agora().Add(-time.Hour)}
	})
	resp, corpo := reservar(t, amb, "token-bom", sessaoID, map[string]any{"poltronas_ids": []string{"A1"}})

	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("esperava 422, obteve %d", resp.StatusCode)
	}
	p := decodificarProblem(t, resp, corpo)
	if p.Type != "https://cinema.example/errors/sessao-nao-reservavel" {
		t.Fatalf("type inesperado: %s", p.Type)
	}
	if amb.estoque.chamadas != 0 {
		t.Error("o estoque foi contatado para uma sessão que já começou")
	}
}

// Sucesso sem os dados obrigatórios nunca vira 201.
func TestReservarSucessoIncompletoDoParceiroDevolve502(t *testing.T) {
	amb := montar(t, func(a *ambiente) {
		a.sessoes.sessao = sessaoReservavel()
		a.estoque.resultado = reserva.ResultadoReserva{ExpiraEm: agora()} // sem reserva_id
	})
	resp, corpo := reservar(t, amb, "token-bom", sessaoID, map[string]any{"poltronas_ids": []string{"A1"}})

	if resp.StatusCode != http.StatusBadGateway {
		t.Fatalf("esperava 502, obteve %d", resp.StatusCode)
	}
	p := decodificarProblem(t, resp, corpo)
	if p.Type != "https://cinema.example/errors/resposta-invalida-do-parceiro" {
		t.Fatalf("type inesperado: %s", p.Type)
	}
}

func TestReservarSessaoIDMalformadoDevolve400(t *testing.T) {
	amb := montar(t, func(a *ambiente) { a.sessoes.sessao = sessaoReservavel() })
	resp, _ := reservar(t, amb, "token-bom", "nao-e-uuid", map[string]any{"poltronas_ids": []string{"A1"}})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("esperava 400, obteve %d", resp.StatusCode)
	}
	if amb.estoque.chamadas != 0 {
		t.Error("o estoque foi contatado com identificador malformado")
	}
}
