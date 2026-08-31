// Package codigo gera e verifica o código de acesso impresso no QR Code.
//
// Formato (research.md D3, data-model.md §4):
//
//	CIN1.<base64url(id)>.<base64url(HMAC-SHA256(segredo, "CIN1."+base64url(id)))>
//
// O prefixo é marcador de formato — permite descartar lixo antes de qualquer
// trabalho criptográfico. NÃO é mecanismo de rotação de chave: rotação está
// fora do escopo desta feature.
package codigo

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"strings"
)

// Prefixo identifica o formato do código.
const Prefixo = "CIN1"

// ErrInvalido cobre toda recusa: prefixo errado, número de partes errado,
// base64 quebrada e assinatura que não confere. É uma só de propósito — quem
// chama não deve conseguir distinguir os casos, e a resposta da portaria
// tampouco (FR-010).
var ErrInvalido = errors.New("codigo: código de acesso inválido")

var b64 = base64.RawURLEncoding

// Assinador produz e confere códigos com um segredo que só este serviço conhece.
type Assinador struct{ segredo []byte }

// NovoAssinador exige segredo não vazio: subir sem ele emitiria ingressos que a
// portaria nunca validaria (research.md D11).
func NovoAssinador(segredo string) (*Assinador, error) {
	if segredo == "" {
		return nil, errors.New("codigo: segredo vazio")
	}
	return &Assinador{segredo: []byte(segredo)}, nil
}

// Gerar devolve o código de acesso de um ingresso. O único dado transportado é
// o identificador opaco do próprio ingresso — nada da pessoa, da reserva ou da
// compra (FR-005).
func (a *Assinador) Gerar(ingressoID string) string {
	corpo := Prefixo + "." + b64.EncodeToString([]byte(ingressoID))
	return corpo + "." + b64.EncodeToString(a.mac(corpo))
}

// Verificar confere a autenticidade e devolve o identificador do ingresso.
// Não consulta o acervo: um código forjado é recusado antes de virar consulta
// (FR-006, FR-010).
func (a *Assinador) Verificar(codigo string) (string, error) {
	// Teto barato contra entrada absurda, antes de qualquer alocação maior.
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
	// Comparação em tempo constante: comparação ingênua vazaria o prefixo
	// correto pelo tempo de resposta.
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
