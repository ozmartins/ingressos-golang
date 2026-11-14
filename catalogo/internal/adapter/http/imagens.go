package http

import (
	"net/http"
	"os"
	"strings"
)

// HandlerImagens serve os arquivos de imagem do diretório `dir` sob /imagens/.
// É conteúdo estático e público (pôsteres e logos que as URLs do catálogo
// apontam). Listagem de diretório não é oferecida: só se serve arquivo.
func HandlerImagens(dir string) http.Handler {
	servidor := http.StripPrefix("/imagens/", http.FileServer(semListagem{http.Dir(dir)}))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=86400")
		servidor.ServeHTTP(w, r)
	})
}

type semListagem struct{ fs http.FileSystem }

func (s semListagem) Open(nome string) (http.File, error) {
	if strings.HasSuffix(nome, "/") {
		return nil, os.ErrNotExist
	}
	f, err := s.fs.Open(nome)
	if err != nil {
		return nil, err
	}
	info, err := f.Stat()
	if err != nil || info.IsDir() {
		_ = f.Close()
		return nil, os.ErrNotExist
	}
	return f, nil
}
