import { Navigate, Outlet, useLocation } from 'react-router-dom'
import { useAuth } from './ContextoAuth'

export function RotaProtegida() {
  const { sessao, restaurando } = useAuth()
  const local = useLocation()

  // Enquanto o refresh token guardado ainda está sendo trocado, mandar para o
  // login seria derrubar quem só recarregou a página.
  if (restaurando) {
    return <p className="p-8 text-slate-400">Restaurando a sessão…</p>
  }
  if (!sessao) {
    return <Navigate to="/entrar" replace state={{ de: local.pathname + local.search }} />
  }
  return <Outlet />
}
