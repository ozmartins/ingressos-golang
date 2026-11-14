import { useState } from 'react'
import type { Cinema, Filme } from '../api/tipos'
import { filtrosCinemasVazios } from '../dominio/filtrosCinemas'
import { filtrosFilmesVazios } from '../dominio/filtrosFilmes'
import { useAbaNaUrl, type Aba } from '../hooks/useAbaNaUrl'
import { AbaCinemas } from './AbaCinemas'
import { AbaFilmes } from './AbaFilmes'
import { Abas, type DefinicaoAba } from './Abas'
import { idAba, idPainel } from './idsAbas'

const ABAS: DefinicaoAba<Aba>[] = [
  { id: 'filmes', rotulo: 'Filmes' },
  { id: 'cinemas', rotulo: 'Cinemas' },
]

interface Props {
  /** Injetáveis para testes; por padrão consultam o catálogo. */
  carregarFilmes?: () => Promise<Filme[]>
  carregarCinemas?: () => Promise<Cinema[]>
}

/**
 * As duas abas ficam montadas e a inativa apenas oculta: assim os filtros e
 * os dados carregados de cada uma sobrevivem à troca de aba.
 */
export function PaginaInicial({ carregarFilmes, carregarCinemas }: Props) {
  const [aba, setAba] = useAbaNaUrl()
  const [filtrosFilmes, setFiltrosFilmes] = useState(filtrosFilmesVazios)
  const [filtrosCinemas, setFiltrosCinemas] = useState(filtrosCinemasVazios)

  return (
    <div className="pagina">
      <header className="cabecalho">
        <h1>Ingressos</h1>
        <p>Encontre o filme e o cinema ideais.</p>
      </header>

      <Abas abas={ABAS} ativa={aba} onChange={setAba} rotulo="Filmes e cinemas" />

      <section role="tabpanel" id={idPainel('filmes')} aria-labelledby={idAba('filmes')} hidden={aba !== 'filmes'}>
        <AbaFilmes filtros={filtrosFilmes} onFiltros={setFiltrosFilmes} carregar={carregarFilmes} />
      </section>
      <section role="tabpanel" id={idPainel('cinemas')} aria-labelledby={idAba('cinemas')} hidden={aba !== 'cinemas'}>
        <AbaCinemas filtros={filtrosCinemas} onFiltros={setFiltrosCinemas} carregar={carregarCinemas} />
      </section>
    </div>
  )
}
