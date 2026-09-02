package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/oseias/ingressos-golang/notificacao/internal/domain/ingresso"
)

func semearVarios(t *testing.T, repo *ingressosFalsos) {
	t.Helper()
	base := []struct {
		id, usuario string
		status      ingresso.Status
	}{
		{"a", usuario1, ingresso.Valido},
		{"b", usuario1, ingresso.Utilizado},
		{"c", usuario1, ingresso.Cancelado},
		{"d", "outra-pessoa", ingresso.Valido},
	}
	for k, b := range base {
		i, err := ingresso.Novo(b.id, "res-"+b.id, b.usuario, "CIN1."+b.id+".assinatura",
			instanteFixo.Add(time.Duration(k)*time.Minute))
		if err != nil {
			t.Fatalf("preparação: %v", err)
		}
		switch b.status {
		case ingresso.Utilizado:
			i, _ = i.Utilizar(instanteFixo)
		case ingresso.Cancelado:
			i, _ = i.Cancelar()
		}
		repo.semear(i)
	}
}

func TestListarDevolveTodosOsEstadosDaPessoa(t *testing.T) {
	repo := novosIngressos()
	semearVarios(t, repo)

	lista, err := ListarIngressos{Ingressos: repo}.Executar(context.Background(), usuario1, "")
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if len(lista) != 3 {
		t.Errorf("%d ingressos, queria 3", len(lista))
	}
	for _, i := range lista {
		if i.UsuarioID != usuario1 {
			t.Errorf("ingresso de terceiro na listagem: %q (FR-014)", i.UsuarioID)
		}
	}
}

func TestListarFiltraPorEstado(t *testing.T) {
	repo := novosIngressos()
	semearVarios(t, repo)

	lista, err := ListarIngressos{Ingressos: repo}.Executar(context.Background(), usuario1, "VALIDO")
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if len(lista) != 1 || lista[0].Status != ingresso.Valido {
		t.Errorf("filtro por VALIDO devolveu %d itens: %+v", len(lista), lista)
	}
}

func TestFiltroDesconhecidoERecusado(t *testing.T) {
	repo := novosIngressos()
	semearVarios(t, repo)

	lista, err := ListarIngressos{Ingressos: repo}.Executar(context.Background(), usuario1, "INVENTADO")
	if !errors.Is(err, ErrStatusDesconhecido) {
		t.Errorf("erro = %v, queria ErrStatusDesconhecido", err)
	}
	if lista != nil {
		t.Error("filtro inválido devolveu a lista em vez de recusar")
	}
}

func TestPessoaSemIngressosRecebeListaVazia(t *testing.T) {
	lista, err := ListarIngressos{Ingressos: novosIngressos()}.Executar(context.Background(), usuario1, "")
	if err != nil {
		t.Fatalf("listagem vazia devolveu erro: %v", err)
	}
	if lista == nil || len(lista) != 0 {
		t.Errorf("lista = %v, queria vazia e não nula", lista)
	}
}
