import { NavLink, Outlet } from 'react-router-dom'
import { useAuth } from './auth/ContextoAuth'
import { Botao } from './componentes/ui'

const ligacoes = [
  { para: '/', texto: 'Sessões', exata: true },
  { para: '/filmes', texto: 'Filmes' },
  { para: '/cinemas', texto: 'Cinemas' },
  { para: '/meus-ingressos', texto: 'Meus ingressos' },
  { para: '/admin', texto: 'Operação' },
  { para: '/portaria', texto: 'Portaria' },
]

export function App() {
  const { sessao, sair } = useAuth()

  return (
    <div className="min-h-screen">
      <header className="border-b border-borda bg-painel/60 backdrop-blur">
        <div className="mx-auto flex max-w-6xl flex-wrap items-center gap-x-6 gap-y-3 px-4 py-4">
          <NavLink to="/" className="text-lg font-bold tracking-tight">
            <span className="text-destaque">●</span> Cinema
          </NavLink>

          <nav className="flex flex-wrap gap-1 text-sm">
            {ligacoes.map((l) => (
              <NavLink
                key={l.para}
                to={l.para}
                end={l.exata}
                className={({ isActive }) =>
                  `rounded-lg px-3 py-1.5 transition ${
                    isActive ? 'bg-destaque/15 text-destaque' : 'text-slate-400 hover:text-slate-100'
                  }`
                }
              >
                {l.texto}
              </NavLink>
            ))}
          </nav>

          <div className="ml-auto flex items-center gap-3 text-sm">
            {sessao ? (
              <>
                <span className="text-slate-400">{sessao.usuario}</span>
                <Botao variante="secundario" onClick={sair}>
                  Sair
                </Botao>
              </>
            ) : (
              <NavLink to="/entrar" className="rounded-lg bg-destaque px-4 py-2 font-semibold text-slate-950">
                Entrar
              </NavLink>
            )}
          </div>
        </div>
      </header>

      <main className="mx-auto max-w-6xl px-4 py-8">
        <Outlet />
      </main>
    </div>
  )
}
