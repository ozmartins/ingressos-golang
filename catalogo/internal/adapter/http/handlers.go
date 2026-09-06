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

type Handlers struct {
	ListarFilmes      usecase.ListarFilmes
	BuscarFilme       usecase.BuscarFilme
	CriarFilme        usecase.CriarFilme
	AtualizarFilme    usecase.AtualizarFilme
	RemoverFilme      usecase.RemoverFilme
	ListarCinemas     usecase.ListarCinemas
	BuscarCinema      usecase.BuscarCinema
	CriarCinema       usecase.CriarCinema
	AtualizarCinema   usecase.AtualizarCinema
	RemoverCinema     usecase.RemoverCinema
	ListarSalas       usecase.ListarSalas
	BuscarSala        usecase.BuscarSala
	CriarSala         usecase.CriarSala
	AtualizarSala     usecase.AtualizarSala
	RemoverSala       usecase.RemoverSala
	ConsultarSessoes  usecase.ConsultarSessoes
	BuscarSessao      usecase.BuscarSessao
	CriarSessao       usecase.CriarSessao
	AtualizarSessao   usecase.AtualizarSessao
	RemoverSessao     usecase.RemoverSessao
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

func (h Handlers) GetFilme(w http.ResponseWriter, r *http.Request) {
	filmeID := r.PathValue("id")
	if err := validarUUID(filmeID, "id"); err != nil {
		EscreverErroDeDominio(w, r, err, "")
		return
	}
	filme, err := h.BuscarFilme.Executar(r.Context(), filmeID)
	if err != nil {
		EscreverErroDeDominio(w, r, err, "filme")
		return
	}
	escreverJSON(w, http.StatusOK, paraFilmeDTO(filme))
}

func (h Handlers) PostFilme(w http.ResponseWriter, r *http.Request) {
	corpo, ok := lerEntradaDeFilme(w, r)
	if !ok {
		return
	}
	filme, err := h.CriarFilme.Executar(r.Context(), corpo.paraDadosFilme())
	if err != nil {
		escreverErroDeEscritaDeFilme(w, r, err)
		return
	}
	w.Header().Set("Location", "/api/v1/filmes/"+filme.ID)
	escreverJSON(w, http.StatusCreated, paraFilmeDTO(filme))
}

func (h Handlers) PutFilme(w http.ResponseWriter, r *http.Request) {
	filmeID := r.PathValue("id")
	if err := validarUUID(filmeID, "id"); err != nil {
		EscreverErroDeDominio(w, r, err, "")
		return
	}
	corpo, ok := lerEntradaDeFilme(w, r)
	if !ok {
		return
	}
	filme, err := h.AtualizarFilme.Executar(r.Context(), filmeID, corpo.paraDadosFilme())
	if err != nil {
		escreverErroDeEscritaDeFilme(w, r, err)
		return
	}
	escreverJSON(w, http.StatusOK, paraFilmeDTO(filme))
}

func (h Handlers) DeleteFilme(w http.ResponseWriter, r *http.Request) {
	filmeID := r.PathValue("id")
	if err := validarUUID(filmeID, "id"); err != nil {
		EscreverErroDeDominio(w, r, err, "")
		return
	}
	if err := h.RemoverFilme.Executar(r.Context(), filmeID); err != nil {
		EscreverErroDeDominio(w, r, err, "filme")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func lerEntradaDeFilme(w http.ResponseWriter, r *http.Request) (filmeEntradaDTO, bool) {
	var corpo filmeEntradaDTO
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64*1024))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&corpo); err != nil {
		EscreverProblem(w, r, catCorpoInvalido, "Corpo da requisição não é um JSON válido para esta operação.")
		return filmeEntradaDTO{}, false
	}
	return corpo, true
}

// Na escrita, entrada inválida veio do corpo, não da URL: o problema é
// `corpo-invalido`, como em PostReservar.
func escreverErroDeEscritaDeFilme(w http.ResponseWriter, r *http.Request, err error) {
	if errors.Is(err, shared.ErrValidacao) {
		EscreverProblem(w, r, catCorpoInvalido, mensagemLimpa(err))
		return
	}
	EscreverErroDeDominio(w, r, err, "filme")
}

