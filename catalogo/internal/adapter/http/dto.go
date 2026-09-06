package http

import (
	"time"

	"github.com/oseias/ingressos-golang/catalogo/internal/domain/catalogo"
	"github.com/oseias/ingressos-golang/catalogo/internal/domain/reserva"
	"github.com/oseias/ingressos-golang/catalogo/internal/domain/shared"
)

type paginacaoDTO struct {
	Pagina     int  `json:"pagina"`
	Tamanho    int  `json:"tamanho"`
	Total      int  `json:"total"`
	TemProxima bool `json:"tem_proxima"`
}

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

// Campos ponteiro para separar "ausente" de "vazio": no PUT, que substitui o
// filme inteiro, um `duracao_minutos` omitido é erro, não zero.
type filmeEntradaDTO struct {
	Titulo              *string `json:"titulo"`
	Sinopse             *string `json:"sinopse"`
	DuracaoMinutos      *int    `json:"duracao_minutos"`
	ClassificacaoEtaria *string `json:"classificacao_etaria"`
	Genero              *string `json:"genero"`
	ImagemURL           *string `json:"imagem_url"`
	Status              *string `json:"status"`
}

func (d filmeEntradaDTO) paraDadosFilme() catalogo.DadosFilme {
	dados := catalogo.DadosFilme{Sinopse: d.Sinopse, ImagemURL: d.ImagemURL}
	if d.Titulo != nil {
		dados.Titulo = *d.Titulo
	}
	if d.DuracaoMinutos != nil {
		dados.DuracaoMinutos = *d.DuracaoMinutos
	}
	if d.ClassificacaoEtaria != nil {
		dados.ClassificacaoEtaria = *d.ClassificacaoEtaria
	}
	if d.Genero != nil {
		dados.Genero = *d.Genero
	}
	if d.Status != nil {
		dados.Status = *d.Status
	}
	return dados
}

type cinemaDTO struct {
	ID       string `json:"id"`
	Nome     string `json:"nome"`
	Cidade   string `json:"cidade"`
	Estado   string `json:"estado"`
	Endereco string `json:"endereco"`
	Ativo    bool   `json:"ativo"`
}

func paraCinemaDTO(c catalogo.Cinema) cinemaDTO {
	return cinemaDTO{ID: c.ID, Nome: c.Nome, Cidade: c.Cidade, Estado: c.Estado,
		Endereco: c.Endereco, Ativo: c.Ativo}
}

// Campos ponteiro pelo mesmo motivo de `filmeEntradaDTO`: no PUT, que substitui
// o cinema inteiro, um `nome` omitido é erro, não string vazia.
type cinemaEntradaDTO struct {
	Nome     *string `json:"nome"`
	Cidade   *string `json:"cidade"`
	Estado   *string `json:"estado"`
	Endereco *string `json:"endereco"`
	Ativo    *bool   `json:"ativo"`
}

func (d cinemaEntradaDTO) paraDadosCinema() catalogo.DadosCinema {
	dados := catalogo.DadosCinema{Ativo: d.Ativo}
	if d.Nome != nil {
		dados.Nome = *d.Nome
	}
	if d.Cidade != nil {
		dados.Cidade = *d.Cidade
	}
	if d.Estado != nil {
		dados.Estado = *d.Estado
	}
	if d.Endereco != nil {
		dados.Endereco = *d.Endereco
	}
	return dados
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
	PrecoBase      string `json:"preco_base"`
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
