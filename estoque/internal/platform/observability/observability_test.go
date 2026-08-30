package observability

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
	"time"
)

// capturar troca o logger por um que escreve em memória, para inspecionar o que
// realmente vai para o registro.
func capturar(t *testing.T) (*Observabilidade, *bytes.Buffer) {
	t.Helper()
	var buffer bytes.Buffer

	obs, err := Iniciar(context.Background(), "debug", "")
	if err != nil {
		t.Fatalf("iniciar: %v", err)
	}
	obs.Log = slog.New(slog.NewJSONHandler(&buffer, &slog.HandlerOptions{Level: slog.LevelDebug}))
	return obs, &buffer
}

// TestLogOperacaoTemOsCamposExigidos verifica FR-042: correlação, operação,
// desfecho e duração em todo registro de operação e de consumo.
func TestLogOperacaoTemOsCamposExigidos(t *testing.T) {
	obs, buffer := capturar(t)

	obs.LogOperacao(context.Background(), "BloquearPoltronas", "concedido", time.Now(),
		"sessao_id", "s-1", "reserva_id", "r-1")

	var registro map[string]any
	if err := json.Unmarshal(buffer.Bytes(), &registro); err != nil {
		t.Fatalf("registro não é JSON legível por máquina: %v\n%s", err, buffer.String())
	}
	for _, campo := range []string{"operacao", "desfecho", "duracao_ms", "trace_id"} {
		if _, presente := registro[campo]; !presente {
			t.Errorf("registro sem o campo %q: %s", campo, buffer.String())
		}
	}
	if registro["operacao"] != "BloquearPoltronas" || registro["desfecho"] != "concedido" {
		t.Errorf("campos incorretos: %v", registro)
	}
}

// TestLogNaoGravaDadosSensiveis é a guarda da constituição (Restrições
// Técnicas): segredos e material criptográfico não aparecem em registro.
//
// O teste é deliberadamente ingênuo — procura marcadores óbvios — porque seu
// papel é falhar quando alguém acrescentar um campo sensível por descuido.
func TestLogNaoGravaDadosSensiveis(t *testing.T) {
	obs, buffer := capturar(t)

	obs.LogOperacao(context.Background(), "BloquearPoltronas", "concedido", time.Now(),
		"sessao_id", "f781a9b2-11e2-4f81-a901-8890bc123456",
		"reserva_id", "9982a1b3-44c1-4221-a123-902183120192",
		"chamador", "servico-catalogo")

	registrado := buffer.String()
	proibidos := []string{
		"BEGIN PRIVATE KEY", "BEGIN RSA", "BEGIN EC PRIVATE",
		"authorization", "bearer ", "password", "senha=",
	}
	for _, proibido := range proibidos {
		if strings.Contains(strings.ToLower(registrado), strings.ToLower(proibido)) {
			t.Errorf("registro contém material sensível (%q): %s", proibido, registrado)
		}
	}
}

func TestTraceIDVazioSemSpanAtivo(t *testing.T) {
	if id := TraceID(context.Background()); id != "" {
		t.Errorf("TraceID sem span = %q, esperado vazio", id)
	}
}

func TestParseNivel(t *testing.T) {
	casos := map[string]slog.Level{
		"debug": slog.LevelDebug, "DEBUG": slog.LevelDebug,
		"warn": slog.LevelWarn, "warning": slog.LevelWarn,
		"error": slog.LevelError, "": slog.LevelInfo, "qualquer": slog.LevelInfo,
	}
	for entrada, esperado := range casos {
		if got := parseNivel(entrada); got != esperado {
			t.Errorf("parseNivel(%q) = %v, esperado %v", entrada, got, esperado)
		}
	}
}