func (h Handlers) GetCinemas(w http.ResponseWriter, r *http.Request) {
	req, err := lerPaginacao(r, h.Limites)
	if err != nil {
		EscreverErroDeDominio(w, r, err, "")
		return
	}

	var filtro usecase.FiltroCinemas
	if bruto := r.URL.Query().Get("ativo"); bruto != "" {
		ativo, err := parseBooleano(bruto, "ativo")
		if err != nil {
			EscreverErroDeDominio(w, r, err, "")
			return
		}
		filtro.Ativo = ativo
	}

	pagina, err := h.ListarCinemas.Executar(r.Context(), filtro, req)
	if err != nil {
		EscreverErroDeDominio(w, r, err, "cinema")
		return
	}
	escreverJSON(w, http.StatusOK, envelope(pagina, paraCinemaDTO))
}

func (h Handlers) GetCinema(w http.ResponseWriter, r *http.Request) {
	cinemaID := r.PathValue("id")
	if err := validarUUID(cinemaID, "id"); err != nil {
		EscreverErroDeDominio(w, r, err, "")
		return
	}
	cinema, err := h.BuscarCinema.Executar(r.Context(), cinemaID)
	if err != nil {
		EscreverErroDeDominio(w, r, err, "cinema")
		return
	}
	escreverJSON(w, http.StatusOK, paraCinemaDTO(cinema))
}

func (h Handlers) PostCinema(w http.ResponseWriter, r *http.Request) {
	corpo, ok := lerEntradaDeCinema(w, r)
	if !ok {
		return
	}
	cinema, err := h.CriarCinema.Executar(r.Context(), corpo.paraDadosCinema())
	if err != nil {
		escreverErroDeEscritaDeCinema(w, r, err)
		return
	}
	w.Header().Set("Location", "/api/v1/cinemas/"+cinema.ID)
	escreverJSON(w, http.StatusCreated, paraCinemaDTO(cinema))
}

func (h Handlers) PutCinema(w http.ResponseWriter, r *http.Request) {
	cinemaID := r.PathValue("id")
	if err := validarUUID(cinemaID, "id"); err != nil {
		EscreverErroDeDominio(w, r, err, "")
		return
	}
	corpo, ok := lerEntradaDeCinema(w, r)
	if !ok {
		return
	}
	cinema, err := h.AtualizarCinema.Executar(r.Context(), cinemaID, corpo.paraDadosCinema())
	if err != nil {
		escreverErroDeEscritaDeCinema(w, r, err)
		return
	}
	escreverJSON(w, http.StatusOK, paraCinemaDTO(cinema))
}

func (h Handlers) DeleteCinema(w http.ResponseWriter, r *http.Request) {
	cinemaID := r.PathValue("id")
	if err := validarUUID(cinemaID, "id"); err != nil {
		EscreverErroDeDominio(w, r, err, "")
		return
	}
	if err := h.RemoverCinema.Executar(r.Context(), cinemaID); err != nil {
		EscreverErroDeDominio(w, r, err, "cinema")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func lerEntradaDeCinema(w http.ResponseWriter, r *http.Request) (cinemaEntradaDTO, bool) {
	var corpo cinemaEntradaDTO
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64*1024))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&corpo); err != nil {
		EscreverProblem(w, r, catCorpoInvalido, "Corpo da requisição não é um JSON válido para esta operação.")
		return cinemaEntradaDTO{}, false
	}
	return corpo, true
}

func escreverErroDeEscritaDeCinema(w http.ResponseWriter, r *http.Request, err error) {
	if errors.Is(err, shared.ErrValidacao) {
		EscreverProblem(w, r, catCorpoInvalido, mensagemLimpa(err))
		return
	}
	EscreverErroDeDominio(w, r, err, "cinema")
}

