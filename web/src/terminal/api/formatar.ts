import type { Linha } from '../linhas'
import { FalhaRequisicao, TIMEOUT_MS, type RespostaApi } from './cliente'
import {
  CONTROLLERS_CRUD,
  NOMES_CONTROLLER,
  definicoesDe,
  type Controller,
  type Definicao,
} from './tabela'

const saida = (texto: string): Linha => ({ tipo: 'saida', texto })
const sistema = (texto: string): Linha => ({ tipo: 'sistema', texto })
const erro = (texto: string): Linha => ({ tipo: 'erro', texto })

function ehObjeto(x: unknown): x is Record<string, unknown> {
  return typeof x === 'object' && x !== null && !Array.isArray(x)
}

type Registro = Record<string, unknown>

export interface OpcoesFormato {
  fuso?: string
}

const ROTULOS: Record<string, string> = {
  EM_CARTAZ: 'Em cartaz',
  BREVE: 'Em breve',
  FORA_DE_CARTAZ: 'Fora de cartaz',
  AGENDADA: 'Agendada',
  EM_ANDAMENTO: 'Em andamento',
  FINALIZADA: 'Finalizada',
  CANCELADA: 'Cancelada',
  DUBLADO: 'Dublado',
  LEGENDADO: 'Legendado',
  PCD: 'PCD',
  NAMORADEIRA: 'namoradeira',
}

function texto(o: Registro, chave: string): string | undefined {
  const v = o[chave]
  return typeof v === 'string' && v !== '' ? v : undefined
}

function numero(o: Registro, chave: string): number | undefined {
  const v = o[chave]
  return typeof v === 'number' ? v : undefined
}

function rotulo(v: unknown): string {
  return typeof v === 'string' ? (ROTULOS[v] ?? v) : ''
}

function dataHora(v: unknown, fuso?: string): string {
  if (typeof v !== 'string') return ''
  const d = new Date(v)
  if (Number.isNaN(d.getTime())) return v
  return d.toLocaleString('pt-BR', { dateStyle: 'short', timeStyle: 'short', timeZone: fuso })
}

function preco(v: unknown): string {
  if (typeof v !== 'string' || v === '') return ''
  return /^\d+(\.\d+)?$/.test(v) ? `R$ ${v.replace('.', ',')}` : v
}

function juntar(partes: Array<string | undefined>, separador = ' · '): string {
  return partes.filter((p): p is string => p !== undefined && p !== '').join(separador)
}

function formatarFileiras(valor: unknown): string {
  if (!Array.isArray(valor)) return ''
  return valor
    .filter(ehObjeto)
    .map((f) => {
      const tipo = texto(f, 'tipo')
      const extra = tipo && tipo !== 'NORMAL' ? ` ${rotulo(tipo)}` : ''
      return `${texto(f, 'fileira') ?? '?'} (${numero(f, 'assentos') ?? '?'}${extra})`
    })
    .join(', ')
}

function formatarGenerico(valor: unknown, recuo = ''): string[] {
  if (valor === null || valor === undefined) return []
  if (Array.isArray(valor)) {
    return valor.flatMap((item) =>
      ehObjeto(item) || Array.isArray(item)
        ? [`${recuo}-`, ...formatarGenerico(item, `${recuo}  `)]
        : [`${recuo}- ${String(item)}`],
    )
  }
  if (ehObjeto(valor)) {
    return Object.entries(valor).flatMap(([chave, v]) =>
      v !== null && typeof v === 'object'
        ? [`${recuo}${chave}:`, ...formatarGenerico(v, `${recuo}  `)]
        : [`${recuo}${chave}: ${String(v)}`],
    )
  }
  return String(valor).split('\n')
}

function formatarPagina(pagina: unknown, quantidade: number): string[] {
  if (!ehObjeto(pagina)) return []
  const atual = numero(pagina, 'pagina') ?? 1
  const tamanho = numero(pagina, 'tamanho')
  const total = numero(pagina, 'total') ?? quantidade
  const paginas = tamanho && tamanho > 0 ? Math.max(1, Math.ceil(total / tamanho)) : 1
  const linhas = [`Página ${atual} de ${paginas} · ${total} ${total === 1 ? 'registro' : 'registros'} no total`]
  if (pagina.tem_proxima === true) linhas.push(`Há mais resultados: use page=${atual + 1}.`)
  return linhas
}

// ---- Tabelas -------------------------------------------------------------

function comprimento(t: string): number {
  return Array.from(t).length
}

