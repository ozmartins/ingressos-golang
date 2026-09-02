package contract

import (
	"net/http"
	"testing"

	"github.com/oseias/ingressos-golang/catalogo/internal/domain/catalogo"
)

const cinemaID = "b1b2c3d4-0000-4000-8000-000000000001"

func TestGetCinemasRetornaLocalizacao(t *testing.T) {
	amb := montar(t, func(a *ambiente) {
		a.cinemas.itens = []catalogo.Cinema{
			{ID: cinemaID, Nome: "CineMark - Shopping Centro", Cidade: "Florianópolis", Estado: "SC", Endereco: "Rua X, 100"},
		}
	})
	resp, corpo := obter(t, amb.servidor, "/api/v1/cinemas")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
	e := decodificarEnvelope(t, corpo)
	for _, campo := range []string{"id", "nome", "cidade", "estado", "endereco"} {
		if _, ok := e.Itens[0][campo]; !ok {
			t.Errorf("campo obrigatório %q ausente", campo)
		}
	}
	if e.Itens[0]["estado"] != "SC" {
		t.Errorf("estado inesperado: %v", e.Itens[0]["estado"])
	}
}

func TestGetSalasDeCinemaInexistenteDevolve404(t *testing.T) {
	amb := montar(t, func(a *ambiente) { a.cinemas.existe = false })
	resp, corpo := obter(t, amb.servidor, "/api/v1/cinemas/00000000-0000-0000-0000-000000000000/salas")

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("esperava 404, obteve %d", resp.StatusCode)
	}
	p := decodificarProblem(t, resp, corpo)
	if p.Type != "https://cinema.example/errors/cinema-nao-encontrado" {
		t.Fatalf("type inesperado: %s", p.Type)
	}
}

func TestGetSalasDeCinemaSemSalasDevolve200Vazio(t *testing.T) {
	amb := montar(t, func(a *ambiente) { a.cinemas.existe = true })
	resp, corpo := obter(t, amb.servidor, "/api/v1/cinemas/"+cinemaID+"/salas")

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("esperava 200, obteve %d", resp.StatusCode)
	}
	e := decodificarEnvelope(t, corpo)
	if len(e.Itens) != 0 || e.Pagina.Total != 0 {
		t.Fatalf("esperava página vazia, obteve %+v", e)
	}
}

func TestGetSalasRecusaCinemaIDMalformado(t *testing.T) {
	amb := montar(t, nil)
	resp, _ := obter(t, amb.servidor, "/api/v1/cinemas/nao-e-uuid/salas")
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("esperava 400, obteve %d", resp.StatusCode)
	}
}
