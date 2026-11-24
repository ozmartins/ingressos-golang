package health

import (
	"fmt"
	"net/http"
	"time"
)

// SondarURL faz um GET na URL e devolve erro se a resposta não for 2xx. Alimenta o
// subcomando `healthcheck` do binário: a imagem é distroless (sem shell, curl
// nem wget), então o HEALTHCHECK do compose chama o próprio serviço.
func SondarURL(url string) error {
	cliente := &http.Client{Timeout: 3 * time.Second}
	resp, err := cliente.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("%s respondeu %d", url, resp.StatusCode)
	}
	return nil
}
