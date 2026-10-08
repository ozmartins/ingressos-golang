import { useState } from 'react'
import type { FormEvent } from 'react'
import { Navigate, useLocation, useNavigate } from 'react-router-dom'
import { useAuth } from '../auth/ContextoAuth'
import { Botao, Campo, Cartao } from '../componentes/ui'

export function Entrar() {
  const { sessao, entrar } = useAuth()
  const navegar = useNavigate()
  const local = useLocation()
  const destino = (local.state as { de?: string } | null)?.de ?? '/'

  const [usuario, setUsuario] = useState('teste')
  const [senha, setSenha] = useState('')
  const [erro, setErro] = useState<string | null>(null)
  const [enviando, setEnviando] = useState(false)

  if (sessao) return <Navigate to={destino} replace />

  async function enviar(evento: FormEvent) {
    evento.preventDefault()
    setErro(null)
    setEnviando(true)
    try {
      await entrar(usuario, senha)
      navegar(destino, { replace: true })
    } catch (e) {
      setErro(e instanceof Error ? e.message : 'Falha ao autenticar.')
    } finally {
      setEnviando(false)
    }
  }

  return (
    <div className="mx-auto max-w-sm">
      <Cartao>
        <h1 className="text-xl font-bold">Entrar</h1>
        <p className="mt-1 text-sm text-slate-400">Autenticação pelo Keycloak do realm <code>cinema</code>.</p>

        <form onSubmit={enviar} className="mt-6 space-y-4">
          <Campo rotulo="Usuário" value={usuario} onChange={(e) => setUsuario(e.target.value)} autoComplete="username" required />
          <Campo
            rotulo="Senha"
            type="password"
            value={senha}
            onChange={(e) => setSenha(e.target.value)}
            autoComplete="current-password"
            required
          />
          {erro && <p className="text-sm text-rose-300">{erro}</p>}
          <Botao type="submit" disabled={enviando} className="w-full">
            {enviando ? 'Entrando…' : 'Entrar'}
          </Botao>
        </form>

        <p className="mt-6 text-xs text-slate-500">
          O realm de desenvolvimento traz o usuário <code>teste</code> com senha <code>teste</code>.
        </p>
      </Cartao>
    </div>
  )
}
