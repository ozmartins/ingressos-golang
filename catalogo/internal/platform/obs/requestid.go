package obs

import (
	"context"
	"crypto/rand"
	"encoding/hex"
)

// HeaderRequestID é o cabeçalho HTTP do identificador de correlação. Gerado na
// borda (ou aceito do cliente, se bem formado), ele acompanha a requisição pelo
// contexto, por metadados gRPC (x-request-id) e por headers AMQP (x-request-id),
// e aparece como request_id em todo log escrito com contexto.
const HeaderRequestID = "X-Request-ID"

// ChaveRequestID é o nome do header AMQP e do metadado gRPC (minúsculo).
const ChaveRequestID = "x-request-id"

const tamanhoMaximoRequestID = 64

type chaveRequestID struct{}

// ComRequestID devolve um contexto que carrega o request id.
func ComRequestID(ctx context.Context, id string) context.Context {
	if id == "" {
		return ctx
	}
	return context.WithValue(ctx, chaveRequestID{}, id)
}

// RequestID devolve o request id do contexto ("" se não houver).
func RequestID(ctx context.Context) string {
	id, _ := ctx.Value(chaveRequestID{}).(string)
	return id
}

// NovoRequestID gera um identificador aleatório de 128 bits em hexadecimal.
func NovoRequestID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

// RequestIDAceito devolve o id recebido se for seguro de logar e propagar
// (1 a 64 caracteres de [A-Za-z0-9._-]); caso contrário, gera um novo. Evita
// que um cliente injete quebras de linha ou lixo nos logs.
func RequestIDAceito(recebido string) string {
	if n := len(recebido); n == 0 || n > tamanhoMaximoRequestID {
		return NovoRequestID()
	}
	for _, c := range recebido {
		ok := c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '.' || c == '_' || c == '-'
		if !ok {
			return NovoRequestID()
		}
	}
	return recebido
}
