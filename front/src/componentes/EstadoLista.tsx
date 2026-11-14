import type { ReactNode } from 'react'
import type { EstadoCarga } from '../hooks/useLista'

interface Props {
  estado: EstadoCarga
  erro: string
  /** Quantidade de itens cadastrados (antes dos filtros). */
  total: number
  /** Quantidade de itens após os filtros. */
  visiveis: number
  /** Nome do que está sendo listado, no plural: "filmes", "cinemas". */
  rotulo: string
  onTentarNovamente: () => void
  onLimparFiltros: () => void
  children: ReactNode
}

/** Escolhe entre carregando, erro, vazio, sem resultado ou a lista em si. */
export function EstadoLista({
  estado,
  erro,
  total,
  visiveis,
  rotulo,
  onTentarNovamente,
  onLimparFiltros,
  children,
}: Props) {
  if (estado === 'carregando') {
    return (
      <p className="estado" role="status">
        Carregando {rotulo}…
      </p>
    )
  }
  if (estado === 'erro') {
    return (
      <div className="estado estado-erro" role="alert">
        <p>Não foi possível carregar os {rotulo}. {erro}</p>
        <button type="button" onClick={onTentarNovamente}>
          Tentar novamente
        </button>
      </div>
    )
  }
  if (total === 0) {
    return <p className="estado">Ainda não há {rotulo} disponíveis.</p>
  }
  if (visiveis === 0) {
    return (
      <div className="estado">
        <p>Nenhum resultado para os filtros escolhidos.</p>
        <button type="button" onClick={onLimparFiltros}>
          Limpar filtros
        </button>
      </div>
    )
  }
  return <>{children}</>
}
