package amqp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/oseias/ingressos-golang/notificacao/internal/domain/aviso"
	"github.com/oseias/ingressos-golang/notificacao/internal/domain/ingresso"
	"github.com/oseias/ingressos-golang/notificacao/internal/usecase"
)

const (
	reserva1 = "9982a1b3-44c1-4221-a123-902183120192"
	usuario1 = "c394c8b3-76a1-4328-b803-02f5923b7a15"
	trans1   = "e402a129-8812-4211-b123-000129381293"
	pagoEm1  = "2026-08-29T21:35:10Z"
)

var instanteFixo = time.Date(2026, 8, 29, 21, 35, 12, 0, time.UTC)

type relogioFixo struct{}

func (relogioFixo) Agora() time.Time { return instanteFixo }

type idsFixos struct {
	mu sync.Mutex
	n  int
}

func (g *idsFixos) Novo() string {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.n++
	return "00000000-0000-4000-8000-00000000000" + string(rune('0'+g.n%10))
}

type assinadorFalso struct{}

func (assinadorFalso) Gerar(id string) string           { return "CIN1." + id + ".assinatura" }
func (assinadorFalso) Verificar(string) (string, error) { return "", errors.New("não usado") }

var errBanco = errors.New("banco indisponível")

type repoFalso struct {
	mu     sync.Mutex
	porRes map[string]ingresso.Ingresso
	falhar bool
}

func (r *repoFalso) CriarSeAusente(_ context.Context, i ingresso.Ingresso) (bool, ingresso.Ingresso, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.falhar {
		return false, ingresso.Ingresso{}, errBanco
	}
	if e, ok := r.porRes[i.ReservaID]; ok {
		return false, e, nil
	}
	r.porRes[i.ReservaID] = i
	return true, i, nil
}
func (r *repoFalso) Utilizar(context.Context, string, time.Time) (bool, error) { return false, nil }
func (r *repoFalso) BuscarPorID(context.Context, string) (ingresso.Ingresso, error) {
	return ingresso.Ingresso{}, usecase.ErrNaoEncontrado
}
func (r *repoFalso) ListarPorUsuario(context.Context, string, ingresso.Status) ([]ingresso.Ingresso, error) {
	return nil, nil
}

type avisosFalsos struct{}

func (avisosFalsos) Registrar(context.Context, aviso.Registro) error { return nil }

type notificadorFalso struct{ falhar bool }

func (notificadorFalso) Canal() aviso.Canal { return aviso.Email }
func (n notificadorFalso) Avisar(context.Context, ingresso.Ingresso) error {
	if n.falhar {
		return errors.New("canal indisponível")
	}
	return nil
}

func montar(t *testing.T, log *slog.Logger) (*Consumidor, *repoFalso) {
	t.Helper()
	repo := &repoFalso{porRes: map[string]ingresso.Ingresso{}}
	return &Consumidor{
		Caso: usecase.EmitirIngresso{
			Ingressos: repo, Avisos: avisosFalsos{}, Notificador: notificadorFalso{},
			Assinador: assinadorFalso{}, Relogio: relogioFixo{}, IDs: &idsFixos{}, Log: log,
		},
		Log: log,
	}, repo
}

func semLog() *slog.Logger { return slog.New(slog.NewTextHandler(new(bytes.Buffer), nil)) }

func anuncioJSON() []byte {
	b, _ := json.Marshal(map[string]any{
		"evento": "PAGAMENTO_SUCESSO", "versao": 1, "ocorrido_em": pagoEm1,
		"transacao_id": trans1, "reserva_id": reserva1, "usuario_id": usuario1,
		"valor_total": 84.00, "pago_em": pagoEm1,
	})
	return b
}

