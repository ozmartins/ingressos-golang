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

function linhaId(o: Registro): string[] {
  const id = texto(o, 'id')
  return id ? [`  Id: ${id}`] : []
}

function formatarFilme(o: Registro): string[] {
  const duracao = numero(o, 'duracao_minutos')
  return [
    texto(o, 'titulo') ?? '(sem título)',
    `  ${juntar([
      duracao === undefined ? undefined : `${duracao} min`,
      texto(o, 'classificacao_etaria'),
      texto(o, 'genero'),
      rotulo(o.status),
    ])}`,
    ...(texto(o, 'sinopse') ? [`  ${texto(o, 'sinopse')}`] : []),
    ...(texto(o, 'imagem_url') ? [`  Imagem: ${texto(o, 'imagem_url')}`] : []),
    ...linhaId(o),
  ]
}

function formatarCinema(o: Registro): string[] {
  const local = juntar([texto(o, 'cidade'), texto(o, 'estado')], '/')
  return [
    texto(o, 'nome') ?? '(sem nome)',
    `  ${juntar([juntar([texto(o, 'endereco'), local], ', '), o.ativo === false ? 'Inativo' : 'Ativo'])}`,
    ...linhaId(o),
  ]
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

function formatarSala(o: Registro): string[] {
  const lugares = numero(o, 'capacidade_total')
  const fileiras = formatarFileiras(o.fileiras)
  return [
    juntar([
      `Sala ${numero(o, 'numero') ?? '?'}`,
      texto(o, 'tipo_tela'),
      lugares === undefined ? undefined : `${lugares} lugares`,
      o.ativo === false ? 'Inativa' : 'Ativa',
    ]),
    ...(fileiras ? [`  Fileiras: ${fileiras}`] : []),
    ...(texto(o, 'cinema_id') ? [`  Cinema: ${texto(o, 'cinema_id')}`] : []),
    ...linhaId(o),
  ]
}

function formatarSessao(o: Registro, opcoes: OpcoesFormato): string[] {
  const quando = dataHora(o.data_hora_inicio, opcoes.fuso)
  if (texto(o, 'filme_titulo')) {
    const sala = `Sala ${numero(o, 'sala_numero') ?? '?'}${texto(o, 'tipo_tela') ? ` (${texto(o, 'tipo_tela')})` : ''}`
    return [
      juntar([texto(o, 'filme_titulo'), quando], ' — '),
      `  ${juntar([texto(o, 'cinema_nome'), sala, rotulo(o.idioma), preco(o.preco_base)])}`,
      ...linhaId(o),
      `  ${juntar([
        texto(o, 'filme_id') ? `Filme: ${texto(o, 'filme_id')}` : undefined,
        texto(o, 'cinema_id') ? `Cinema: ${texto(o, 'cinema_id')}` : undefined,
      ])}`,
    ]
  }
  return [
    `Sessão em ${quando}`,
    `  ${juntar([rotulo(o.idioma), preco(o.preco_base), rotulo(o.status)])}`,
    ...(texto(o, 'filme_id') ? [`  Filme: ${texto(o, 'filme_id')}`] : []),
    ...(texto(o, 'sala_id') ? [`  Sala: ${texto(o, 'sala_id')}`] : []),
    ...linhaId(o),
  ]
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

function formatarHealth(o: Registro): string[] {
  const dependencias = ehObjeto(o.dependencias) ? o.dependencias : {}
  return [
    `Serviço: ${texto(o, 'status') ?? '?'}`,
    ...Object.entries(dependencias).map(([nome, estado]) => `  ${nome}: ${String(estado)}`),
  ]
}

function formatarItem(controller: Controller, corpo: unknown, opcoes: OpcoesFormato): string[] {
  if (!ehObjeto(corpo)) return formatarGenerico(corpo)
  switch (controller) {
    case 'filmes':
      return formatarFilme(corpo)
    case 'cinemas':
      return formatarCinema(corpo)
    case 'salas':
      return formatarSala(corpo)
    case 'sessoes':
      return formatarSessao(corpo, opcoes)
    case 'operacional':
      return formatarHealth(corpo)
  }
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

function formatarLista(controller: Controller, corpo: Registro, opcoes: OpcoesFormato): string[] {
  const itens = Array.isArray(corpo.itens) ? corpo.itens : []
  const linhas: string[] = []
  itens.forEach((item, i) => {
    if (i > 0) linhas.push('')
    linhas.push(...formatarItem(controller, item, opcoes))
  })
  if (itens.length === 0) linhas.push('Nenhum registro encontrado.')
  const pagina = formatarPagina(corpo.pagina, itens.length)
  return pagina.length > 0 ? [...linhas, '', ...pagina] : linhas
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

  const formatado =
    def.acao === 'listar' && ehObjeto(corpo)
      ? formatarLista(def.controller, corpo, opcoes)
      : formatarItem(def.controller, corpo, opcoes)
  linhas.push(...formatado.map(saida))

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
  linhas.push(saida('Sessão'))
  linhas.push(saida('  login <client_id> <client_secret>'))
  linhas.push(sistema('      autentica para os comandos que exigem login'))
  linhas.push(saida('  logout'))
  linhas.push(sistema('      encerra a sessão'))
  return linhas
}