func (h Handlers) GetSalas(w http.ResponseWriter, r *http.Request) {
	req, err := lerPaginacao(r, h.Limites)
	if err != nil {
		EscreverErroDeDominio(w, r, err, "")
		return
	}

	var filtro usecase.FiltroSalas
	q := r.URL.Query()
	if v := q.Get("cinema_id"); v != "" {
		if err := validarUUID(v, "cinema_id"); err != nil {
			EscreverErroDeDominio(w, r, err, "")
			return
		}
		filtro.CinemaID = v
	}
	if bruto := q.Get("ativo"); bruto != "" {
		ativo, err := parseBooleano(bruto, "ativo")
		if err != nil {
			EscreverErroDeDominio(w, r, err, "")
			return
		}
		filtro.Ativo = ativo
	}

	pagina, err := h.ListarSalas.Executar(r.Context(), filtro, req)
	if err != nil {
		EscreverErroDeDominio(w, r, err, "cinema")
		return
	}
	escreverJSON(w, http.StatusOK, envelope(pagina, paraSalaDTO))
}

func (h Handlers) GetSala(w http.ResponseWriter, r *http.Request) {
	salaID := r.PathValue("id")
	if err := validarUUID(salaID, "id"); err != nil {
		EscreverErroDeDominio(w, r, err, "")
		return
	}
	sala, err := h.BuscarSala.Executar(r.Context(), salaID)
	if err != nil {
		EscreverErroDeDominio(w, r, err, "sala")
		return
	}
	escreverJSON(w, http.StatusOK, paraSalaDTO(sala))
}

func (h Handlers) PostSala(w http.ResponseWriter, r *http.Request) {
	dados, ok := lerEntradaDeSala(w, r)
	if !ok {
		return
	}
	sala, err := h.CriarSala.Executar(r.Context(), dados)
	if err != nil {
		escreverErroDeEscritaDeSala(w, r, err)
		return
	}
	w.Header().Set("Location", "/api/v1/salas/"+sala.ID)
	escreverJSON(w, http.StatusCreated, paraSalaDTO(sala))
}

func (h Handlers) PutSala(w http.ResponseWriter, r *http.Request) {
	salaID := r.PathValue("id")
	if err := validarUUID(salaID, "id"); err != nil {
		EscreverErroDeDominio(w, r, err, "")
		return
	}
	dados, ok := lerEntradaDeSala(w, r)
	if !ok {
		return
	}
	sala, err := h.AtualizarSala.Executar(r.Context(), salaID, dados)
	if err != nil {
		escreverErroDeEscritaDeSala(w, r, err)
		return
	}
	escreverJSON(w, http.StatusOK, paraSalaDTO(sala))
}

func (h Handlers) DeleteSala(w http.ResponseWriter, r *http.Request) {
	salaID := r.PathValue("id")
	if err := validarUUID(salaID, "id"); err != nil {
		EscreverErroDeDominio(w, r, err, "")
		return
	}
	if err := h.RemoverSala.Executar(r.Context(), salaID); err != nil {
		EscreverErroDeDominio(w, r, err, "sala")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func lerEntradaDeSala(w http.ResponseWriter, r *http.Request) (catalogo.DadosSala, bool) {
	var corpo salaEntradaDTO
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64*1024))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&corpo); err != nil {
		EscreverProblem(w, r, catCorpoInvalido, "Corpo da requisição não é um JSON válido para esta operação.")
		return catalogo.DadosSala{}, false
	}
	dados, err := corpo.paraDadosSala()
	if err != nil {
		EscreverProblem(w, r, catCorpoInvalido, mensagemLimpa(err))
		return catalogo.DadosSala{}, false
	}
	return dados, true
}