func TestDesfechosDoConsumo(t *testing.T) {
	ctx := context.Background()

	t.Run("anúncio válido, reserva nova → ack", func(t *testing.T) {
		c, _ := montar(t, semLog())
		if g := c.Classificar(ctx, anuncioJSON(), false); g != Confirmar {
			t.Errorf("gesto = %v, queria Confirmar", g)
		}
	})

	t.Run("reserva já com ingresso → ack inerte", func(t *testing.T) {
		c, _ := montar(t, semLog())
		c.Classificar(ctx, anuncioJSON(), false)
		if g := c.Classificar(ctx, anuncioJSON(), false); g != Confirmar {
			t.Errorf("gesto da reentrega = %v, queria Confirmar", g)
		}
	})

	t.Run("JSON ilegível → quarentena", func(t *testing.T) {
		c, _ := montar(t, semLog())
		if g := c.Classificar(ctx, []byte("{nao e json"), false); g != Quarentena {
			t.Errorf("gesto = %v, queria Quarentena", g)
		}
	})

	t.Run("campo obrigatório ausente → quarentena", func(t *testing.T) {
		c, _ := montar(t, semLog())
		corpo := []byte(`{"reserva_id":"` + reserva1 + `","usuario_id":"` + usuario1 + `"}`)
		if g := c.Classificar(ctx, corpo, false); g != Quarentena {
			t.Errorf("gesto = %v, queria Quarentena", g)
		}
	})

	t.Run("UUID malformado → quarentena", func(t *testing.T) {
		c, _ := montar(t, semLog())
		corpo := []byte(`{"transacao_id":"` + trans1 + `","reserva_id":"nao-e-uuid","usuario_id":"` + usuario1 + `","pago_em":"` + pagoEm1 + `"}`)
		if g := c.Classificar(ctx, corpo, false); g != Quarentena {
			t.Errorf("gesto = %v, queria Quarentena", g)
		}
	})

	t.Run("banco indisponível → nova tentativa", func(t *testing.T) {
		c, repo := montar(t, semLog())
		repo.falhar = true
		if g := c.Classificar(ctx, anuncioJSON(), false); g != Devolver {
			t.Errorf("gesto = %v, queria Devolver", g)
		}
	})

	t.Run("canal de aviso com erro → ack, não reprocessa", func(t *testing.T) {
		c, _ := montar(t, semLog())
		c.Caso.Notificador = notificadorFalso{falhar: true}
		if g := c.Classificar(ctx, anuncioJSON(), false); g != Confirmar {
			t.Errorf("gesto = %v, queria Confirmar: falha de aviso NÃO pode reprocessar (FR-025)", g)
		}
	})
}

func TestLogRegistraDesfechoDaEmissao(t *testing.T) {
	var buf bytes.Buffer
	log := slog.New(slog.NewJSONHandler(&buf, nil))
	c, _ := montar(t, log)

	c.Classificar(context.Background(), anuncioJSON(), false)

	saida := buf.String()
	if !strings.Contains(saida, "ingresso_id") {
		t.Errorf("o log não registra o ingresso_id do desfecho:\n%s", saida)
	}
	if !strings.Contains(saida, "desfecho") {
		t.Errorf("o log não registra o desfecho:\n%s", saida)
	}
	if !strings.Contains(saida, reserva1) {
		t.Errorf("o log não correlaciona com a reserva:\n%s", saida)
	}
}

func TestLogNaoVazaCodigoDeAcesso(t *testing.T) {
	casos := map[string]func(*Consumidor){
		"emissão bem-sucedida": func(*Consumidor) {},
		"aviso falhou":         func(c *Consumidor) { c.Caso.Notificador = notificadorFalso{falhar: true} },
	}
	for nome, ajustar := range casos {
		t.Run(nome, func(t *testing.T) {
			var buf bytes.Buffer
			c, repo := montar(t, slog.New(slog.NewJSONHandler(&buf, nil)))
			ajustar(c)

			c.Classificar(context.Background(), anuncioJSON(), false)

			emitido, ok := repo.porRes[reserva1]
			if !ok {
				t.Fatal("nada foi emitido; o teste não exerce o que alega")
			}
			if emitido.CodigoQR == "" {
				t.Fatal("ingresso sem código; o teste não exerce o que alega")
			}
			if strings.Contains(buf.String(), emitido.CodigoQR) {
				t.Errorf("o código de acesso vazou para o log (FR-021):\n%s", buf.String())
			}
			if !strings.Contains(buf.String(), emitido.ID) {
				t.Error("o log deveria identificar a operação pelo ingresso_id")
			}
		})
	}
}
