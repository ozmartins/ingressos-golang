import {
  CAMPO_FILEIRA_ASSENTOS,
  CAMPO_FILEIRA_LETRA,
  CAMPO_FILEIRA_TIPO,
  converter,
  montarCorpo,
  type CampoDef,
  type Fileira,
} from './api/formularios'
import type { Definicao } from './api/tabela'
import type { Linha } from './linhas'

export type Etapa =
  | 'campo'
  | 'fileira-letra'
  | 'fileira-assentos'
  | 'fileira-tipo'
  | 'fileira-mais'
  | 'confirmacao'

export interface Dialogo {
  definicao: Definicao
  id?: string
  campos: CampoDef[]
  indice: number
  respostas: Record<string, unknown>
  fileiras: Fileira[]
  fileiraAtual: Partial<Fileira>
  etapa: Etapa
}

export interface Passo {
  dialogo: Dialogo | null
  linhas: Linha[]
  enviar?: { corpo: Record<string, unknown> }
}

const sistema = (texto: string): Linha => ({ tipo: 'sistema', texto })
const erro = (texto: string): Linha => ({ tipo: 'erro', texto })

export const MSG_CANCELADO = 'Preenchimento cancelado. Nada foi enviado.'

/** As respostas de um diálogo ativo nunca entram no histórico de comandos. */
export function registrarNoHistorico(dialogo: Dialogo | null): boolean {
  return dialogo === null
}

export function cancelar(): Passo {
  return { dialogo: null, linhas: [sistema(MSG_CANCELADO)] }
}

function pergunta(c: CampoDef): Linha {
  const partes: string[] = [c.obrigatorio ? 'obrigatório' : 'opcional']
  if (c.valorPadrao !== undefined) partes.push(`padrão ${c.valorPadrao}`)
  let sufixo = ''
  if (c.tipo === 'opcao') sufixo = ` [${c.opcoes.join('|')}]`
  else if (c.tipo === 'logico') sufixo = ' [s/n]'
  return sistema(`${c.rotulo} (${partes.join(', ')})${sufixo}:`)
}

function resumoFileiras(fs: Fileira[]): string {
  return fs.map((f) => `${f.fileira}:${f.assentos} ${f.tipo}`).join(', ')
}

function resumo(d: Dialogo): Linha[] {
  const linhas: Linha[] = [sistema('Resumo:')]
  for (const c of d.campos) {
    if (c.tipo === 'fileiras') {
      linhas.push(sistema(`  ${c.chave}: ${resumoFileiras(d.fileiras)}`))
      continue
    }
    const v = d.respostas[c.chave]
    if (v !== undefined) linhas.push(sistema(`  ${c.chave}: ${String(v)}`))
  }
  linhas.push(sistema('Enviar? [s/n]:'))
  return linhas
}

function primeiraPergunta(d: Dialogo): Linha {
  const c = d.campos[d.indice]
  if (c.tipo === 'fileiras') return sistema('Fileiras da sala: informe uma de cada vez.')
  return pergunta(c)
}

/** Avança para o próximo campo (ou para a confirmação). */
function proximoCampo(d: Dialogo, linhas: Linha[]): Passo {
  const indice = d.indice + 1
  if (indice >= d.campos.length) {
    const prox: Dialogo = { ...d, indice, etapa: 'confirmacao' }
    return { dialogo: prox, linhas: [...linhas, ...resumo(prox)] }
  }
  const c = d.campos[indice]
  if (c.tipo === 'fileiras') {
    const prox: Dialogo = { ...d, indice, etapa: 'fileira-letra', fileiraAtual: {} }
    return {
      dialogo: prox,
      linhas: [...linhas, sistema('Fileiras da sala: informe uma de cada vez.'), pergunta(CAMPO_FILEIRA_LETRA)],
    }
  }
  const prox: Dialogo = { ...d, indice, etapa: 'campo' }
  return { dialogo: prox, linhas: [...linhas, pergunta(c)] }
}

