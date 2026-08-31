package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/oseias/ingressos-golang/notificacao/internal/domain/aviso"
	"github.com/oseias/ingressos-golang/notificacao/internal/domain/ingresso"
)

// Anuncio é o fato pagamento.sucesso como este serviço o consome. Campos que o
// produtor publica e que aqui não interessam — valor_total, evento, versao,
// ocorrido_em — não aparecem: são tolerados e descartados (research.md D1).
type Anuncio struct {
	TransacaoID string `json:"transacao_id"`
	ReservaID   string `json:"reserva_id"`
	UsuarioID   string `json:"usuario_id"`
	PagoEm      string `json:"pago_em"`
}

// Desfecho é o que o consumidor faz com a mensagem (contracts/eventos.md §3).
type Desfecho int

const (
	// Confirmar — trabalho concluído (inclusive a reentrega inerte).
	Confirmar Desfecho = iota
	// Quarentena — defeito permanente: nunca vai melhorar na reentrega.
	Quarentena
	// NovaTentativa — defeito transitório: pode melhorar.
	NovaTentativa
)

func (d Desfecho) String() string {
	switch d {
	case Confirmar:
		return "confirmado"
	case Quarentena:
		return "quarentena"
	default:
		return "nova_tentativa"
	}
}

// ErrAnuncioInvalido marca o defeito permanente da FR-002.
var ErrAnuncioInvalido = errors.New("usecase: anúncio inválido")

// EmitirIngresso é o fluxo do consumo (FR-001 a FR-006, FR-016 a FR-018).
type EmitirIngresso struct {
	Ingressos   Ingressos
	Avisos      Avisos
	Notificador Notificador
	Assinador   Assinador
	Relogio     Relogio
	IDs         GeradorID
	Log         *slog.Logger
}

// Executar processa um anúncio. Ordem fixa (research.md D6): validar → gravar o
// ingresso → avisar → registrar o aviso → confirmar.
//
// O erro do notificador é capturado e NUNCA propagado: uma falha de aviso não
// pode virar reprocessamento da mensagem (FR-025).
func (u EmitirIngresso) Executar(ctx context.Context, a Anuncio) (Desfecho, error) {
	if err := validar(a); err != nil {
		return Quarentena, err
	}

	// O código assina o identificador, então o identificador vem primeiro e o
	// ingresso já nasce completo — nenhum campo é preenchido depois da criação,
	// que é o que a FR-020 exige do ciclo de vida.
	id := u.IDs.Novo()
	novo, err := ingresso.Novo(id, a.ReservaID, a.UsuarioID, u.Assinador.Gerar(id), u.Relogio.Agora())
	if err != nil {
		return Quarentena, fmt.Errorf("%w: %w", ErrAnuncioInvalido, err)
	}

	criado, atual, err := u.Ingressos.CriarSeAusente(ctx, novo)
	if err != nil {
		// Falha do banco é transitória por padrão: pode melhorar na reentrega.
		return NovaTentativa, fmt.Errorf("gravar ingresso: %w", err)
	}
	if !criado {
		// Reentrega inerte: o ingresso já existe e o aviso já saiu (ou já
		// falhou) na primeira vez. Não se avisa de novo (research.md D6).
		u.log().Info("anúncio já processado, nada a fazer",
			"reserva_id", a.ReservaID, "ingresso_id", atual.ID,
			"desfecho", Confirmar.String())
		return Confirmar, nil
	}

	u.avisar(ctx, atual)

	u.log().Info("ingresso emitido",
		"reserva_id", a.ReservaID, "ingresso_id", atual.ID,
		"transacao_id", a.TransacaoID, "desfecho", Confirmar.String())
	return Confirmar, nil
}

// avisar dispara o aviso e registra o desfecho. Não devolve erro de propósito:
// é aqui que a garantia da FR-025 mora, e ela é uma regra local — este erro não
// sobe.
func (u EmitirIngresso) avisar(ctx context.Context, ing ingresso.Ingresso) {
	agora := u.Relogio.Agora()
	canal := u.Notificador.Canal()

	var reg aviso.Registro
	if err := u.Notificador.Avisar(ctx, ing); err != nil {
		var errReg error
		reg, errReg = aviso.NovoFalho(u.IDs.Novo(), ing.ID, ing.UsuarioID, canal, err.Error(), agora)
		if errReg != nil {
			// Só acontece com detalhe vazio; guarda de sanidade.
			reg, _ = aviso.NovoFalho(u.IDs.Novo(), ing.ID, ing.UsuarioID, canal, "falha sem detalhe", agora)
		}
		u.log().Warn("aviso não saiu; ingresso permanece válido",
			"ingresso_id", ing.ID, "canal", string(canal),
			"desfecho_aviso", string(aviso.Falha), "erro", err)
	} else {
		reg = aviso.NovoEnviado(u.IDs.Novo(), ing.ID, ing.UsuarioID, canal, agora)
		u.log().Info("aviso enviado",
			"ingresso_id", ing.ID, "canal", string(canal),
			"desfecho_aviso", string(aviso.Enviado))
	}

	if err := u.Avisos.Registrar(ctx, reg); err != nil {
		// Também não sobe: a emissão está feita e o ingresso é válido. O que se
		// perde é a trilha, e isso é registrado para ser investigado.
		u.log().Error("registro de aviso não gravado",
			"ingresso_id", ing.ID, "erro", err)
	}
}

func (u EmitirIngresso) log() *slog.Logger {
	if u.Log != nil {
		return u.Log
	}
	return slog.Default()
}

// validar aplica a FR-002: os quatro campos que este serviço usa precisam estar
// presentes e bem formados. Ausência ou má formação é defeito permanente.
func validar(a Anuncio) error {
	for _, c := range []struct {
		nome, valor string
	}{
		{"reserva_id", a.ReservaID},
		{"usuario_id", a.UsuarioID},
		{"transacao_id", a.TransacaoID},
	} {
		if c.valor == "" {
			return fmt.Errorf("%w: %s ausente", ErrAnuncioInvalido, c.nome)
		}
		if !uuidBemFormado(c.valor) {
			return fmt.Errorf("%w: %s malformado", ErrAnuncioInvalido, c.nome)
		}
	}
	if a.PagoEm == "" {
		return fmt.Errorf("%w: pago_em ausente", ErrAnuncioInvalido)
	}
	if _, err := time.Parse(time.RFC3339, a.PagoEm); err != nil {
		return fmt.Errorf("%w: pago_em fora do RFC 3339", ErrAnuncioInvalido)
	}
	return nil
}

// uuidBemFormado confere a forma canônica 8-4-4-4-12 em hexadecimal.
func uuidBemFormado(s string) bool {
	if len(s) != 36 {
		return false
	}
	for i, r := range s {
		switch i {
		case 8, 13, 18, 23:
			if r != '-' {
				return false
			}
		default:
			if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')) {
				return false
			}
		}
	}
	return true
}

// DecodificarAnuncio lê o payload. JSON quebrado é defeito permanente.
func DecodificarAnuncio(corpo []byte) (Anuncio, error) {
	var a Anuncio
	if err := json.Unmarshal(corpo, &a); err != nil {
		return Anuncio{}, fmt.Errorf("%w: json ilegível: %w", ErrAnuncioInvalido, err)
	}
	return a, nil
}
