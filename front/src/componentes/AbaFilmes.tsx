import { useMemo } from 'react'
import { buscarFilmes } from '../api/catalogo'
import type { Filme } from '../api/tipos'
import { filtrarFilmes, filtrosFilmesVazios, opcoesFilmes, type FiltrosFilmes } from '../dominio/filtrosFilmes'
import { ROTULO_STATUS } from '../dominio/formatar'
import { useLista } from '../hooks/useLista'
import { CampoBusca } from './CampoBusca'
import { CampoSelect } from './CampoSelect'
import { CartaoFilme } from './CartaoFilme'
import { EstadoLista } from './EstadoLista'

interface Props {
  filtros: FiltrosFilmes
  onFiltros: (filtros: FiltrosFilmes) => void
  /** Injetável para testes; por padrão consulta o catálogo. */
  carregar?: () => Promise<Filme[]>
}

const OPCOES_STATUS = [
  { valor: 'EM_CARTAZ', rotulo: ROTULO_STATUS.EM_CARTAZ },
  { valor: 'BREVE', rotulo: ROTULO_STATUS.BREVE },
]

export function AbaFilmes({ filtros, onFiltros, carregar = buscarFilmes }: Props) {
  const { estado, itens, erro, recarregar } = useLista(carregar)
  const opcoes = useMemo(() => opcoesFilmes(itens), [itens])
  const visiveis = useMemo(() => filtrarFilmes(itens, filtros), [itens, filtros])
  const limpar = () => onFiltros(filtrosFilmesVazios)
  const alterar = (parcial: Partial<FiltrosFilmes>) => onFiltros({ ...filtros, ...parcial })
  const comFiltros = JSON.stringify(filtros) !== JSON.stringify(filtrosFilmesVazios)

  return (
    <div>
      <form className="filtros" role="search" aria-label="Filtrar filmes" onSubmit={(e) => e.preventDefault()}>
        <CampoBusca rotulo="Buscar por título" valor={filtros.texto} onChange={(texto) => alterar({ texto })} />
        <CampoSelect
          rotulo="Gênero"
          valor={filtros.genero}
          todas="Todos os gêneros"
          opcoes={opcoes.generos.map((g) => ({ valor: g, rotulo: g }))}
          onChange={(genero) => alterar({ genero })}
        />
        <CampoSelect
          rotulo="Classificação"
          valor={filtros.classificacao}
          todas="Todas as classificações"
          opcoes={opcoes.classificacoes.map((c) => ({ valor: c, rotulo: c }))}
          onChange={(classificacao) => alterar({ classificacao })}
        />
        <CampoSelect
          rotulo="Situação"
          valor={filtros.status}
          todas="Todas as situações"
          opcoes={OPCOES_STATUS}
          onChange={(status) => alterar({ status: status as FiltrosFilmes['status'] })}
        />
        <button type="button" className="botao-secundario" onClick={limpar} disabled={!comFiltros}>
          Limpar filtros
        </button>
      </form>

      {estado === 'pronto' ? (
        <p className="contagem" aria-live="polite">
          {visiveis.length} {visiveis.length === 1 ? 'filme' : 'filmes'}
        </p>
      ) : null}

      <EstadoLista
        estado={estado}
        erro={erro}
        total={itens.length}
        visiveis={visiveis.length}
        rotulo="filmes"
        onTentarNovamente={recarregar}
        onLimparFiltros={limpar}
      >
        <ul className="grade">
          {visiveis.map((filme) => (
            <CartaoFilme key={filme.id} filme={filme} />
          ))}
        </ul>
      </EstadoLista>
    </div>
  )
}
