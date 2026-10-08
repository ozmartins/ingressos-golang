import { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState } from 'react'
import type { ReactNode } from 'react'
import { entrar as pedirEntrada, renovar, sair as pedirSaida } from './keycloak'
import type { Sessao } from './keycloak'

const CHAVE_REFRESH = 'ingressos.refresh_token'

interface Autenticacao {
  sessao: Sessao | null
  /** Já tentou restaurar a sessão do `sessionStorage`? Antes disso as rotas
   *  protegidas não podem concluir que ninguém está autenticado. */
  restaurando: boolean
  entrar: (usuario: string, senha: string) => Promise<void>
  sair: () => void
  /** Devolve um access token válido, renovando se estiver perto de vencer. */
  token: () => Promise<string>
}

const Contexto = createContext<Autenticacao | null>(null)

export function ProvedorAuth({ children }: { children: ReactNode }) {
  const [sessao, setSessao] = useState<Sessao | null>(null)
  const [restaurando, setRestaurando] = useState(true)
  // A renovação é lida fora do ciclo de render, por `token()`, que pode ser
  // chamada de dentro de uma query já em voo.
  const atual = useRef<Sessao | null>(null)

  const guardar = useCallback((nova: Sessao | null) => {
    atual.current = nova
    setSessao(nova)
    if (nova) sessionStorage.setItem(CHAVE_REFRESH, nova.refreshToken)
    else sessionStorage.removeItem(CHAVE_REFRESH)
  }, [])

  // O access token nunca sai da memória; só o refresh token sobrevive a um
  // recarregamento, e apenas enquanto a aba existir.
  useEffect(() => {
    const salvo = sessionStorage.getItem(CHAVE_REFRESH)
    if (!salvo) {
      setRestaurando(false)
      return
    }
    renovar(salvo)
      .then(guardar)
      .catch(() => guardar(null))
      .finally(() => setRestaurando(false))
  }, [guardar])

  const token = useCallback(async () => {
    const viva = atual.current
    if (!viva) throw new Error('sessão ausente')
    if (Date.now() < viva.expiraEm) return viva.accessToken
    const renovada = await renovar(viva.refreshToken)
    guardar(renovada)
    return renovada.accessToken
  }, [guardar])

  const valor = useMemo<Autenticacao>(
    () => ({
      sessao,
      restaurando,
      entrar: async (usuario, senha) => guardar(await pedirEntrada(usuario, senha)),
      sair: () => {
        const viva = atual.current
        guardar(null)
        if (viva) void pedirSaida(viva.refreshToken)
      },
      token,
    }),
    [sessao, restaurando, guardar, token],
  )

  return <Contexto.Provider value={valor}>{children}</Contexto.Provider>
}

export function useAuth(): Autenticacao {
  const valor = useContext(Contexto)
  if (!valor) throw new Error('useAuth fora do ProvedorAuth')
  return valor
}