func escreverErroDeEscritaDeSala(w http.ResponseWriter, r *http.Request, err error) {
	if errors.Is(err, shared.ErrValidacao) {
		EscreverProblem(w, r, catCorpoInvalido, mensagemLimpa(err))
		return
	}
	EscreverErroDeDominio(w, r, err, "sala")
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

func (h Handlers) GetSessao(w http.ResponseWriter, r *http.Request) {
	sessaoID := r.PathValue("id")
	if err := validarUUID(sessaoID, "id"); err != nil {
		EscreverErroDeDominio(w, r, err, "")
		return
	}
	sessao, err := h.BuscarSessao.Executar(r.Context(), sessaoID)
	if err != nil {
		EscreverErroDeDominio(w, r, err, "sessao")
		return
	}
	escreverJSON(w, http.StatusOK, paraSessaoRecursoDTO(sessao))
}

func (h Handlers) PostSessao(w http.ResponseWriter, r *http.Request) {
	dados, ok := lerEntradaDeSessao(w, r)
	if !ok {
		return
	}
	sessao, err := h.CriarSessao.Executar(r.Context(), dados)
	if err != nil {
		escreverErroDeEscritaDeSessao(w, r, err)
		return
	}
	w.Header().Set("Location", "/api/v1/sessoes/"+sessao.ID)
	escreverJSON(w, http.StatusCreated, paraSessaoRecursoDTO(sessao))
}

func (h Handlers) PutSessao(w http.ResponseWriter, r *http.Request) {
	sessaoID := r.PathValue("id")
	if err := validarUUID(sessaoID, "id"); err != nil {
		EscreverErroDeDominio(w, r, err, "")
		return
	}
	dados, ok := lerEntradaDeSessao(w, r)
	if !ok {
		return
	}
	sessao, err := h.AtualizarSessao.Executar(r.Context(), sessaoID, dados)
	if err != nil {
		escreverErroDeEscritaDeSessao(w, r, err)
		return
	}
	escreverJSON(w, http.StatusOK, paraSessaoRecursoDTO(sessao))
}

func (h Handlers) DeleteSessao(w http.ResponseWriter, r *http.Request) {
	sessaoID := r.PathValue("id")
	if err := validarUUID(sessaoID, "id"); err != nil {
		EscreverErroDeDominio(w, r, err, "")
		return
	}
	if err := h.RemoverSessao.Executar(r.Context(), sessaoID); err != nil {
		EscreverErroDeDominio(w, r, err, "sessao")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Diferente das outras entradas, aqui a conversão pode falhar sozinha: o
// instante vem como texto, e um formato errado é problema de corpo.
func lerEntradaDeSessao(w http.ResponseWriter, r *http.Request) (catalogo.DadosSessao, bool) {
	var corpo sessaoEntradaDTO
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64*1024))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&corpo); err != nil {
		EscreverProblem(w, r, catCorpoInvalido, "Corpo da requisição não é um JSON válido para esta operação.")
		return catalogo.DadosSessao{}, false
	}
	dados, err := corpo.paraDadosSessao()
	if err != nil {
		EscreverProblem(w, r, catCorpoInvalido, mensagemLimpa(err))
		return catalogo.DadosSessao{}, false
	}
	return dados, true
}

func escreverErroDeEscritaDeSessao(w http.ResponseWriter, r *http.Request, err error) {
	if errors.Is(err, shared.ErrValidacao) {
		EscreverProblem(w, r, catCorpoInvalido, mensagemLimpa(err))
		return
	}
	EscreverErroDeDominio(w, r, err, "sessao")
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
		if errors.Is(err, shared.ErrValidacao) {
			EscreverProblem(w, r, catCorpoInvalido, mensagemLimpa(err))
			return
		}
		EscreverErroDeDominio(w, r, err, "sessao")
		return
	}
	escreverJSON(w, http.StatusCreated, paraReservaDTO(resultado))
}

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

// `strconv.ParseBool` aceitaria "1", "t" e "TRUE"; o contrato promete apenas
// `true` e `false`, e o erro precisa listar o que é aceito.
func parseBooleano(v, campo string) (*bool, error) {
	switch v {
	case "true":
		valor := true
		return &valor, nil
	case "false":
		valor := false
		return &valor, nil
	default:
		return nil, fmt.Errorf("%w: %s aceita apenas true ou false", shared.ErrValidacao, campo)
	}
}

func parseData(v string) (*usecase.DataDoDia, error) {
	t, err := time.Parse("2006-01-02", v)
	if err != nil {
		return nil, fmt.Errorf("%w: data deve estar no formato YYYY-MM-DD", shared.ErrValidacao)
	}
	if t.Format("2006-01-02") != v {
		return nil, fmt.Errorf("%w: data %q não existe no calendário", shared.ErrValidacao, v)
	}
	return &usecase.DataDoDia{Ano: t.Year(), Mes: int(t.Month()), Dia: t.Day()}, nil
}
