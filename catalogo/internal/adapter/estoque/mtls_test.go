package estoque

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// O canal com o estoque é mTLS obrigatório: o serviço não sobe sem material
// válido. Estes testes cobrem o que `NovoCliente` recusa — nenhum deles precisa
// de servidor, porque a falha acontece antes de qualquer conexão.
func TestNovoClienteRecusaMaterialInvalido(t *testing.T) {
	dir := t.TempDir()
	ca, cert, chave := escreverParDeTeste(t, dir)

	casos := map[string]struct {
		opcoes Opcoes
		trecho string
	}{
		"sem material nenhum": {
			Opcoes{Endereco: "estoque:50051"},
			"par de cliente",
		},
		"certificado inexistente": {
			Opcoes{Endereco: "estoque:50051", CAFile: ca,
				CertFile: filepath.Join(dir, "nao-existe.pem"), KeyFile: chave},
			"par de cliente",
		},
		"chave que não é do certificado": {
			Opcoes{Endereco: "estoque:50051", CAFile: ca, CertFile: cert, KeyFile: ca},
			"par de cliente",
		},
		"CA inexistente": {
			Opcoes{Endereco: "estoque:50051", CAFile: filepath.Join(dir, "nao-existe.pem"),
				CertFile: cert, KeyFile: chave},
			"CA do estoque",
		},
		"CA que não é PEM": {
			Opcoes{Endereco: "estoque:50051", CAFile: escreverLixo(t, dir),
				CertFile: cert, KeyFile: chave},
			"não contém certificado PEM válido",
		},
	}

	for nome, caso := range casos {
		t.Run(nome, func(t *testing.T) {
			cliente, err := NovoCliente(caso.opcoes)
			if err == nil {
				_ = cliente.Fechar()
				t.Fatal("esperava recusa: sem material válido não há canal com o estoque")
			}
			if !strings.Contains(err.Error(), caso.trecho) {
				t.Fatalf("o erro deveria dizer o que faltou (%q), veio: %v", caso.trecho, err)
			}
		})
	}
}

func TestNovoClienteAceitaMaterialValido(t *testing.T) {
	dir := t.TempDir()
	ca, cert, chave := escreverParDeTeste(t, dir)

	cliente, err := NovoCliente(Opcoes{
		Endereco: "estoque:50051", CAFile: ca, CertFile: cert, KeyFile: chave,
	})
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	t.Cleanup(func() { _ = cliente.Fechar() })
}

// Um par autoassinado, só para exercitar o carregamento. Não precisa ser
// assinado por CA de verdade: o que se testa aqui é o que `NovoCliente` recusa
// antes de abrir conexão, e o handshake em si é exercitado com o estoque de pé.
func escreverParDeTeste(t *testing.T, dir string) (ca, cert, chave string) {
	t.Helper()

	privada, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	molde := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "servico-catalogo"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
	}
	der, err := x509.CreateCertificate(rand.Reader, molde, molde, &privada.PublicKey, privada)
	if err != nil {
		t.Fatal(err)
	}
	derChave, err := x509.MarshalECPrivateKey(privada)
	if err != nil {
		t.Fatal(err)
	}

	cert = escrever(t, dir, "cliente.pem", pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}))
	chave = escrever(t, dir, "cliente-key.pem", pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: derChave}))
	// O mesmo certificado serve de CA: ele é autoassinado.
	ca = escrever(t, dir, "ca.pem", pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}))
	return ca, cert, chave
}

func escrever(t *testing.T, dir, nome string, conteudo []byte) string {
	t.Helper()
	caminho := filepath.Join(dir, nome)
	if err := os.WriteFile(caminho, conteudo, 0o600); err != nil {
		t.Fatal(err)
	}
	return caminho
}

func escreverLixo(t *testing.T, dir string) string {
	t.Helper()
	caminho := filepath.Join(dir, "lixo.pem")
	if err := os.WriteFile(caminho, []byte("isto não é um certificado"), 0o600); err != nil {
		t.Fatal(err)
	}
	return caminho
}
