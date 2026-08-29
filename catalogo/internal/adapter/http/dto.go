package http

import (
	"time"

	"github.com/oseias/ingressos-golang/catalogo/internal/domain/catalogo"
	"github.com/oseias/ingressos-golang/catalogo/internal/domain/reserva"
	"github.com/oseias/ingressos-golang/catalogo/internal/domain/shared"
)

// paginacaoDTO acompanha toda coleção. Total e tem_proxima existem para o
// cliente navegar sem adivinhar (FR-003).
type paginacaoDTO struct {
	Pagina     int  `json:"pagina"`
	Tamanho    int  `json:"tamanho"`
	Total      int  `json:"total"`
	TemProxima bool `json:"tem_proxima"`
}

// paginaDTO é o envelope de toda resposta de coleção.
//
// Diverge da ERS original, que mostrava arrays nus: um array não comporta o
// total nem a indicação de próxima página.
type paginaDTO[T any] struct {
	Itens  []T          `json:"itens"`
	Pagina paginacaoDTO `json:"pagina"`
}

func envelope[D any, E any](p shared.Page[E], converter func(E) D) paginaDTO[D] {
	itens := make([]D, 0, len(p.Itens))
	for _, e := range p.Itens {
		itens = append(itens, converter(e))
	}
	return paginaDTO[D]{
		Itens: itens,
		Pagina: paginacaoDTO{
			Pagina:     p.Numero,
			Tamanho:    p.Tamanho,
			Total:      p.Total,
			TemProxima: p.TemProxima,
		},
	}
}

type filmeDTO struct {
	ID                  string  `json:"id"`
	Titulo              string  `json:"titulo"`
	Sinopse             *string `json:"sinopse,omitempty"`
	DuracaoMinutos      int     `json:"duracao_minutos"`
	ClassificacaoEtaria string  `json:"classificacao_etaria"`
	Genero              string  `json:"genero"`
	ImagemURL           *string `json:"imagem_url,omitempty"`
	Status              string  `json:"status"`
}

func paraFilmeDTO(f catalogo.Filme) filmeDTO {
	return filmeDTO{
		ID: f.ID, Titulo: f.Titulo, Sinopse: f.Sinopse,
		DuracaoMinutos: f.DuracaoMinutos, ClassificacaoEtaria: f.ClassificacaoEtaria,
		Genero: f.Genero, ImagemURL: f.ImagemURL, Status: string(f.Status),
	}
}

type cinemaDTO struct {
	ID       string `json:"id"`
	Nome     string `json:"nome"`
	Cidade   string `json:"cidade"`
	Estado   string `json:"estado"`
	Endereco string `json:"endereco"`
}

func paraCinemaDTO(c catalogo.Cinema) cinemaDTO {
	return cinemaDTO{ID: c.ID, Nome: c.Nome, Cidade: c.Cidade, Estado: c.Estado, Endereco: c.Endereco}
}

type salaDTO struct {
	ID              string `json:"id"`
	CinemaID        string `json:"cinema_id"`
	Numero          int    `json:"numero"`
	TipoTela        string `json:"tipo_tela"`
	CapacidadeTotal int    `json:"capacidade_total"`
}

func paraSalaDTO(s catalogo.Sala) salaDTO {
	return salaDTO{ID: s.ID, CinemaID: s.CinemaID, Numero: s.Numero,
		TipoTela: string(s.TipoTela), CapacidadeTotal: s.CapacidadeTotal}
}

type sessaoDTO struct {
	ID             string `json:"id"`
	FilmeID        string `json:"filme_id"`
	FilmeTitulo    string `json:"filme_titulo"`
	CinemaID       string `json:"cinema_id"`
	CinemaNome     string `json:"cinema_nome"`
	SalaNumero     int    `json:"sala_numero"`
	TipoTela       string `json:"tipo_tela"`
	DataHoraInicio string `json:"data_hora_inicio"`
	Idioma         string `json:"idioma"`
	// PrecoBase é texto para preservar a exatidão decimal: em JSON, número vira
	// float64 no cliente e 42.00 deixa de ser 42.00.
	PrecoBase string `json:"preco_base"`
}

func paraSessaoDTO(s catalogo.SessaoDetalhada) sessaoDTO {
	return sessaoDTO{
		ID: s.ID, FilmeID: s.FilmeID, FilmeTitulo: s.FilmeTitulo,
		CinemaID: s.CinemaID, CinemaNome: s.CinemaNome, SalaNumero: s.SalaNumero,
		TipoTela: string(s.TipoTela), DataHoraInicio: s.DataHoraInicio.UTC().Format(time.RFC3339),
		Idioma: string(s.Idioma), PrecoBase: s.PrecoBase.String(),
	}
}

type solicitacaoReservaDTO struct {
	PoltronasIDs []string `json:"poltronas_ids"`
}

type reservaConfirmadaDTO struct {
	ReservaID string `json:"reserva_id"`
	ExpiraEm  string `json:"expira_em"`
}

func paraReservaDTO(r reserva.ResultadoReserva) reservaConfirmadaDTO {
	return reservaConfirmadaDTO{ReservaID: r.ReservaID, ExpiraEm: r.ExpiraEm.UTC().Format(time.RFC3339)}
}
