package codigo

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"strings"
)

const Prefixo = "CIN1"

var ErrInvalido = errors.New("codigo: código de acesso inválido")

var b64 = base64.RawURLEncoding

type Assinador struct{ segredo []byte }

func NovoAssinador(segredo string) (*Assinador, error) {
	if segredo == "" {
		return nil, errors.New("codigo: segredo vazio")
	}
	return &Assinador{segredo: []byte(segredo)}, nil
}

func (a *Assinador) Gerar(ingressoID string) string {
	corpo := Prefixo + "." + b64.EncodeToString([]byte(ingressoID))
	return corpo + "." + b64.EncodeToString(a.mac(corpo))
}

func (a *Assinador) Verificar(codigo string) (string, error) {
	if len(codigo) == 0 || len(codigo) > 255 {
		return "", ErrInvalido
	}
	partes := strings.Split(codigo, ".")
	if len(partes) != 3 || partes[0] != Prefixo {
		return "", ErrInvalido
	}

	id, err := b64.DecodeString(partes[1])
	if err != nil || len(id) == 0 {
		return "", ErrInvalido
	}
	assinatura, err := b64.DecodeString(partes[2])
	if err != nil {
		return "", ErrInvalido
	}

	corpo := partes[0] + "." + partes[1]
	if subtle.ConstantTimeCompare(assinatura, a.mac(corpo)) != 1 {
		return "", ErrInvalido
	}
	return string(id), nil
}

func (a *Assinador) mac(corpo string) []byte {
	m := hmac.New(sha256.New, a.segredo)
	m.Write([]byte(corpo))
	return m.Sum(nil)
}
