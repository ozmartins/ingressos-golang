import { describe, expect, it } from 'vitest'
import { TABELA, buscarDefinicao } from './tabela'

describe('tabela de comandos', () => {
  it('tem exatamente 21 endpoints', () => {
    expect(TABELA).toHaveLength(21)
  })

  it('não repete (controller, ação)', () => {
    const chaves = TABELA.map((d) => `${d.controller} ${d.acao}`)
    expect(new Set(chaves).size).toBe(21)
  })

  it('não repete (método, caminho)', () => {
    const chaves = TABELA.map((d) => `${d.metodo} ${d.raiz ? 'raiz' : 'api'}${d.caminho}`)
    expect(new Set(chaves).size).toBe(21)
  })

  it('só health fica na raiz', () => {
    const raiz = TABELA.filter((d) => d.raiz)
    expect(raiz.map((d) => d.acao)).toEqual(['health'])
  })

  it('escritas exigem token e leituras não', () => {
    for (const d of TABELA) {
      expect(d.protegido).toBe(['POST', 'PUT', 'DELETE'].includes(d.metodo))
    }
  })

  it('cada controller tem as 5 ações do CRUD', () => {
    for (const c of ['filmes', 'cinemas', 'salas', 'sessoes']) {
      const acoes = TABELA.filter((d) => d.controller === c).map((d) => d.acao)
      expect(acoes.sort()).toEqual(['atualizar', 'criar', 'listar', 'obter', 'remover'])
    }
  })

  it('listagens aceitam paginação e os filtros do endpoint', () => {
    expect(buscarDefinicao('filmes', 'listar')?.filtros).toEqual(['status', 'page', 'page_size'])
    expect(buscarDefinicao('salas', 'listar')?.filtros).toEqual(['cinema_id', 'ativo', 'page', 'page_size'])
    expect(buscarDefinicao('sessoes', 'listar')?.filtros).toEqual([
      'filme_id', 'cinema_id', 'data', 'page', 'page_size',
    ])
  })

  it('buscarDefinicao ignora maiúsculas e devolve undefined para desconhecidos', () => {
    expect(buscarDefinicao('FILMES', 'Obter')?.metodo).toBe('GET')
    expect(buscarDefinicao('filmes', 'xpto')).toBeUndefined()
  })
})
