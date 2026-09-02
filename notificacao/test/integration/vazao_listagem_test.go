//go:build integration

package integration

import (
	"context"
	"sort"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/oseias/ingressos-golang/notificacao/internal/domain/ingresso"
	"github.com/oseias/ingressos-golang/notificacao/internal/usecase"
)

const historicoAlvo = 200

func TestListagemDe200IngressosRespondeDentroDoPrazo(t *testing.T) {
	a := subirAmbiente(t)
	usuario := uuid.NewString()
	caso := a.caso(false)

	for k := 0; k < historicoAlvo; k++ {
		if _, err := caso.Executar(context.Background(), usecase.Anuncio{
			TransacaoID: uuid.NewString(), ReservaID: uuid.NewString(),
			UsuarioID: usuario, PagoEm: time.Now().UTC().Format(time.RFC3339),
		}); err != nil {
			t.Fatalf("semear ingresso %d: %v", k, err)
		}
	}

	listagem := usecase.ListarIngressos{Ingressos: a.Ingressos}
	const amostras = 40
	duracoes := make([]time.Duration, 0, amostras)
	for k := 0; k < amostras; k++ {
		inicio := time.Now()
		lista, err := listagem.Executar(context.Background(), usuario, "")
		decorrido := time.Since(inicio)
		if err != nil {
			t.Fatalf("listar: %v", err)
		}
		if len(lista) != historicoAlvo {
			t.Fatalf("%d ingressos devolvidos, queria %d", len(lista), historicoAlvo)
		}
		duracoes = append(duracoes, decorrido)
	}

	p95 := percentil(duracoes, 0.95)
	t.Logf("listagem de %d ingressos: p95 = %s", historicoAlvo, p95)
	if p95 > 2*time.Second {
		t.Errorf("p95 = %s, queria menos de 2s (SC-009)", p95)
	}
}

func TestListagemVemOrdenadaDoMaisRecenteAoMaisAntigo(t *testing.T) {
	a := subirAmbiente(t)
	usuario := uuid.NewString()
	caso := a.caso(false)

	for k := 0; k < 25; k++ {
		if _, err := caso.Executar(context.Background(), usecase.Anuncio{
			TransacaoID: uuid.NewString(), ReservaID: uuid.NewString(),
			UsuarioID: usuario, PagoEm: time.Now().UTC().Format(time.RFC3339),
		}); err != nil {
			t.Fatal(err)
		}
	}

	lista, err := usecase.ListarIngressos{Ingressos: a.Ingressos}.
		Executar(context.Background(), usuario, "")
	if err != nil {
		t.Fatal(err)
	}
	for k := 1; k < len(lista); k++ {
		anterior, atual := lista[k-1], lista[k]
		if atual.CriadoEm.After(anterior.CriadoEm) {
			t.Fatalf("ordem quebrada em %d: %v vem depois de %v", k, atual.CriadoEm, anterior.CriadoEm)
		}
		if atual.CriadoEm.Equal(anterior.CriadoEm) && atual.ID > anterior.ID {
			t.Fatalf("desempate por id quebrado em %d", k)
		}
	}

	validos, err := usecase.ListarIngressos{Ingressos: a.Ingressos}.
		Executar(context.Background(), usuario, string(ingresso.Valido))
	if err != nil {
		t.Fatal(err)
	}
	if len(validos) != len(lista) {
		t.Errorf("filtro VALIDO devolveu %d, queria %d", len(validos), len(lista))
	}
}

func percentil(ds []time.Duration, p float64) time.Duration {
	c := append([]time.Duration(nil), ds...)
	sort.Slice(c, func(i, j int) bool { return c[i] < c[j] })
	idx := int(float64(len(c)-1) * p)
	return c[idx]
}
