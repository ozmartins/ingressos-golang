package http

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/oseias/ingressos-golang/catalogo/internal/adapter/http/middleware"
	"github.com/oseias/ingressos-golang/catalogo/internal/domain/catalogo"
	"github.com/oseias/ingressos-golang/catalogo/internal/domain/reserva"
	"github.com/oseias/ingressos-golang/catalogo/internal/domain/shared"
	"github.com/oseias/ingressos-golang/catalogo/internal/usecase"
)

// Handlers traduz requisições em casos de uso. Nenhuma regra de negócio aqui:
// handler fino é o que mantém a regra testável sem servidor HTTP.
type Handlers struct {
	ListarFilmes      usecase.ListarFilmes
	ListarCinemas     usecase.ListarCinemas
	ListarSalas       usecase.ListarSalas
	ConsultarSessoes  usecase.ConsultarSessoes
	ReservarPoltronas usecase.ReservarPoltronas
	Limites           LimitesPaginacao
}

func escreverJSON(w http.ResponseWriter, codigo int, corpo any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(codigo)
	_ = json.NewEncoder(w).Encode(corpo)
}

func (h Handlers) GetFilmes(w http.ResponseWriter, r *http.Request) {
	req, err := lerPaginacao(r, h.Limites)
	if err != nil {
		EscreverErroDeDominio(w, r, err, "")
		return
	}

	var filtro usecase.FiltroFilmes
	if bruto := r.URL.Query().Get("status"); bruto != "" {
		s, err := catalogo.ParseStatusFilme(bruto)
		if err != nil {
			EscreverErroDeDominio(w, r, err, "")
			return
		}
		filtro.Status = &s
	}

	pagina, err := h.ListarFilmes.Executar(r.Context(), filtro, req)
	if err != nil {
		EscreverErroDeDominio(w, r, err, "filme")
		return
	}
	escreverJSON(w, http.StatusOK, envelope(pagina, paraFilmeDTO))
}

func (h Handlers) GetCinemas(w http.ResponseWriter, r *http.Request) {
	req, err := lerPaginacao(r, h.Limites)
	if err != nil {
		EscreverErroDeDominio(w, r, err, "")
		return
	}
	pagina, err := h.ListarCinemas.Executar(r.Context(), req)
	if err != nil {
		EscreverErroDeDominio(w, r, err, "cinema")
		return
	}
	escreverJSON(w, http.StatusOK, envelope(pagina, paraCinemaDTO))
}

func (h Handlers) GetSalasDoCinema(w http.ResponseWriter, r *http.Request) {
	cinemaID := r.PathValue("id")
	if err := validarUUID(cinemaID, "id"); err != nil {
		EscreverErroDeDominio(w, r, err, "")
		return
	}
	req, err := lerPaginacao(r, h.Limites)
	if err != nil {
		EscreverErroDeDominio(w, r, err, "")
		return
	}
	pagina, err := h.ListarSalas.Executar(r.Context(), cinemaID, req)
	if err != nil {
		EscreverErroDeDominio(w, r, err, "cinema")
		return
	}
	escreverJSON(w, http.StatusOK, envelope(pagina, paraSalaDTO))
}

func (h Handlers) GetSessoes(w http.ResponseWriter, r *http.Request) {
	req, err := lerPaginacao(r, h.Limites)
	if err != nil {
		EscreverErroDeDominio(w, r, err, "")
		return
	}

	var filtro usecase.FiltroSessoes
	q := r.URL.Query()
	if v := q.Get("filme_id"); v != "" {
		if err := validarUUID(v, "filme_id"); err != nil {
			EscreverErroDeDominio(w, r, err, "")
			return
		}
		filtro.FilmeID = v
	}
	if v := q.Get("cinema_id"); v != "" {
		if err := validarUUID(v, "cinema_id"); err != nil {
			EscreverErroDeDominio(w, r, err, "")
			return
		}
		filtro.CinemaID = v
	}
	if v := q.Get("data"); v != "" {
		d, err := parseData(v)
		if err != nil {
			EscreverErroDeDominio(w, r, err, "")
			return
		}
		filtro.Data = d
	}

	pagina, err := h.ConsultarSessoes.Executar(r.Context(), filtro, req)
	if err != nil {
		EscreverErroDeDominio(w, r, err, "sessao")
		return
	}
	escreverJSON(w, http.StatusOK, envelope(pagina, paraSessaoDTO))
}

func (h Handlers) PostReservar(w http.ResponseWriter, r *http.Request) {
	sessaoID := r.PathValue("id")
	if err := validarUUID(sessaoID, "id"); err != nil {
		EscreverErroDeDominio(w, r, err, "")
		return
	}

	var corpo solicitacaoReservaDTO
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64*1024))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&corpo); err != nil {
		EscreverProblem(w, r, catCorpoInvalido, "Corpo da requisição não é um JSON válido para esta operação.")
		return
	}

	solicitacao := reserva.SolicitacaoReserva{
		SessaoID:     sessaoID,
		PoltronasIDs: corpo.PoltronasIDs,
		UsuarioID:    middleware.UsuarioDoContexto(r.Context()),
	}

	resultado, err := h.ReservarPoltronas.Executar(r.Context(), solicitacao)
	if err != nil {
		// Validação da solicitação é problema de corpo, não de parâmetro.
		if errors.Is(err, shared.ErrValidacao) {
			EscreverProblem(w, r, catCorpoInvalido, mensagemLimpa(err))
			return
		}
		EscreverErroDeDominio(w, r, err, "sessao")
		return
	}
	escreverJSON(w, http.StatusCreated, paraReservaDTO(resultado))
}

// validarUUID checa a forma antes de ir ao banco: identificador malformado é
// entrada inválida (400), não recurso ausente (404).
func validarUUID(v, campo string) error {
	if len(v) != 36 {
		return fmt.Errorf("%w: %s deve ser um UUID", shared.ErrValidacao, campo)
	}
	for i, c := range v {
		switch i {
		case 8, 13, 18, 23:
			if c != '-' {
				return fmt.Errorf("%w: %s deve ser um UUID", shared.ErrValidacao, campo)
			}
		default:
			if !strings.ContainsRune("0123456789abcdefABCDEF", c) {
				return fmt.Errorf("%w: %s deve ser um UUID", shared.ErrValidacao, campo)
			}
		}
	}
	return nil
}

func parseData(v string) (*usecase.DataDoDia, error) {
	t, err := time.Parse("2006-01-02", v)
	if err != nil {
		return nil, fmt.Errorf("%w: data deve estar no formato YYYY-MM-DD", shared.ErrValidacao)
	}
	// time.Parse aceita 2026-02-31 normalizando para março; a comparação textual
	// rejeita datas que não existem.
	if t.Format("2006-01-02") != v {
		return nil, fmt.Errorf("%w: data %q não existe no calendário", shared.ErrValidacao, v)
	}
	return &usecase.DataDoDia{Ano: t.Year(), Mes: int(t.Month()), Dia: t.Day()}, nil
}
