package ingresso

import (
	"errors"
	"testing"
	"time"
)

var (
	t0 = time.Date(2026, 8, 29, 21, 35, 12, 0, time.UTC)
	t1 = time.Date(2026, 8, 29, 22, 0, 0, 0, time.UTC)
)

func novoValido(t *testing.T) Ingresso {
	t.Helper()
	i, err := Novo("ing-1", "res-1", "usr-1", "CIN1.abc.def", t0)
	if err != nil {
		t.Fatalf("Novo devolveu erro inesperado: %v", err)
	}
	return i
}

func TestNovoNasceValidoSemInstante(t *testing.T) {
	i := novoValido(t)
	if i.Status != Valido {
		t.Errorf("status = %q, queria %q", i.Status, Valido)
	}
	if i.UtilizadoEm != nil {
		t.Error("ingresso recém-criado não pode ter instante de utilização")
	}
	if !i.CriadoEm.Equal(t0) {
		t.Errorf("criadoEm = %v, queria %v", i.CriadoEm, t0)
	}
	if !i.InvarianteInstante() {
		t.Error("invariante do instante violada na criação")
	}
}

func TestNovoExigeIdentidadeCompleta(t *testing.T) {
	casos := map[string][4]string{
		"sem id":      {"", "res-1", "usr-1", "cod"},
		"sem reserva": {"ing-1", "", "usr-1", "cod"},
		"sem usuario": {"ing-1", "res-1", "", "cod"},
		"sem codigo":  {"ing-1", "res-1", "usr-1", ""},
	}
	for nome, c := range casos {
		t.Run(nome, func(t *testing.T) {
			if _, err := Novo(c[0], c[1], c[2], c[3], t0); !errors.Is(err, ErrDadosObrigatorios) {
				t.Errorf("erro = %v, queria ErrDadosObrigatorios", err)
			}
		})
	}
}

func TestUtilizarGravaInstanteEMantemInvariante(t *testing.T) {
	i := novoValido(t)
	u, err := i.Utilizar(t1)
	if err != nil {
		t.Fatalf("Utilizar devolveu erro: %v", err)
	}
	if u.Status != Utilizado {
		t.Errorf("status = %q, queria %q", u.Status, Utilizado)
	}
	if u.UtilizadoEm == nil || !u.UtilizadoEm.Equal(t1) {
		t.Errorf("utilizadoEm = %v, queria %v", u.UtilizadoEm, t1)
	}
	if !u.InvarianteInstante() {
		t.Error("invariante do instante violada após a baixa")
	}
}

func TestUtilizarNaoAlteraCamposImutaveis(t *testing.T) {
	i := novoValido(t)
	u, err := i.Utilizar(t1)
	if err != nil {
		t.Fatalf("Utilizar devolveu erro: %v", err)
	}
	if u.ID != i.ID {
		t.Errorf("ID mudou: %q → %q", i.ID, u.ID)
	}
	if u.ReservaID != i.ReservaID {
		t.Errorf("ReservaID mudou: %q → %q", i.ReservaID, u.ReservaID)
	}
	if u.UsuarioID != i.UsuarioID {
		t.Errorf("UsuarioID mudou: %q → %q", i.UsuarioID, u.UsuarioID)
	}
	if u.CodigoQR != i.CodigoQR {
		t.Errorf("CodigoQR mudou: %q → %q", i.CodigoQR, u.CodigoQR)
	}
	if !u.CriadoEm.Equal(i.CriadoEm) {
		t.Errorf("CriadoEm mudou: %v → %v", i.CriadoEm, u.CriadoEm)
	}
}

func TestCancelarNaoAlteraCamposImutaveis(t *testing.T) {
	i := novoValido(t)
	c, err := i.Cancelar()
	if err != nil {
		t.Fatalf("Cancelar devolveu erro: %v", err)
	}
	if c.ReservaID != i.ReservaID || c.UsuarioID != i.UsuarioID ||
		c.CodigoQR != i.CodigoQR || !c.CriadoEm.Equal(i.CriadoEm) {
		t.Error("cancelamento alterou campo imutável (FR-020)")
	}
	if c.UtilizadoEm != nil {
		t.Error("ingresso cancelado não pode ganhar instante de utilização")
	}
	if !c.InvarianteInstante() {
		t.Error("invariante do instante violada no cancelamento")
	}
}

func TestEstadoTerminalNaoTransiciona(t *testing.T) {
	utilizado, err := novoValido(t).Utilizar(t1)
	if err != nil {
		t.Fatalf("preparação falhou: %v", err)
	}
	cancelado, err := novoValido(t).Cancelar()
	if err != nil {
		t.Fatalf("preparação falhou: %v", err)
	}

	casos := map[string]Ingresso{"utilizado": utilizado, "cancelado": cancelado}
	for nome, base := range casos {
		t.Run(nome+"/utilizar", func(t *testing.T) {
			depois, err := base.Utilizar(t1.Add(time.Hour))
			if !errors.Is(err, ErrTransicaoInvalida) {
				t.Errorf("erro = %v, queria ErrTransicaoInvalida", err)
			}
			if depois.Status != base.Status {
				t.Errorf("estado mudou apesar do erro: %q → %q", base.Status, depois.Status)
			}
			if base.Status == Utilizado && !depois.UtilizadoEm.Equal(*base.UtilizadoEm) {
				t.Error("instante de utilização original foi alterado (FR-008)")
			}
		})
		t.Run(nome+"/cancelar", func(t *testing.T) {
			depois, err := base.Cancelar()
			if !errors.Is(err, ErrTransicaoInvalida) {
				t.Errorf("erro = %v, queria ErrTransicaoInvalida", err)
			}
			if depois.Status != base.Status {
				t.Errorf("estado mudou apesar do erro: %q → %q", base.Status, depois.Status)
			}
		})
	}
}

func TestReconhecido(t *testing.T) {
	for _, s := range []Status{Valido, Utilizado, Cancelado} {
		if !Reconhecido(s) {
			t.Errorf("%q deveria ser reconhecido", s)
		}
	}
	for _, s := range []Status{"", "INVENTADO", "valido"} {
		if Reconhecido(s) {
			t.Errorf("%q não deveria ser reconhecido", s)
		}
	}
}
