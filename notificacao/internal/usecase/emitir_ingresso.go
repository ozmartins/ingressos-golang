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

type Anuncio struct {
	TransacaoID string `json:"transacao_id"`
	ReservaID   string `json:"reserva_id"`
	UsuarioID   string `json:"usuario_id"`
	PagoEm      string `json:"pago_em"`
}

type Desfecho int

const (
	Confirmar Desfecho = iota
	Quarentena
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

var ErrAnuncioInvalido = errors.New("usecase: anúncio inválido")

type EmitirIngresso struct {
	Ingressos   Ingressos
	Avisos      Avisos
	Notificador Notificador
	Assinador   Assinador
	Relogio     Relogio
	IDs         GeradorID
	Log         *slog.Logger
}

func (u EmitirIngresso) Executar(ctx context.Context, a Anuncio) (Desfecho, error) {
	if err := validar(a); err != nil {
		return Quarentena, err
	}

	id := u.IDs.Novo()
	novo, err := ingresso.Novo(id, a.ReservaID, a.UsuarioID, u.Assinador.Gerar(id), u.Relogio.Agora())
	if err != nil {
		return Quarentena, fmt.Errorf("%w: %w", ErrAnuncioInvalido, err)
	}

	criado, atual, err := u.Ingressos.CriarSeAusente(ctx, novo)
	if err != nil {
		return NovaTentativa, fmt.Errorf("gravar ingresso: %w", err)
	}
	if !criado {
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

func (u EmitirIngresso) avisar(ctx context.Context, ing ingresso.Ingresso) {
	agora := u.Relogio.Agora()
	canal := u.Notificador.Canal()

	var reg aviso.Registro
	if err := u.Notificador.Avisar(ctx, ing); err != nil {
		var errReg error
		reg, errReg = aviso.NovoFalho(u.IDs.Novo(), ing.ID, ing.UsuarioID, canal, err.Error(), agora)
		if errReg != nil {
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

func DecodificarAnuncio(corpo []byte) (Anuncio, error) {
	var a Anuncio
	if err := json.Unmarshal(corpo, &a); err != nil {
		return Anuncio{}, fmt.Errorf("%w: json ilegível: %w", ErrAnuncioInvalido, err)
	}
	return a, nil
}