export function iniciar(definicao: Definicao, campos: CampoDef[], id?: string): Passo {
  const base: Dialogo = {
    definicao,
    campos,
    indice: 0,
    respostas: {},
    fileiras: [],
    fileiraAtual: {},
    etapa: 'campo',
    ...(id === undefined ? {} : { id }),
  }
  const alvo = id === undefined ? definicao.uso : `${definicao.controller} ${definicao.acao} ${id}`
  const intro = sistema(
    `Preencha os campos de "${alvo}". Enter em branco omite campo opcional; digite "cancelar" (ou Esc) para abortar.`,
  )
  const inicial: Dialogo =
    campos[0].tipo === 'fileiras' ? { ...base, etapa: 'fileira-letra' } : base
  const perguntaInicial =
    campos[0].tipo === 'fileiras'
      ? [primeiraPergunta(base), pergunta(CAMPO_FILEIRA_LETRA)]
      : [primeiraPergunta(base)]
  return { dialogo: inicial, linhas: [intro, ...perguntaInicial] }
}

function responderFileira(d: Dialogo, texto: string): Passo {
  const repetir = (msg: string, c: CampoDef): Passo => ({ dialogo: d, linhas: [erro(msg), pergunta(c)] })

  if (d.etapa === 'fileira-letra') {
    const r = converter(CAMPO_FILEIRA_LETRA, texto)
    if ('erro' in r) return repetir(r.erro, CAMPO_FILEIRA_LETRA)
    return {
      dialogo: { ...d, etapa: 'fileira-assentos', fileiraAtual: { fileira: String(r.valor) } },
      linhas: [pergunta(CAMPO_FILEIRA_ASSENTOS)],
    }
  }
  if (d.etapa === 'fileira-assentos') {
    const r = converter(CAMPO_FILEIRA_ASSENTOS, texto)
    if ('erro' in r) return repetir(r.erro, CAMPO_FILEIRA_ASSENTOS)
    return {
      dialogo: { ...d, etapa: 'fileira-tipo', fileiraAtual: { ...d.fileiraAtual, assentos: Number(r.valor) } },
      linhas: [pergunta(CAMPO_FILEIRA_TIPO)],
    }
  }
  if (d.etapa === 'fileira-tipo') {
    const r = converter(CAMPO_FILEIRA_TIPO, texto)
    if ('erro' in r) return repetir(r.erro, CAMPO_FILEIRA_TIPO)
    const f: Fileira = {
      fileira: d.fileiraAtual.fileira ?? '',
      assentos: d.fileiraAtual.assentos ?? 0,
      tipo: r.valor === undefined ? 'NORMAL' : String(r.valor),
    }
    return {
      dialogo: { ...d, etapa: 'fileira-mais', fileiras: [...d.fileiras, f], fileiraAtual: {} },
      linhas: [sistema(`Fileira ${f.fileira} registrada.`), sistema('Adicionar outra fileira? [s/n]:')],
    }
  }
  // fileira-mais
  const v = texto.trim().toLowerCase()
  if (['s', 'sim'].includes(v)) {
    return {
      dialogo: { ...d, etapa: 'fileira-letra', fileiraAtual: {} },
      linhas: [pergunta(CAMPO_FILEIRA_LETRA)],
    }
  }
  if (['n', 'nao', 'não'].includes(v)) {
    return proximoCampo({ ...d, respostas: { ...d.respostas, fileiras: d.fileiras } }, [])
  }
  return { dialogo: d, linhas: [erro('responda s ou n'), sistema('Adicionar outra fileira? [s/n]:')] }
}

export function responder(d: Dialogo, texto: string): Passo {
  if (texto.trim().toLowerCase() === 'cancelar') return cancelar()

  if (d.etapa === 'confirmacao') {
    const v = texto.trim().toLowerCase()
    if (['s', 'sim'].includes(v)) {
      return { dialogo: null, linhas: [], enviar: { corpo: montarCorpo(d.campos, d.respostas) } }
    }
    if (['n', 'nao', 'não'].includes(v)) return cancelar()
    return { dialogo: d, linhas: [erro('responda s ou n'), sistema('Enviar? [s/n]:')] }
  }

  if (d.etapa !== 'campo') return responderFileira(d, texto)

  const campo = d.campos[d.indice]
  const r = converter(campo, texto)
  if ('erro' in r) return { dialogo: d, linhas: [erro(r.erro), pergunta(campo)] }
  return proximoCampo({ ...d, respostas: { ...d.respostas, [campo.chave]: r.valor } }, [])
}