function cortar(t: string, max?: number): string {
  if (max === undefined || comprimento(t) <= max) return t
  return `${Array.from(t).slice(0, max - 1).join('')}…`
}

function quebrar(t: string, largura: number): string {
  const linhas: string[] = []
  let atual = ''
  for (const palavra of t.split(/\s+/).filter(Boolean)) {
    if (atual === '') atual = palavra
    else if (comprimento(atual) + 1 + comprimento(palavra) <= largura) atual += ` ${palavra}`
    else {
      linhas.push(atual)
      atual = palavra
    }
  }
  if (atual !== '') linhas.push(atual)
  return linhas.join('\n')
}

export function desenharTabela(cabecalho: string[], linhas: string[][]): string {
  const celulas = linhas.map((l) => l.map((c) => c.split('\n')))
  const larguras = cabecalho.map((titulo, i) =>
    Math.max(comprimento(titulo), ...celulas.map((l) => Math.max(...l[i].map(comprimento)))),
  )
  const separador = `+${larguras.map((w) => '-'.repeat(w + 2)).join('+')}+`
  const linha = (partes: string[]) =>
    `| ${partes.map((p, i) => p + ' '.repeat(larguras[i] - comprimento(p))).join(' | ')} |`
  const saidaTexto = [separador, linha(cabecalho), separador]
  for (const l of celulas) {
    const altura = Math.max(...l.map((c) => c.length))
    for (let k = 0; k < altura; k++) saidaTexto.push(linha(l.map((c) => c[k] ?? '')))
  }
  saidaTexto.push(separador)
  return saidaTexto.join('\n')
}

const tabela = (texto: string): Linha => ({ tipo: 'tabela', texto })

interface Campo {
  rotulo: string
  valor: (o: Registro, opcoes: OpcoesFormato) => string
  lista?: boolean
  max?: number
  quebrar?: boolean
}

const situacao = (o: Registro, positivo: string, negativo: string) =>
  o.ativo === false ? negativo : positivo

const CAMPOS_FILME: Campo[] = [
  { rotulo: 'Id', valor: (o) => texto(o, 'id') ?? '', lista: true },
  { rotulo: 'Título', valor: (o) => texto(o, 'titulo') ?? '', lista: true, max: 32 },
  {
    rotulo: 'Duração',
    valor: (o) => (numero(o, 'duracao_minutos') === undefined ? '' : `${numero(o, 'duracao_minutos')} min`),
    lista: true,
  },
  { rotulo: 'Classificação', valor: (o) => texto(o, 'classificacao_etaria') ?? '', lista: true },
  { rotulo: 'Gênero', valor: (o) => texto(o, 'genero') ?? '', lista: true, max: 18 },
  { rotulo: 'Status', valor: (o) => rotulo(o.status), lista: true },
  { rotulo: 'Sinopse', valor: (o) => texto(o, 'sinopse') ?? '', quebrar: true },
  { rotulo: 'Imagem', valor: (o) => texto(o, 'imagem_url') ?? '' },
]

const CAMPOS_CINEMA: Campo[] = [
  { rotulo: 'Id', valor: (o) => texto(o, 'id') ?? '', lista: true },
  { rotulo: 'Nome', valor: (o) => texto(o, 'nome') ?? '', lista: true, max: 30 },
  { rotulo: 'Endereço', valor: (o) => texto(o, 'endereco') ?? '', lista: true, max: 40 },
  {
    rotulo: 'Cidade/UF',
    valor: (o) => juntar([texto(o, 'cidade'), texto(o, 'estado')], '/'),
    lista: true,
  },
  { rotulo: 'Situação', valor: (o) => situacao(o, 'Ativo', 'Inativo'), lista: true },
]

const CAMPOS_SALA: Campo[] = [
  { rotulo: 'Id', valor: (o) => texto(o, 'id') ?? '', lista: true },
  { rotulo: 'Número', valor: (o) => String(numero(o, 'numero') ?? ''), lista: true },
  { rotulo: 'Tela', valor: (o) => texto(o, 'tipo_tela') ?? '', lista: true },
  {
    rotulo: 'Lugares',
    valor: (o) => String(numero(o, 'capacidade_total') ?? ''),
    lista: true,
  },
  { rotulo: 'Situação', valor: (o) => situacao(o, 'Ativa', 'Inativa'), lista: true },
  { rotulo: 'Fileiras', valor: (o) => formatarFileiras(o.fileiras), lista: true, max: 32 },
  { rotulo: 'Cinema', valor: (o) => texto(o, 'cinema_id') ?? '', lista: true },
]

