package reserva

import (
	"errors"
	"testing"
	"time"

	"github.com/oseias/ingressos-golang/catalogo/internal/domain/shared"
)

func solicitacaoValida() SolicitacaoReserva {
	return SolicitacaoReserva{
		SessaoID:     "f781a9b2-11e2-4f81-a901-8890bc123456",
		PoltronasIDs: []string{"A1", "A2"},
		UsuarioID:    "9982a1b3-44c1-4221-a123-902183120192",
	}
}

func TestValidarAceitaSolicitacaoBemFormada(t *testing.T) {
	if err := solicitacaoValida().Validar(); err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
}

func TestValidarRecusaListaVazia(t *testing.T) {
	s := solicitacaoValida()
	s.PoltronasIDs = nil
	if err := s.Validar(); !errors.Is(err, shared.ErrValidacao) {
		t.Fatalf("esperava ErrValidacao, obteve %v", err)
	}
}

func TestValidarRecusaDuplicatas(t *testing.T) {
	s := solicitacaoValida()
	s.PoltronasIDs = []string{"A1", "A2", "A1"}
	err := s.Validar()
	if !errors.Is(err, shared.ErrValidacao) {
		t.Fatalf("esperava ErrValidacao, obteve %v", err)
	}
	if !contains(err.Error(), "A1") {
		t.Errorf("erro deveria nomear a poltrona repetida: %v", err)
	}
}

func TestValidarRecusaUsuarioAusente(t *testing.T) {
	s := solicitacaoValida()
	s.UsuarioID = ""
	if err := s.Validar(); !errors.Is(err, shared.ErrValidacao) {
		t.Fatalf("esperava ErrValidacao, obteve %v", err)
	}
}

func TestValidarDistingueCaixaNasPoltronas(t *testing.T) {
	s := solicitacaoValida()
	s.PoltronasIDs = []string{"a1", "A1"}
	if err := s.Validar(); err != nil {
		t.Fatalf("a1 e A1 são identificadores distintos; não deveria recusar: %v", err)
	}
}

func TestIntegridadeRecusaSucessoIncompleto(t *testing.T) {
	casos := map[string]ResultadoReserva{
		"sem identificador": {ExpiraEm: time.Now()},
		"sem expiração":     {ReservaID: "abc"},
		"vazio":             {},
	}
	for nome, r := range casos {
		if err := r.ValidarIntegridade(); !errors.Is(err, shared.ErrRespostaInvalidaDoParceiro) {
			t.Errorf("%s: esperava ErrRespostaInvalidaDoParceiro, obteve %v", nome, err)
		}
	}
	completo := ResultadoReserva{ReservaID: "abc", ExpiraEm: time.Now()}
	if err := completo.ValidarIntegridade(); err != nil {
		t.Errorf("resultado completo não deveria falhar: %v", err)
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
