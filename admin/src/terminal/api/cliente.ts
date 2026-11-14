import type { Metodo } from './tabela'

export interface Base {
  catalogo: string
  identidade: string
}

export interface Requisicao {
  metodo: Metodo
  caminho: string
  raiz: boolean
  query: Record<string, string>
  corpo?: unknown
  token: string | null
}

export interface RespostaApi {
  status: number
  corpo: unknown
  location?: string
}

export type TipoFalha = 'rede' | 'timeout'

export class FalhaRequisicao extends Error {
  tipo: TipoFalha
  constructor(tipo: TipoFalha) {
    super(tipo)
    this.tipo = tipo
  }
}

export const TIMEOUT_MS = 10_000

export interface RespostaBruta {
  status: number
  texto: string
  location?: string
}

export async function chamar(
  fetchFn: typeof fetch,
  url: string,
  init: RequestInit,
  timeoutMs: number = TIMEOUT_MS,
): Promise<RespostaBruta> {
  const controlador = new AbortController()
  const temporizador = setTimeout(() => controlador.abort(), timeoutMs)
  try {
    const resposta = await fetchFn(url, { ...init, signal: controlador.signal })
    const texto = await resposta.text()
    return {
      status: resposta.status,
      texto,
      location: resposta.headers.get('Location') ?? undefined,
    }
  } catch {
    throw new FalhaRequisicao(controlador.signal.aborted ? 'timeout' : 'rede')
  } finally {
    clearTimeout(temporizador)
  }
}

function interpretar(texto: string): unknown {
  if (texto.trim() === '') return null
  try {
    return JSON.parse(texto) as unknown
  } catch {
    return texto
  }
}

export function montarUrl(base: Base, req: Pick<Requisicao, 'caminho' | 'raiz' | 'query'>): string {
  const prefixo = req.raiz ? base.catalogo : `${base.catalogo}/api/v1`
  const consulta = new URLSearchParams(req.query).toString()
  return `${prefixo}${req.caminho}${consulta === '' ? '' : `?${consulta}`}`
}

export async function requisitar(
  fetchFn: typeof fetch,
  base: Base,
  req: Requisicao,
  timeoutMs: number = TIMEOUT_MS,
): Promise<RespostaApi> {
  const cabecalhos: Record<string, string> = { Accept: 'application/json' }
  if (req.token !== null) cabecalhos.Authorization = `Bearer ${req.token}`
  const init: RequestInit = { method: req.metodo, headers: cabecalhos }
  if (req.corpo !== undefined) {
    cabecalhos['Content-Type'] = 'application/json'
    init.body = JSON.stringify(req.corpo)
  }
  const bruta = await chamar(fetchFn, montarUrl(base, req), init, timeoutMs)
  return { status: bruta.status, corpo: interpretar(bruta.texto), location: bruta.location }
}
