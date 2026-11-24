package health

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSondarURLAceitaSo2xx(t *testing.T) {
	status := http.StatusOK
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(status) }))
	defer srv.Close()

	if err := SondarURL(srv.URL); err != nil {
		t.Fatalf("200 deveria passar: %v", err)
	}
	status = http.StatusServiceUnavailable
	if err := SondarURL(srv.URL); err == nil {
		t.Fatal("503 deveria falhar")
	}
	srv.Close()
	if err := SondarURL(srv.URL); err == nil {
		t.Fatal("conexão recusada deveria falhar")
	}
}
