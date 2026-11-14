import { useMemo, useState } from 'react'
import { buscarCinemas, buscarFilme, buscarSessoesDoFilme, FilmeNaoEncontrado } from '../api/catalogo'
import type { Cinema, Filme, SessaoNaGrade } from '../api/tipos'
import { cinemasComOFilme } from '../dominio/cinemasDoFilme'
import {
  ajustarCidade,
  filtrarCinemas,
  filtrosCinemasVazios,
  opcoesCinemas,
  type FiltrosCinemas,
} from '../dominio/filtrosCinemas'
import { formatarDuracao, ROTULO_STATUS } from '../dominio/formatar'
import { caminhoCompra } from '../dominio/rotas'
import { interceptarClique, navegar, veioDoApp } from '../hooks/useRota'
import { useRecurso } from '../hooks/useRecurso'
import { CampoSelect } from './CampoSelect'
import { CartaoCinemaDoFilme } from './CartaoCinemaDoFilme'
import { EstadoLista } from './EstadoLista'

const CAMINHO_LISTA = '/?aba=filmes'

export interface CarregadoresFilme {
  carregarFilme: (id: string) => Promise<Filme>
  carregarSessoes: (filmeId: string) => Promise<SessaoNaGrade[]>
  carregarCinemas: () => Promise<Cinema[]>
}

const PADRAO: CarregadoresFilme = {
  carregarFilme: buscarFilme,
  carregarSessoes: buscarSessoesDoFilme,
  carregarCinemas: buscarCinemas,
}

function voltarParaLista() {
  if (veioDoApp()) window.history.back()
  else navegar(CAMINHO_LISTA)
}

function LinkVoltar() {
  return (
    <a className="link-voltar" href={CAMINHO_LISTA} onClick={(e) => interceptarClique(e, voltarParaLista)}>
      ← Voltar aos filmes
    </a>
  )
}

function CabecalhoFilme({ filme }: { filme: Filme }) {
  return (
    <header className="cinema-cabecalho">
      {filme.imagem_url ? (
        <img className="filme-poster" src={filme.imagem_url} alt={`Pôster de ${filme.titulo}`} />
      ) : (
        <div className="filme-poster filme-poster-vazio" aria-hidden="true">
          🎬
        </div>
      )}
      <div>
        <h1>{filme.titulo}</h1>
        <p className="cartao-meta">
          {filme.genero} · {formatarDuracao(filme.duracao_minutos)}
        </p>
        <p className="cartao-selos">
          <span className="selo">Classificação: {filme.classificacao_etaria}</span>
          <span className={`selo selo-${filme.status.toLowerCase()}`}>{ROTULO_STATUS[filme.status]}</span>
        </p>
      </div>
    </header>
  )
}

/** Um filme e os cinemas que o têm em cartaz, com filtro por UF e cidade. */
export function PaginaFilme({ id, carregadores }: { id: string; carregadores?: Partial<CarregadoresFilme> }) {
  const c = { ...PADRAO, ...carregadores }
  const filme = useRecurso(() => c.carregarFilme(id), id)
  const cinemas = useRecurso(async () => {
    const [sessoes, ativos] = await Promise.all([c.carregarSessoes(id), c.carregarCinemas()])
    return cinemasComOFilme(sessoes, ativos).map((item) => item.cinema)
  }, id)
  const [filtros, setFiltros] = useState<FiltrosCinemas>(filtrosCinemasVazios)

  const lista = useMemo(() => cinemas.dado ?? [], [cinemas.dado])
  const opcoes = useMemo(() => opcoesCinemas(lista, filtros.estado), [lista, filtros.estado])
  const visiveis = useMemo(() => filtrarCinemas(lista, filtros), [lista, filtros])
  const limpar = () => setFiltros(filtrosCinemasVazios)
  const comFiltros = filtros.estado !== '' || filtros.cidade !== ''

  if (filme.estado === 'carregando') {
    return (
      <div className="pagina">
        <p className="estado" role="status">
          Carregando filme…
        </p>
      </div>
    )
  }

  if (filme.estado === 'erro' || !filme.dado) {
    const naoEncontrado = filme.erro instanceof FilmeNaoEncontrado
    return (
      <div className="pagina">
        <LinkVoltar />
        {naoEncontrado ? (
          <div className="estado" role="alert">
            <h1>Filme não encontrado</h1>
            <p>Não encontramos este filme. Ele pode ter sido removido ou o endereço está incorreto.</p>
          </div>
        ) : (
          <div className="estado estado-erro" role="alert">
            <p>Não foi possível carregar o filme.</p>
            <button type="button" onClick={filme.recarregar}>
              Tentar novamente
            </button>
          </div>
        )}
      </div>
    )
  }

  return (
    <div className="pagina">
      <LinkVoltar />
      <CabecalhoFilme filme={filme.dado} />

      <section aria-labelledby="titulo-cinemas-do-filme" className="secao">
        <h2 id="titulo-cinemas-do-filme">Cinemas em cartaz</h2>
        {cinemas.estado === 'carregando' ? (
          <p className="estado" role="status">
            Carregando cinemas…
          </p>
        ) : cinemas.estado === 'erro' ? (
          <div className="estado estado-erro" role="alert">
            <p>Não foi possível carregar os cinemas.</p>
            <button type="button" onClick={cinemas.recarregar}>
              Tentar novamente
            </button>
          </div>
        ) : lista.length === 0 ? (
          <p className="estado">Este filme não está em cartaz em nenhum cinema.</p>
        ) : (
          <>
            <form className="filtros" role="search" aria-label="Filtrar cinemas" onSubmit={(e) => e.preventDefault()}>
              <CampoSelect
                rotulo="UF"
                valor={filtros.estado}
                todas="Todas as UFs"
                opcoes={opcoes.estados.map((e) => ({ valor: e, rotulo: e }))}
                onChange={(estado) => setFiltros(ajustarCidade(lista, { ...filtros, estado }))}
              />
              <CampoSelect
                rotulo="Cidade"
                valor={filtros.cidade}
                todas="Todas as cidades"
                opcoes={opcoes.cidades.map((cidade) => ({ valor: cidade, rotulo: cidade }))}
                onChange={(cidade) => setFiltros({ ...filtros, cidade })}
              />
              <button type="button" className="botao-secundario" onClick={limpar} disabled={!comFiltros}>
                Limpar filtros
              </button>
            </form>

            <p className="contagem" aria-live="polite">
              {visiveis.length} {visiveis.length === 1 ? 'cinema' : 'cinemas'}
            </p>

            <EstadoLista
              estado="pronto"
              erro=""
              total={lista.length}
              visiveis={visiveis.length}
              rotulo="cinemas"
              onTentarNovamente={cinemas.recarregar}
              onLimparFiltros={limpar}
            >
              <ul className="grade">
                {visiveis.map((cinema) => (
                  <CartaoCinemaDoFilme
                    key={cinema.id}
                    cinema={cinema}
                    onComprar={() => navegar(caminhoCompra(cinema.id, id))}
                  />
                ))}
              </ul>
            </EstadoLista>
          </>
        )}
      </section>
    </div>
  )
}
