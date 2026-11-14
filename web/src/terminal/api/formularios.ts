import type { Definicao } from './tabela'

interface Base {
  chave: string
  rotulo: string
  obrigatorio: boolean
  /** Valor padrão aplicado pelo serviço; só é exibido na pergunta. */
  valorPadrao?: string
}

export type CampoDef =
  | (Base & {
      tipo: 'texto'
      minTamanho?: number
      maxTamanho?: number
      maiusculas?: boolean
      formato?: RegExp
      formatoMsg?: string
    })
  | (Base & { tipo: 'inteiro'; minimo?: number })
  | (Base & { tipo: 'logico' })
  | (Base & { tipo: 'opcao'; opcoes: string[] })
  | (Base & { tipo: 'uuid' })
  | (Base & { tipo: 'decimal' })
  | (Base & { tipo: 'datahora' })
  | (Base & { tipo: 'fileiras' })

export type Conversao = { valor: unknown } | { erro: string }

export interface Fileira {
  fileira: string
  assentos: number
  tipo: string
}

const UUID = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i
const DECIMAL = /^\d+([.,]\d{1,2})?$/
const DATA_LOCAL = /^(\d{4})-(\d{2})-(\d{2})[ T](\d{2}):(\d{2})(?::(\d{2}))?$/
const DATA_ISO = /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}(:\d{2}(\.\d+)?)?(Z|[+-]\d{2}:\d{2})$/i

function datahora(t: string): Conversao {
  const m = DATA_LOCAL.exec(t)
  if (m) {
    const [ano, mes, dia, hora, min, seg] = m.slice(1).map((x) => (x === undefined ? 0 : Number(x)))
    const d = new Date(ano, mes - 1, dia, hora, min, seg)
    const valida =
      d.getFullYear() === ano && d.getMonth() === mes - 1 && d.getDate() === dia && hora < 24 && min < 60
    if (valida) return { valor: d.toISOString() }
  } else if (DATA_ISO.test(t)) {
    const d = new Date(t)
    if (!Number.isNaN(d.getTime())) return { valor: d.toISOString() }
  }
  return { erro: 'data/hora inválida (use AAAA-MM-DD HH:MM)' }
}

export function converter(campo: CampoDef, entrada: string): Conversao {
  const t = entrada.trim()
  if (t === '') {
    return campo.obrigatorio ? { erro: 'campo obrigatório' } : { valor: undefined }
  }
  switch (campo.tipo) {
    case 'texto': {
      const v = campo.maiusculas ? t.toUpperCase() : t
      if (campo.minTamanho !== undefined && v.length < campo.minTamanho) {
        return { erro: `mínimo de ${campo.minTamanho} caractere(s)` }
      }
      if (campo.maxTamanho !== undefined && v.length > campo.maxTamanho) {
        return { erro: `máximo de ${campo.maxTamanho} caracteres` }
      }
      if (campo.formato && !campo.formato.test(v)) {
        return { erro: campo.formatoMsg ?? 'formato inválido' }
      }
      return { valor: v }
    }
    case 'inteiro': {
      if (!/^-?\d+$/.test(t)) return { erro: 'informe um número inteiro' }
      const n = Number(t)
      if (campo.minimo !== undefined && n < campo.minimo) return { erro: `o mínimo é ${campo.minimo}` }
      return { valor: n }
    }
    case 'logico': {
      const v = t.toLowerCase()
      if (['s', 'sim', 'true', 'y', 'yes'].includes(v)) return { valor: true }
      if (['n', 'nao', 'não', 'false', 'no'].includes(v)) return { valor: false }
      return { erro: 'responda s ou n' }
    }
    case 'opcao': {
      const achada = campo.opcoes.find((o) => o.toUpperCase() === t.toUpperCase())
      return achada === undefined
        ? { erro: `opção inválida (use: ${campo.opcoes.join(', ')})` }
        : { valor: achada }
    }
    case 'uuid':
      return UUID.test(t) ? { valor: t.toLowerCase() } : { erro: 'identificador inválido (esperado um UUID)' }
    case 'decimal':
      return DECIMAL.test(t)
        ? { valor: t.replace(',', '.') }
        : { erro: 'valor inválido (use até 2 casas, ex.: 42.50)' }
    case 'datahora':
      return datahora(t)
    case 'fileiras':
      return { erro: 'fileiras são informadas passo a passo' }
  }
}

export function montarCorpo(
  campos: CampoDef[],
  respostas: Record<string, unknown>,
): Record<string, unknown> {
  const corpo: Record<string, unknown> = {}
  for (const c of campos) {
    const v = respostas[c.chave]
    if (v !== undefined) corpo[c.chave] = v
  }
  return corpo
}

export const TIPOS_FILEIRA = ['NORMAL', 'PCD', 'NAMORADEIRA']

export const CAMPO_FILEIRA_LETRA: CampoDef = {
  tipo: 'texto',
  chave: 'fileira',
  rotulo: 'Letra da fileira (A-Z, até 5 letras)',
  obrigatorio: true,
  maiusculas: true,
  formato: /^[A-Z]{1,5}$/,
  formatoMsg: 'use de 1 a 5 letras, de A a Z',
}
export const CAMPO_FILEIRA_ASSENTOS: CampoDef = {
  tipo: 'inteiro',
  chave: 'assentos',
  rotulo: 'Assentos da fileira',
  obrigatorio: true,
  minimo: 1,
}
export const CAMPO_FILEIRA_TIPO: CampoDef = {
  tipo: 'opcao',
  chave: 'tipo',
  rotulo: 'Tipo dos assentos',
  obrigatorio: false,
  valorPadrao: 'NORMAL',
  opcoes: TIPOS_FILEIRA,
}

