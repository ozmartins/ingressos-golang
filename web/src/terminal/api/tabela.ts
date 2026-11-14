export type Controller = 'filmes' | 'cinemas' | 'salas' | 'sessoes' | 'operacional'
export type Acao = 'listar' | 'obter' | 'criar' | 'atualizar' | 'remover' | 'health'
export type Metodo = 'GET' | 'POST' | 'PUT' | 'DELETE'

export interface Definicao {
  controller: Controller
  acao: Acao
  metodo: Metodo
  caminho: string
  raiz: boolean
  protegido: boolean
  filtros: string[]
  usaId: boolean
  usaCorpo: boolean
  uso: string
  descricao: string
}

export const NOMES_CONTROLLER: Record<Controller, string> = {
  filmes: 'Filmes',
  cinemas: 'Cinemas',
  salas: 'Salas',
  sessoes: 'Sessões',
  operacional: 'Operacional',
}

export const CONTROLLERS_CRUD: Controller[] = ['filmes', 'cinemas', 'salas', 'sessoes']

const PAGINACAO = ['page', 'page_size']
const USO_PAGINACAO = '[page=N] [page_size=N]'

interface Recurso {
  controller: Controller
  singular: string
  artigo: 'um' | 'uma'
  plural: string
  filtros: string[]
  usoFiltros: string
}

const RECURSOS: Recurso[] = [
  {
    controller: 'filmes',
    singular: 'filme',
    artigo: 'um',
    plural: 'os filmes',
    filtros: ['status'],
    usoFiltros: '[status=EM_CARTAZ|BREVE|FORA_DE_CARTAZ]',
  },
  {
    controller: 'cinemas',
    singular: 'cinema',
    artigo: 'um',
    plural: 'os cinemas',
    filtros: ['ativo'],
    usoFiltros: '[ativo=true|false]',
  },
  {
    controller: 'salas',
    singular: 'sala',
    artigo: 'uma',
    plural: 'as salas',
    filtros: ['cinema_id', 'ativo'],
    usoFiltros: '[cinema_id=<uuid>] [ativo=true|false]',
  },
  {
    controller: 'sessoes',
    singular: 'sessão',
    artigo: 'uma',
    plural: 'a grade de sessões',
    filtros: ['filme_id', 'cinema_id', 'data'],
    usoFiltros: '[filme_id=<uuid>] [cinema_id=<uuid>] [data=AAAA-MM-DD]',
  },
]

function definicoesDoRecurso(r: Recurso): Definicao[] {
  const base = { controller: r.controller, raiz: false, filtros: [] as string[] }
  const caminho = `/${r.controller}`
  const nome = r.controller
  return [
    {
      ...base,
      acao: 'listar',
      metodo: 'GET',
      caminho,
      protegido: false,
      filtros: [...r.filtros, ...PAGINACAO],
      usaId: false,
      usaCorpo: false,
      uso: `${nome} listar ${r.usoFiltros} ${USO_PAGINACAO}`,
      descricao: `lista ${r.plural}, com filtros e paginação`,
    },
    {
      ...base,
      acao: 'criar',
      metodo: 'POST',
      caminho,
      protegido: true,
      usaId: false,
      usaCorpo: true,
      uso: `${nome} criar <json>`,
      descricao: `cadastra ${r.artigo} ${r.singular} (exige login)`,
    },
    {
      ...base,
      acao: 'obter',
      metodo: 'GET',
      caminho: `${caminho}/{id}`,
      protegido: false,
      usaId: true,
      usaCorpo: false,
      uso: `${nome} obter <id>`,
      descricao: `mostra ${r.artigo} ${r.singular} pelo id`,
    },
    {
      ...base,
      acao: 'atualizar',
      metodo: 'PUT',
      caminho: `${caminho}/{id}`,
      protegido: true,
      usaId: true,
      usaCorpo: true,
      uso: `${nome} atualizar <id> <json>`,
      descricao: `atualiza ${r.artigo} ${r.singular} (exige login)`,
    },
    {
      ...base,
      acao: 'remover',
      metodo: 'DELETE',
      caminho: `${caminho}/{id}`,
      protegido: true,
      usaId: true,
      usaCorpo: false,
      uso: `${nome} remover <id>`,
      descricao: `remove ${r.artigo} ${r.singular} (exige login)`,
    },
  ]
}

export const TABELA: Definicao[] = [
  ...RECURSOS.flatMap(definicoesDoRecurso),
  {
    controller: 'operacional',
    acao: 'health',
    metodo: 'GET',
    caminho: '/health',
    raiz: true,
    protegido: false,
    filtros: [],
    usaId: false,
    usaCorpo: false,
    uso: 'health',
    descricao: 'mostra se o catálogo está apto a receber tráfego',
  },
]

export function buscarDefinicao(controller: string, acao: string): Definicao | undefined {
  const c = controller.toLowerCase()
  const a = acao.toLowerCase()
  return TABELA.find((d) => d.controller === c && d.acao === a)
}

export function definicoesDe(controller: Controller): Definicao[] {
  return TABELA.filter((d) => d.controller === controller)
}

export function ehControllerCrud(nome: string): nome is Controller {
  return (CONTROLLERS_CRUD as string[]).includes(nome.toLowerCase())
}