const salaDaGrade = (o: Registro) =>
  `${numero(o, 'sala_numero') ?? '?'}${texto(o, 'tipo_tela') ? ` (${texto(o, 'tipo_tela')})` : ''}`

const CAMPOS_SESSAO_GRADE: Campo[] = [
  { rotulo: 'Id', valor: (o) => texto(o, 'id') ?? '', lista: true },
  { rotulo: 'Filme', valor: (o) => texto(o, 'filme_titulo') ?? '', lista: true, max: 28 },
  { rotulo: 'Cinema', valor: (o) => texto(o, 'cinema_nome') ?? '', lista: true, max: 24 },
  { rotulo: 'Sala', valor: salaDaGrade, lista: true },
  { rotulo: 'Início', valor: (o, op) => dataHora(o.data_hora_inicio, op.fuso), lista: true },
  { rotulo: 'Idioma', valor: (o) => rotulo(o.idioma), lista: true },
  { rotulo: 'Preço', valor: (o) => preco(o.preco_base), lista: true },
  { rotulo: 'Filme (id)', valor: (o) => texto(o, 'filme_id') ?? '' },
  { rotulo: 'Cinema (id)', valor: (o) => texto(o, 'cinema_id') ?? '' },
]

const CAMPOS_SESSAO: Campo[] = [
  { rotulo: 'Id', valor: (o) => texto(o, 'id') ?? '', lista: true },
  { rotulo: 'Início', valor: (o, op) => dataHora(o.data_hora_inicio, op.fuso), lista: true },
  { rotulo: 'Idioma', valor: (o) => rotulo(o.idioma), lista: true },
  { rotulo: 'Preço', valor: (o) => preco(o.preco_base), lista: true },
  { rotulo: 'Status', valor: (o) => rotulo(o.status), lista: true },
  { rotulo: 'Filme', valor: (o) => texto(o, 'filme_id') ?? '', lista: true },
  { rotulo: 'Sala', valor: (o) => texto(o, 'sala_id') ?? '', lista: true },
]

function camposDe(controller: Controller, amostra: unknown): Campo[] {
  switch (controller) {
    case 'filmes':
      return CAMPOS_FILME
    case 'cinemas':
      return CAMPOS_CINEMA
    case 'salas':
      return CAMPOS_SALA
    case 'sessoes':
      return ehObjeto(amostra) && texto(amostra, 'filme_titulo') ? CAMPOS_SESSAO_GRADE : CAMPOS_SESSAO
    case 'operacional':
      return []
  }
}

const LARGURA_TEXTO_LONGO = 60

function tabelaDeRegistro(campos: Campo[], o: Registro, opcoes: OpcoesFormato): string {
  const linhas = campos
    .map((c): [string, string] => {
      const v = c.valor(o, opcoes)
      return [c.rotulo, c.quebrar ? quebrar(v, LARGURA_TEXTO_LONGO) : v]
    })
    .filter(([, v]) => v !== '')
  return desenharTabela(['Campo', 'Valor'], linhas)
}

function tabelaDeLista(campos: Campo[], itens: Registro[], opcoes: OpcoesFormato): string {
  const colunas = campos.filter((c) => c.lista)
  return desenharTabela(
    colunas.map((c) => c.rotulo),
    itens.map((o) => colunas.map((c) => cortar(c.valor(o, opcoes), c.max))),
  )
}

function tabelaHealth(o: Registro): string {
  const dependencias = ehObjeto(o.dependencias) ? o.dependencias : {}
  return desenharTabela(
    ['Campo', 'Valor'],
    [
      ['Serviço', texto(o, 'status') ?? '?'],
      ...Object.entries(dependencias).map(([nome, estado]): [string, string] => [nome, String(estado)]),
    ],
  )
}

function formatarItem(controller: Controller, corpo: unknown, opcoes: OpcoesFormato): Linha[] {
  if (!ehObjeto(corpo)) return formatarGenerico(corpo).map(saida)
  if (controller === 'operacional') return [tabela(tabelaHealth(corpo))]
  return [tabela(tabelaDeRegistro(camposDe(controller, corpo), corpo, opcoes))]
}

