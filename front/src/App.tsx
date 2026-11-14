import { useState } from 'react'
import type { Cinema, Filme } from './api/tipos'
import { BarraConta } from './componentes/BarraConta'
import { PaginaCinema, type CarregadoresCinema } from './componentes/PaginaCinema'
import { PaginaCompra, type CarregadoresCompra } from './componentes/PaginaCompra'
import { PaginaInicial } from './componentes/PaginaInicial'
import { useRota } from './hooks/useRota'
import { useSessao } from './hooks/useSessao'

interface Props {
  /** Injetáveis para testes; por padrão consultam o catálogo. */
  carregarFilmes?: () => Promise<Filme[]>
  carregarCinemas?: () => Promise<Cinema[]>
  carregadoresCinema?: Partial<CarregadoresCinema>
  carregadoresCompra?: Partial<CarregadoresCompra>
}

/**
 * A página inicial, uma vez aberta, permanece montada (oculta) enquanto o
 * detalhe de um cinema é exibido: assim aba, filtros e dados sobrevivem ao "voltar".
 */
function App({ carregarFilmes, carregarCinemas, carregadoresCinema, carregadoresCompra }: Props) {
  const [rota] = useRota()
  const { sessao, sair } = useSessao()
  const [inicioAberto, setInicioAberto] = useState(rota.tipo === 'inicio')
  if (rota.tipo === 'inicio' && !inicioAberto) setInicioAberto(true)

  return (
    <>
      <BarraConta sessao={sessao} onSair={sair} />
      {inicioAberto ? (
        <div hidden={rota.tipo !== 'inicio'}>
          <PaginaInicial carregarFilmes={carregarFilmes} carregarCinemas={carregarCinemas} />
        </div>
      ) : null}
      {rota.tipo === 'cinema' ? (
        <PaginaCinema key={rota.id} id={rota.id} carregadores={carregadoresCinema} />
      ) : null}
      {rota.tipo === 'compra' ? (
        <PaginaCompra
          key={`${rota.cinemaId}/${rota.filmeId}`}
          cinemaId={rota.cinemaId}
          filmeId={rota.filmeId}
          sessao={sessao}
          carregadores={carregadoresCompra}
        />
      ) : null}
    </>
  )
}

export default App
