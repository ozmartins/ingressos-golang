import type { Base } from './cliente'

export const BASE_TESTE: Base = { catalogo: '/catalogo', identidade: '/identidade' }

export interface Chamada {
  url: string
  init: RequestInit
}

export function respostaJson(status: number, corpo: unknown, cabecalhos: Record<string, string> = {}) {
  return new Response(JSON.stringify(corpo), {
    status,
    headers: { 'content-type': 'application/json', ...cabecalhos },
  })
}

export function fakeFetch(resposta: () => Response) {
  const chamadas: Chamada[] = []
  const fn = (async (url: RequestInfo | URL, init?: RequestInit) => {
    chamadas.push({ url: String(url), init: init ?? {} })
    return resposta()
  }) as typeof fetch
  return { fn, chamadas }
}

export function fetchQueFalha() {
  return (async () => {
    throw new TypeError('Failed to fetch')
  }) as typeof fetch
}

export function fetchPendente() {
  return ((_url: RequestInfo | URL, init?: RequestInit) =>
    new Promise<Response>((_resolve, rejeitar) => {
      init?.signal?.addEventListener('abort', () => {
        const e = new Error('aborted')
        e.name = 'AbortError'
        rejeitar(e)
      })
    })) as typeof fetch
}