function formatarLista(controller: Controller, corpo: Registro, opcoes: OpcoesFormato): Linha[] {
  const itens = Array.isArray(corpo.itens) ? corpo.itens.filter(ehObjeto) : []
  const linhas: Linha[] =
    itens.length === 0
      ? [saida('Nenhum registro encontrado.')]
      : [tabela(tabelaDeLista(camposDe(controller, itens[0]), itens, opcoes))]
  const pagina = formatarPagina(corpo.pagina, itens.length)
  return [...linhas, ...pagina.map(saida)]
}

export function formatarResposta(
  def: Definicao,
  resposta: RespostaApi,
  opcoes: OpcoesFormato = {},
): Linha[] {
  if (resposta.status === 204) return [saida('Removido com sucesso.')]
  const { corpo } = resposta
  const linhas: Linha[] = []
  if (def.acao === 'criar') linhas.push(sistema('Cadastrado com sucesso.'))
  if (def.acao === 'atualizar') linhas.push(sistema('Atualizado com sucesso.'))

  linhas.push(
    ...(def.acao === 'listar' && ehObjeto(corpo)
      ? formatarLista(def.controller, corpo, opcoes)
      : formatarItem(def.controller, corpo, opcoes)),
  )

  if (resposta.status === 201 && resposta.location) {
    linhas.push(sistema(`Localização: ${resposta.location}`))
  }
  return linhas
}

const TITULOS_PADRAO: Record<number, string> = {
  400: 'Entrada inválida',
  401: 'Não autenticado',
  404: 'Não encontrado',
  409: 'Conflito',
  503: 'Serviço indisponível',
}

export function formatarErro(resposta: RespostaApi): Linha[] {
  const { status, corpo } = resposta
  const padrao = TITULOS_PADRAO[status] ?? 'Erro'
  if (!ehObjeto(corpo)) {
    const linhas = [erro(`${status} ${padrao}`)]
    if (typeof corpo === 'string' && corpo.trim() !== '') linhas.push(erro(`  ${corpo.trim()}`))
    return linhas
  }
  const titulo = typeof corpo.title === 'string' && corpo.title !== '' ? corpo.title : padrao
  const linhas = [erro(`${status} ${titulo}`)]
  if (typeof corpo.detail === 'string' && corpo.detail !== '') linhas.push(erro(`  ${corpo.detail}`))
  if (Array.isArray(corpo.errors)) {
    for (const item of corpo.errors) {
      if (!ehObjeto(item)) continue
      const campo = typeof item.campo === 'string' ? item.campo : '?'
      const mensagem = typeof item.mensagem === 'string' ? item.mensagem : ''
      linhas.push(erro(`  ${campo}: ${mensagem}`))
    }
  }
  if (!('title' in corpo) && !('detail' in corpo) && !('errors' in corpo)) {
    linhas.push(...formatarGenerico(corpo).map((l) => erro(`  ${l}`)))
  }
  return linhas
}

export function formatarFalha(falha: FalhaRequisicao): Linha[] {
  return falha.tipo === 'timeout'
    ? [erro(`tempo esgotado: o serviço não respondeu em ${TIMEOUT_MS / 1000} s`)]
    : [erro('serviço indisponível: não foi possível conectar ao catálogo')]
}

export function formatarUso(def: Definicao): Linha[] {
  return [erro(`uso: ${def.uso}`)]
}

export function formatarAcoes(controller: Controller): Linha[] {
  const acoes = definicoesDe(controller).map((d) => d.acao)
  return [erro(`ações de ${controller}: ${acoes.join(', ')}`)]
}

export function formatarAjuda(): Linha[] {
  const linhas: Linha[] = [saida(''), saida('Comandos da API do catálogo:')]
  for (const controller of [...CONTROLLERS_CRUD, 'operacional' as const]) {
    linhas.push(saida(`${NOMES_CONTROLLER[controller]}`))
    for (const def of definicoesDe(controller)) {
      linhas.push(saida(`  ${def.uso}`))
      linhas.push(sistema(`      ${def.descricao}`))
    }
  }
  linhas.push(saida(''))
  linhas.push(sistema('criar e atualizar perguntam os campos um a um; digite "cancelar" (ou Esc) para abortar.'))
  linhas.push(saida('Sessão'))
  linhas.push(saida('  login'))
  linhas.push(sistema('      abre a página de login do Keycloak (usuário e senha)'))
  linhas.push(saida('  login <client_id> <client_secret>'))
  linhas.push(sistema('      autentica por client credentials, sem sair do terminal'))
  linhas.push(saida('  logout'))
  linhas.push(sistema('      encerra a sessão'))
  return linhas
}
