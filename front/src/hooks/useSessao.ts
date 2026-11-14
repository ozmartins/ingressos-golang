import { useEffect, useRef, useState } from 'react'
import {
  concluirAutenticacao,
  lerSessao,
  limparSessao,
  salvarSessao,
  temRetorno,
  urlLogout,
  type Sessao,
} from '../dominio/keycloak'
import { navegar } from './useRota'

/** Sessão do usuário; trata a volta do Keycloak (?code=...) uma única vez. */
export function useSessao() {
  const [sessao, setSessao] = useState<Sessao | null>(lerSessao)
  const tratado = useRef(false)

  useEffect(() => {
    if (tratado.current || !temRetorno(window.location.search)) return
    tratado.current = true
    const consulta = window.location.search
    window.history.replaceState(null, '', window.location.pathname)
    void concluirAutenticacao(consulta)
      .then((r) => {
        if (r === null) return
        salvarSessao(r.sessao)
        setSessao(r.sessao)
        navegar(r.retorno)
      })
      .catch(() => undefined)
  }, [])

  function sair() {
    if (sessao === null) return
    limparSessao()
    setSessao(null)
    window.location.assign(urlLogout(sessao))
  }

  return { sessao, sair }
}
