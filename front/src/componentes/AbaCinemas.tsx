import { useMemo } from 'react'
import { buscarCinemas } from '../api/catalogo'
import type { Cinema } from '../api/tipos'
import {
  ajustarCidade,
  filtrarCinemas,
  filtrosCinemasVazios,
  opcoesCinemas,
  type FiltrosCinemas,
} from '../dominio/filtrosCinemas'
import { useLista } from '../hooks/useLista'
import { CampoBusca } from './CampoBusca'
import { CampoSelect } from './CampoSelect'
import { CartaoCinema } from './CartaoCinema'
import { EstadoLista } from './EstadoLista'

interface Props {
  filtros: FiltrosCinemas
  onFiltros: (filtros: FiltrosCinemas) => void
  /** Injetável para testes; por padrão consulta o catálogo. */
  carregar?: () => Promise<Cinema[]>
}

export function AbaCinemas({ filtros, onFiltros, carregar = buscarCinemas }: Props) {
  const { estado, itens, erro, recarregar } = useLista(carregar)
  const opcoes = useMemo(() => opcoesCinemas(itens, filtros.estado), [itens, filtros.estado])
  const visiveis = useMemo(() => filtrarCinemas(itens, filtros), [itens, filtros])
  const limpar = () => onFiltros(filtrosCinemasVazios)
  const comFiltros = JSON.stringify(filtros) !== JSON.stringify(filtrosCinemasVazios)

  return (
    <div>
      <form className="filtros" role="search" aria-label="Filtrar cinemas" onSubmit={(e) => e.preventDefault()}>
        <CampoBusca
          rotulo="Buscar por nome"
          valor={filtros.texto}
          onChange={(texto) => onFiltros({ ...filtros, texto })}
        />
        <CampoSelect
          rotulo="Estado"
          valor={filtros.estado}
          todas="Todos os estados"
          opcoes={opcoes.estados.map((e) => ({ valor: e, rotulo: e }))}
          onChange={(novo) => onFiltros(ajustarCidade(itens, { ...filtros, estado: novo }))}
        />
        <CampoSelect
          rotulo="Cidade"
          valor={filtros.cidade}
          todas="Todas as cidades"
          opcoes={opcoes.cidades.map((c) => ({ valor: c, rotulo: c }))}
          onChange={(cidade) => onFiltros({ ...filtros, cidade })}
        />
        <button type="button" className="botao-secundario" onClick={limpar} disabled={!comFiltros}>
          Limpar filtros
        </button>
      </form>

      {estado === 'pronto' ? (
        <p className="contagem" aria-live="polite">
          {visiveis.length} {visiveis.length === 1 ? 'cinema' : 'cinemas'}
        </p>
      ) : null}

      <EstadoLista
        estado={estado}
        erro={erro}
        total={itens.length}
        visiveis={visiveis.length}
        rotulo="cinemas"
        onTentarNovamente={recarregar}
        onLimparFiltros={limpar}
      >
        <ul className="grade">
          {visiveis.map((cinema) => (
            <CartaoCinema key={cinema.id} cinema={cinema} />
          ))}
        </ul>
      </EstadoLista>
    </div>
  )
}