const ATIVO: CampoDef = { tipo: 'logico', chave: 'ativo', rotulo: 'Ativo', obrigatorio: false, valorPadrao: 'true' }

const FILME: CampoDef[] = [
  { tipo: 'texto', chave: 'titulo', rotulo: 'Título', obrigatorio: true, minTamanho: 1, maxTamanho: 255 },
  { tipo: 'texto', chave: 'sinopse', rotulo: 'Sinopse', obrigatorio: false },
  { tipo: 'inteiro', chave: 'duracao_minutos', rotulo: 'Duração em minutos', obrigatorio: true, minimo: 1 },
  { tipo: 'texto', chave: 'classificacao_etaria', rotulo: 'Classificação etária', obrigatorio: true, maxTamanho: 50 },
  { tipo: 'texto', chave: 'genero', rotulo: 'Gênero', obrigatorio: true, maxTamanho: 100 },
  { tipo: 'texto', chave: 'imagem_url', rotulo: 'URL da imagem', obrigatorio: false, maxTamanho: 500 },
  {
    tipo: 'opcao',
    chave: 'status',
    rotulo: 'Status',
    obrigatorio: false,
    valorPadrao: 'EM_CARTAZ',
    opcoes: ['EM_CARTAZ', 'BREVE', 'FORA_DE_CARTAZ'],
  },
]

const CINEMA: CampoDef[] = [
  { tipo: 'texto', chave: 'nome', rotulo: 'Nome', obrigatorio: true, maxTamanho: 255 },
  { tipo: 'texto', chave: 'cidade', rotulo: 'Cidade', obrigatorio: true, maxTamanho: 100 },
  {
    tipo: 'texto',
    chave: 'estado',
    rotulo: 'Estado (sigla, 2 letras)',
    obrigatorio: true,
    minTamanho: 2,
    maxTamanho: 2,
    maiusculas: true,
    formato: /^[A-Z]{2}$/,
    formatoMsg: 'informe a sigla com 2 letras',
  },
  { tipo: 'texto', chave: 'endereco', rotulo: 'Endereço', obrigatorio: true },
  ATIVO,
]

const TIPO_TELA: CampoDef = {
  tipo: 'opcao',
  chave: 'tipo_tela',
  rotulo: 'Tipo de tela',
  obrigatorio: true,
  opcoes: ['2D', '3D', 'IMAX', 'VIP'],
}
const NUMERO_SALA: CampoDef = { tipo: 'inteiro', chave: 'numero', rotulo: 'Número da sala', obrigatorio: true, minimo: 1 }

const SALA_CRIAR: CampoDef[] = [
  { tipo: 'uuid', chave: 'cinema_id', rotulo: 'Id do cinema', obrigatorio: true },
  NUMERO_SALA,
  TIPO_TELA,
  { tipo: 'fileiras', chave: 'fileiras', rotulo: 'Fileiras', obrigatorio: true },
  ATIVO,
]
const SALA_ATUALIZAR: CampoDef[] = [NUMERO_SALA, TIPO_TELA, ATIVO]

const SESSAO_COMUM: CampoDef[] = [
  { tipo: 'datahora', chave: 'data_hora_inicio', rotulo: 'Início (AAAA-MM-DD HH:MM)', obrigatorio: true },
  { tipo: 'opcao', chave: 'idioma', rotulo: 'Idioma', obrigatorio: true, opcoes: ['DUBLADO', 'LEGENDADO'] },
  { tipo: 'decimal', chave: 'preco_base', rotulo: 'Preço base (ex.: 42.50)', obrigatorio: true },
  {
    tipo: 'opcao',
    chave: 'status',
    rotulo: 'Status',
    obrigatorio: false,
    valorPadrao: 'AGENDADA',
    opcoes: ['AGENDADA', 'EM_ANDAMENTO', 'FINALIZADA', 'CANCELADA'],
  },
]
const FILME_ID: CampoDef = { tipo: 'uuid', chave: 'filme_id', rotulo: 'Id do filme', obrigatorio: true }
const SESSAO_CRIAR: CampoDef[] = [
  FILME_ID,
  { tipo: 'uuid', chave: 'sala_id', rotulo: 'Id da sala', obrigatorio: true },
  ...SESSAO_COMUM,
]
const SESSAO_ATUALIZAR: CampoDef[] = [FILME_ID, ...SESSAO_COMUM]

export const FORMULARIOS: Record<string, CampoDef[]> = {
  'filmes.criar': FILME,
  'filmes.atualizar': FILME,
  'cinemas.criar': CINEMA,
  'cinemas.atualizar': CINEMA,
  'salas.criar': SALA_CRIAR,
  'salas.atualizar': SALA_ATUALIZAR,
  'sessoes.criar': SESSAO_CRIAR,
  'sessoes.atualizar': SESSAO_ATUALIZAR,
}

export function obterFormulario(def: Definicao): CampoDef[] | undefined {
  return FORMULARIOS[`${def.controller}.${def.acao}`]
}
