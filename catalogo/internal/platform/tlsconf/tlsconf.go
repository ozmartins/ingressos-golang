// Package tlsconf monta a configuração TLS do servidor gRPC.
package tlsconf

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"os"
)

// Servidor carrega o par certificado/chave do servidor. Se clienteCA não for
// vazio, exige certificado de cliente assinado por essa CA (mTLS).
func Servidor(certFile, keyFile, clienteCA string) (*tls.Config, error) {
	par, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("carregar certificado TLS: %w", err)
	}
	cfg := &tls.Config{
		Certificates: []tls.Certificate{par},
		MinVersion:   tls.VersionTLS13,
	}
	if clienteCA != "" {
		pem, err := os.ReadFile(clienteCA)
		if err != nil {
			return nil, fmt.Errorf("ler CA de clientes: %w", err)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(pem) {
			return nil, errors.New("CA de clientes sem certificado PEM válido")
		}
		cfg.ClientCAs = pool
		cfg.ClientAuth = tls.RequireAndVerifyClientCert
	}
	return cfg, nil
}
