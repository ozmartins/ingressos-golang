import { useEffect, useRef } from 'react'
import { urlAutenticacao, type Sessao } from '../dominio/keycloak'

/** Leva à tela de login do Keycloak, pedindo que volte a `retorno`. */
export function irParaLogin(retorno: string): void {
  void urlAutenticacao('login', retorno).then((url) => window.location.assign(url))
}

/**
 * Exige login: sem sessão válida, redireciona uma única vez ao Keycloak com
 * retorno à página atual (caminho + consulta). Devolve `true` enquanto não há sessão.
 */
export function useGuardaLogin(sessao: Sessao | null, ir: (retorno: string) => void = irParaLogin): boolean {
  const redirecionado = useRef(false)
  const exigido = sessao === null

  useEffect(() => {
    if (!exigido || redirecionado.current) return
    redirecionado.current = true
    ir(window.location.pathname + window.location.search)
  }, [exigido, ir])

  return exigido
}
