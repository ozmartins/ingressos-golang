import { useMemo, useState } from 'react'
import {
  buscarCinema,
  buscarSalas,
  buscarSessoes,
  buscarTodosFilmes,
  CinemaNaoEncontrado,
} from '../api/catalogo'
import type { Cinema, Filme, Sala, SessaoNaGrade } from '../api/tipos'
import { formatarDia } from '../dominio/formatar'
import {
  agruparPorFilme,
  diasDisponiveis,
  filtrarPorDia,
  montarSessoes,
  salasComSessoes,
} from '../dominio/programacao'
import { interceptarClique, navegar, veioDoApp } from '../hooks/useRota'
import { useRecurso } from '../hooks/useRecurso'
import { CabecalhoCinema } from './CabecalhoCinema'
import { CampoSelect } from './CampoSelect'
import { ListaSalas } from './ListaSalas'
import { ProgramacaoFilme } from './ProgramacaoFilme'

const CAMINHO_LISTA = '/?aba=cinemas'

export interface CarregadoresCinema {
  carregarCinema: (id: string) => Promise<Cinema>
  carregarSalas: (cinemaId: string) => Promise<Sala[]>
  carregarSessoes: (cinemaId: string) => Promise<SessaoNaGrade[]>
  carregarFilmes: () => Promise<Filme[]>
}

const PADRAO: CarregadoresCinema = {
  carregarCinema: buscarCinema,
  carregarSalas: buscarSalas,
  carregarSessoes: buscarSessoes,
  carregarFilmes: buscarTodosFilmes,
}

function voltarParaLista() {
  if (veioDoApp()) window.history.back()
  else navegar(CAMINHO_LISTA)
}

function LinkVoltar({ rotulo }: { rotulo: string }) {
  return (
    <a className="link-voltar" href={CAMINHO_LISTA} onClick={(e) => interceptarClique(e, voltarParaLista)}>
      ← {rotulo}
    </a>
  )
}

function Erro({ mensagem, onTentarNovamente }: { mensagem: string; onTentarNovamente: () => void }) {
  return (
    <div className="estado estado-erro" role="alert">
      <p>{mensagem}</p>
      <button type="button" onClick={onTentarNovamente}>
        Tentar novamente
      </button>
    </div>
  )
}

/** Dados de um cinema: salas e programação (somente filmes em cartaz). */
export function PaginaCinema({ id, carregadores }: { id: string; carregadores?: Partial<CarregadoresCinema> }) {
  const c = { ...PADRAO, ...carregadores }
  const cinema = useRecurso(() => c.carregarCinema(id), id)
  const salas = useRecurso(() => c.carregarSalas(id), id)
  const sessoes = useRecurso(async () => {
    const [grade, filmes] = await Promise.all([c.carregarSessoes(id), c.carregarFilmes()])
    return montarSessoes(grade, filmes)
  }, id)
  const [dia, setDia] = useState('')

  const exibidas = useMemo(() => sessoes.dado ?? [], [sessoes.dado])
  const dias = useMemo(() => diasDisponiveis(exibidas), [exibidas])
  const grupos = useMemo(() => agruparPorFilme(filtrarPorDia(exibidas, dia)), [exibidas, dia])
  const listaSalas = useMemo(() => salasComSessoes(salas.dado ?? [], exibidas), [salas.dado, exibidas])

  if (cinema.estado === 'carregando') {
    return (
      <div className="pagina">
        <p className="estado" role="status">
          Carregando cinema…
        </p>
      </div>
    )
  }

  if (cinema.estado === 'erro' || !cinema.dado || cinema.dado.ativo === false) {
    const naoEncontrado =
      cinema.estado === 'pronto' || cinema.erro instanceof CinemaNaoEncontrado
    return (
      <div className="pagina">
        <LinkVoltar rotulo="Voltar aos cinemas" />
        {naoEncontrado ? (
          <div className="estado" role="alert">
            <h1>Cinema não encontrado</h1>
            <p>Não encontramos este cinema. Ele pode ter sido removido ou o endereço está incorreto.</p>
          </div>
        ) : (
          <Erro mensagem="Não foi possível carregar o cinema." onTentarNovamente={cinema.recarregar} />
        )}
      </div>
    )
  }

  return (
    <div className="pagina">
      <LinkVoltar rotulo="Voltar aos cinemas" />
      <CabecalhoCinema cinema={cinema.dado} />

      <section aria-labelledby="titulo-salas" className="secao">
        <h2 id="titulo-salas">Salas</h2>
        {salas.estado === 'carregando' || sessoes.estado === 'carregando' ? (
          <p className="estado" role="status">
            Carregando salas…
          </p>
        ) : salas.estado === 'erro' ? (
          <Erro mensagem="Não foi possível carregar as salas." onTentarNovamente={salas.recarregar} />
        ) : (
          <ListaSalas salas={listaSalas} />
        )}
      </section>

      <section aria-labelledby="titulo-programacao" className="secao">
        <h2 id="titulo-programacao">Programação</h2>
        {sessoes.estado === 'carregando' ? (
          <p className="estado" role="status">
            Carregando programação…
          </p>
        ) : sessoes.estado === 'erro' ? (
          <Erro mensagem="Não foi possível carregar a programação." onTentarNovamente={sessoes.recarregar} />
        ) : exibidas.length === 0 ? (
          <p className="estado">Este cinema não tem sessões de filmes em cartaz programadas.</p>
        ) : (
          <>
            <div className="filtros">
              <CampoSelect
                rotulo="Dia"
                valor={dia}
                todas="Todos os dias"
                opcoes={dias.map((d) => ({ valor: d, rotulo: formatarDia(d) }))}
                onChange={setDia}
              />
            </div>
            <ul className="programacao">
              {grupos.map((grupo) => (
                <ProgramacaoFilme key={grupo.filme.id} programacao={grupo} />
              ))}
            </ul>
          </>
        )}
      </section>
    </div>
  )
}
