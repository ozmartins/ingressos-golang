package catalogo

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	pb "github.com/oseias/ingressos-golang/estoque/gen/pb/catalogo/v1"
)

type par struct {
	cert *x509.Certificate
	key  *ecdsa.PrivateKey
}

func emitir(t *testing.T, cn string, ca *par, ehCA bool, dns []string, uso x509.ExtKeyUsage) par {
	t.Helper()
	chave, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	modelo := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()), Subject: pkix.Name{CommonName: cn},
		NotBefore: time.Now().Add(-time.Minute), NotAfter: time.Now().Add(time.Hour),
		KeyUsage: x509.KeyUsageDigitalSignature, DNSNames: dns,
		BasicConstraintsValid: true, IsCA: ehCA,
	}
	if ehCA {
		modelo.KeyUsage |= x509.KeyUsageCertSign
	} else {
		modelo.ExtKeyUsage = []x509.ExtKeyUsage{uso}
	}
	pai, chavePai := modelo, chave
	if ca != nil {
		pai, chavePai = ca.cert, ca.key
	}
	der, err := x509.CreateCertificate(rand.Reader, modelo, pai, &chave.PublicKey, chavePai)
	if err != nil {
		t.Fatal(err)
	}
	c, _ := x509.ParseCertificate(der)
	return par{c, chave}
}

func gravar(t *testing.T, dir, nome string, p par) (certPath, keyPath string) {
	t.Helper()
	certPath = filepath.Join(dir, nome+".pem")
	keyPath = filepath.Join(dir, nome+"-key.pem")
	if err := os.WriteFile(certPath, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: p.cert.Raw}), 0o600); err != nil {
		t.Fatal(err)
	}
	der, _ := x509.MarshalECPrivateKey(p.key)
	if err := os.WriteFile(keyPath, pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: der}), 0o600); err != nil {
		t.Fatal(err)
	}
	return
}

// servidorTLS sobe o servidor falso com mTLS e devolve o endereço e a CA.
func servidorTLS(t *testing.T, dir string, ca par) string {
	t.Helper()
	srvPar := emitir(t, "catalogo", &ca, false, []string{"localhost"}, x509.ExtKeyUsageServerAuth)
	cPath, kPath := gravar(t, dir, "catalogo", srvPar)
	kp, err := tls.LoadX509KeyPair(cPath, kPath)
	if err != nil {
		t.Fatal(err)
	}
	pool := x509.NewCertPool()
	pool.AddCert(ca.cert)
	srv := grpc.NewServer(grpc.Creds(credentials.NewTLS(&tls.Config{
		Certificates: []tls.Certificate{kp}, ClientCAs: pool,
		ClientAuth: tls.RequireAndVerifyClientCert, MinVersion: tls.VersionTLS13,
	})))
	pb.RegisterCatalogoServiceServer(srv, &servidorFalso{
		resp: func(context.Context, string) (*pb.Sessao, error) { return sessaoValida(), nil },
	})
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)
	return lis.Addr().String()
}

func TestTLSMutuoComCertificadoValido(t *testing.T) {
	dir := t.TempDir()
	ca := emitir(t, "ca", nil, true, nil, 0)
	caPath, _ := gravar(t, dir, "ca", ca)
	addr := servidorTLS(t, dir, ca)
	cli := emitir(t, "estoque", &ca, false, nil, x509.ExtKeyUsageClientAuth)
	cliCert, cliKey := gravar(t, dir, "estoque", cli)

	// O endereço é 127.0.0.1; o certificado só vale para "localhost".
	c, err := NovoCliente(Opcoes{
		Endereco: "localhost:" + addr[len("127.0.0.1:"):], Timeout: 2 * time.Second,
		FalhasParaAbrir: 5, IntervaloAberto: time.Hour,
		TLSCA: caPath, TLSCert: cliCert, TLSKey: cliKey,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Fechar()
	if _, err := c.BuscarSessao(context.Background(), "s1"); err != nil {
		t.Fatalf("handshake mTLS falhou: %v", err)
	}
}

func TestTLSRecusaClienteSemCertificado(t *testing.T) {
	dir := t.TempDir()
	ca := emitir(t, "ca", nil, true, nil, 0)
	caPath, _ := gravar(t, dir, "ca", ca)
	addr := servidorTLS(t, dir, ca)

	c, err := NovoCliente(Opcoes{
		Endereco: "localhost:" + addr[len("127.0.0.1:"):], Timeout: 2 * time.Second,
		FalhasParaAbrir: 5, IntervaloAberto: time.Hour, TLSCA: caPath,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Fechar()
	if _, err := c.BuscarSessao(context.Background(), "s1"); err == nil {
		t.Fatal("servidor com mTLS aceitou cliente sem certificado")
	}
}

func TestTLSRecusaServidorDeOutraCA(t *testing.T) {
	dir := t.TempDir()
	ca := emitir(t, "ca", nil, true, nil, 0)
	addr := servidorTLS(t, dir, ca)
	outra := emitir(t, "outra-ca", nil, true, nil, 0)
	outraPath, _ := gravar(t, dir, "outra", outra)

	c, err := NovoCliente(Opcoes{
		Endereco: "localhost:" + addr[len("127.0.0.1:"):], Timeout: 2 * time.Second,
		FalhasParaAbrir: 5, IntervaloAberto: time.Hour, TLSCA: outraPath,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Fechar()
	if _, err := c.BuscarSessao(context.Background(), "s1"); err == nil {
		t.Fatal("cliente confiou em servidor assinado por outra CA")
	}
}

func TestTLSConfiguracaoInvalida(t *testing.T) {
	if _, err := NovoCliente(Opcoes{Endereco: "x:1", TLSCA: "/nao/existe.pem"}); err == nil {
		t.Error("CA inexistente deveria falhar")
	}
	dir := t.TempDir()
	ca := emitir(t, "ca", nil, true, nil, 0)
	caPath, _ := gravar(t, dir, "ca", ca)
	if _, err := NovoCliente(Opcoes{Endereco: "x:1", TLSCA: caPath, TLSCert: "so-cert.pem"}); err == nil {
		t.Error("cert sem chave deveria falhar")
	}
}
