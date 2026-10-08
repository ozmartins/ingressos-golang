import { chamar } from './cliente'
import type {
  Cinema, CinemaEntrada, Filme, FilmeEntrada, Pagina, ReservaConfirmada,
  Sala, SalaAtualizacao, SalaEntrada, Sessao, SessaoAtualizacao, SessaoEntrada,
  SessaoNaGrade, StatusFilme,
} from './tipos'

// Navegação é pública; escrita e reserva exigem token. Cada função declara o
// que precisa, em vez de um token global implícito.

export const listarFilmes = (consulta: { page?: number; page_size?: number; status?: StatusFilme } = {}) =>
  chamar<Pagina<Filme>>('catalogo', '/filmes', { consulta })

export const buscarFilme = (id: string) => chamar<Filme>('catalogo', `/filmes/${id}`)

export const criarFilme = (token: string, corpo: FilmeEntrada) =>
  chamar<Filme>('catalogo', '/filmes', { metodo: 'POST', corpo, token })

export const substituirFilme = (token: string, id: string, corpo: FilmeEntrada) =>
  chamar<Filme>('catalogo', `/filmes/${id}`, { metodo: 'PUT', corpo, token })

export const removerFilme = (token: string, id: string) =>
  chamar<void>('catalogo', `/filmes/${id}`, { metodo: 'DELETE', token })

export const listarCinemas = (consulta: { page?: number; page_size?: number } = {}) =>
  chamar<Pagina<Cinema>>('catalogo', '/cinemas', { consulta })

export const buscarCinema = (id: string) => chamar<Cinema>('catalogo', `/cinemas/${id}`)

export const criarCinema = (token: string, corpo: CinemaEntrada) =>
  chamar<Cinema>('catalogo', '/cinemas', { metodo: 'POST', corpo, token })

export const substituirCinema = (token: string, id: string, corpo: CinemaEntrada) =>
  chamar<Cinema>('catalogo', `/cinemas/${id}`, { metodo: 'PUT', corpo, token })

export const removerCinema = (token: string, id: string) =>
  chamar<void>('catalogo', `/cinemas/${id}`, { metodo: 'DELETE', token })

export const listarSalas = (consulta: { page?: number; page_size?: number; cinema_id?: string } = {}) =>
  chamar<Pagina<Sala>>('catalogo', '/salas', { consulta })

export const buscarSala = (id: string) => chamar<Sala>('catalogo', `/salas/${id}`)

export const criarSala = (token: string, corpo: SalaEntrada) =>
  chamar<Sala>('catalogo', '/salas', { metodo: 'POST', corpo, token })

// A planta congela no cadastro: a atualização não redesenha `fileiras`, e por
// isso tem corpo próprio em vez de reaproveitar `SalaEntrada`.
export const substituirSala = (token: string, id: string, corpo: SalaAtualizacao) =>
  chamar<Sala>('catalogo', `/salas/${id}`, { metodo: 'PUT', corpo, token })

export const removerSala = (token: string, id: string) =>
  chamar<void>('catalogo', `/salas/${id}`, { metodo: 'DELETE', token })

export const listarSessoes = (
  consulta: { page?: number; page_size?: number; filme_id?: string; cinema_id?: string; data?: string } = {},
) => chamar<Pagina<SessaoNaGrade>>('catalogo', '/sessoes', { consulta })

export const buscarSessao = (id: string) => chamar<Sessao>('catalogo', `/sessoes/${id}`)

export const criarSessao = (token: string, corpo: SessaoEntrada) =>
  chamar<Sessao>('catalogo', '/sessoes', { metodo: 'POST', corpo, token })

export const substituirSessao = (token: string, id: string, corpo: SessaoAtualizacao) =>
  chamar<Sessao>('catalogo', `/sessoes/${id}`, { metodo: 'PUT', corpo, token })

export const removerSessao = (token: string, id: string) =>
  chamar<void>('catalogo', `/sessoes/${id}`, { metodo: 'DELETE', token })

/** O caminho de compra: o catálogo calcula o valor e fala com o estoque por
 *  gRPC. A tela nunca chama o estoque para escrever. */
export const reservarPoltronas = (token: string, sessaoId: string, poltronas: string[]) =>
  chamar<ReservaConfirmada>('catalogo', `/sessoes/${sessaoId}/reservar`, {
    metodo: 'POST',
    corpo: { poltronas_ids: poltronas },
    token,
  })
