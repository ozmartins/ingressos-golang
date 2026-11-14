import { useEffect, useMemo, useRef, useState } from 'react'
import { buscarCinema, buscarSessoes, buscarTodosFilmes, CinemaNaoEncontrado } from '../api/catalogo'
import {
  bloquearPoltronas,
  buscarMapa,
  ConflitoPoltronas,
  LimiteExcedido,
  NaoAutenticado,
  SessaoIndisponivel,
} from '../api/estoque'
import type { Cinema, Filme, Poltrona, RespostaBloqueio, SessaoNaGrade } from '../api/tipos'
import { formatarHora } from '../dominio/formatar'
import { limparSessao, type Sessao } from '../dominio/keycloak'
import { rotulosLivres } from '../dominio/mapaAssentos'
import { alternarAssento, calcularPedido, LIMITE_ASSENTOS, precoEmCentavos } from '../dominio/pedido'
import { caminhoCinema } from '../dominio/rotas'
import { sessaoPadrao, sessoesDoFilme } from '../dominio/sessoesDoFilme'
import { irParaLogin, useGuardaLogin } from '../hooks/useGuardaLogin'
import { useRecurso } from '../hooks/useRecurso'
import { interceptarClique, navegar, veioDoApp } from '../hooks/useRota'
import { CabecalhoSessao } from './CabecalhoSessao'
import { MapaAssentos } from './MapaAssentos'
import { ResumoPedido } from './ResumoPedido'

export interface CarregadoresCompra {
  carregarCinema: (id: string) => Promise<Cinema>
  carregarFilmes: () => Promise<Filme[]>
  carregarSessoes: (cinemaId: string) => Promise<SessaoNaGrade[]>
  carregarMapa: (sessaoId: string, token: string) => Promise<Poltrona[]>
  reservar: (sessaoId: string, rotulos: string[], token: string) => Promise<RespostaBloqueio>
  /** Leva ao login com retorno à página atual. */
  irParaLogin: (retorno: string) => void
}

const PADRAO: CarregadoresCompra = {
  carregarCinema: buscarCinema,
  carregarFilmes: buscarTodosFilmes,
  carregarSessoes: buscarSessoes,
  carregarMapa: buscarMapa,
  reservar: bloquearPoltronas,
  irParaLogin,
}

interface Props {
  cinemaId: string
  filmeId: string
  /** Sessão de login; `null` redireciona ao login antes de qualquer consulta. */
  sessao: Sessao | null
  carregadores?: Partial<CarregadoresCompra>
}

const chaveSelecao = (sessaoId: string) => `compra:selecao:${sessaoId}`

function lerSelecaoSalva(sessaoId: string): string[] {
  try {
    const bruto = sessionStorage.getItem(chaveSelecao(sessaoId))
    const lista: unknown = bruto === null ? [] : JSON.parse(bruto)
    return Array.isArray(lista) ? lista.filter((r): r is string => typeof r === 'string') : []
  } catch {
    return []
  }
}

function guardarSelecao(sessaoId: string, rotulos: string[]) {
  try {
    if (rotulos.length === 0) sessionStorage.removeItem(chaveSelecao(sessaoId))
    else sessionStorage.setItem(chaveSelecao(sessaoId), JSON.stringify(rotulos))
  } catch {
    // sem armazenamento disponível: a seleção só não sobrevive ao login
  }
}

function voltarAoCinema(cinemaId: string) {
  if (veioDoApp()) window.history.back()
  else navegar(caminhoCinema(cinemaId))
}

function LinkVoltar({ cinemaId }: { cinemaId: string }) {
  return (
    <a className="link-voltar" href={caminhoCinema(cinemaId)} onClick={(e) => interceptarClique(e, () => voltarAoCinema(cinemaId))}>
      ← Voltar ao cinema
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

/** Página de compra: exige login e, estando logado, mostra a seleção de assentos. */
export function PaginaCompra({ cinemaId, filmeId, sessao, carregadores }: Props) {
  const c = { ...PADRAO, ...carregadores }
  const exigeLogin = useGuardaLogin(sessao, c.irParaLogin)
  if (exigeLogin || sessao === null) {
    return (
      <div className="pagina">
        <p className="estado" role="status">
          Redirecionando para o login…
        </p>
      </div>
    )
  }
  return <CompraAutenticada cinemaId={cinemaId} filmeId={filmeId} token={sessao.accessToken} c={c} />
}

interface Interna {
  cinemaId: string
  filmeId: string
  token: string
  c: CarregadoresCompra
}

type Reserva =
  | { estado: 'ocioso' }
  | { estado: 'reservando' }
  | { estado: 'reservado'; rotulos: string[]; expiraEm: Date }

function CompraAutenticada({ cinemaId, filmeId, token, c }: Interna) {
  const chave = `${cinemaId}/${filmeId}`
  const dados = useRecurso(async () => {
    const [cinema, filmes, grade] = await Promise.all([c.carregarCinema(cinemaId), c.carregarFilmes(), c.carregarSessoes(cinemaId)])
    return { cinema, filme: filmes.find((f) => f.id === filmeId), grade }
  }, chave)

  const [idPedido, setIdPedido] = useState<string | null>(() => new URLSearchParams(window.location.search).get('sessao'))
  const sessoes = useMemo(() => (dados.dado ? sessoesDoFilme(dados.dado.grade, filmeId) : []), [dados.dado, filmeId])
  const atual = sessaoPadrao(sessoes, idPedido)
  const atualId = atual?.id

  // Reflete a sessão escolhida no endereço, para o link ser compartilhável.
  useEffect(() => {
    if (!atualId) return
    const consulta = new URLSearchParams(window.location.search)
    if (consulta.get('sessao') === atualId) return
    consulta.set('sessao', atualId)
    window.history.replaceState(window.history.state, '', `${window.location.pathname}?${consulta}`)
  }, [atualId])

  const mapa = useRecurso(() => (atualId ? c.carregarMapa(atualId, token) : Promise.resolve([])), atualId ?? '')

  const [selecao, setSelecao] = useState<{ sessaoId: string; rotulos: string[] } | null>(null)
  const [reserva, setReserva] = useState<Reserva>({ estado: 'ocioso' })
  const [aviso, setAviso] = useState<string | null>(null)
  const [erroReserva, setErroReserva] = useState<string | null>(null)

  const livres = useMemo(() => rotulosLivres(mapa.dado ?? []), [mapa.dado])
  const bruta = selecao && selecao.sessaoId === atualId ? selecao.rotulos : atualId ? lerSelecaoSalva(atualId) : []
  // Assentos que deixaram de estar livres (após recarregar o mapa) saem da seleção.
  const selecionados = mapa.dado ? bruta.filter((r) => livres.has(r)) : bruta

  // Token recusado ao carregar o mapa: entrar de novo e voltar a esta página.
  const mapaNaoAutenticado = mapa.estado === 'erro' && mapa.erro instanceof NaoAutenticado
  const irParaLoginRef = useRef(c.irParaLogin)
  useEffect(() => {
    irParaLoginRef.current = c.irParaLogin
  })
  useEffect(() => {
    if (!mapaNaoAutenticado) return
    limparSessao()
    irParaLoginRef.current(window.location.pathname + window.location.search)
  }, [mapaNaoAutenticado])

  if (dados.estado === 'carregando') {
    return (
      <div className="pagina">
        <p className="estado" role="status">
          Carregando…
        </p>
      </div>
    )
  }

  if (dados.estado === 'erro') {
    const naoEncontrado = dados.erro instanceof CinemaNaoEncontrado
    return (
      <div className="pagina">
        <LinkVoltar cinemaId={cinemaId} />
        {naoEncontrado ? (
          <div className="estado" role="alert">
            <h1>Cinema não encontrado</h1>
            <p>Não encontramos este cinema. Ele pode ter sido removido ou o endereço está incorreto.</p>
          </div>
        ) : (
          <Erro mensagem="Não foi possível carregar os dados da sessão." onTentarNovamente={dados.recarregar} />
        )}
      </div>
    )
  }

  const { cinema, filme } = dados.dado!
  if (cinema.ativo === false || !filme) {
    return (
      <div className="pagina">
        <LinkVoltar cinemaId={cinemaId} />
        <div className="estado" role="alert">
          <h1>Filme ou cinema não encontrado</h1>
          <p>Não encontramos esta sessão. O endereço pode estar incorreto.</p>
        </div>
      </div>
    )
  }

  if (!atual) {
    return (
      <div className="pagina">
        <LinkVoltar cinemaId={cinemaId} />
        <div className="estado">
          <h1>{filme.titulo}</h1>
          <p>Não há sessões disponíveis para este filme neste cinema.</p>
        </div>
      </div>
    )
  }

  const pedido = calcularPedido(precoEmCentavos(atual.preco_base), selecionados)
  const reservado = reserva.estado === 'reservado'
  const reservando = reserva.estado === 'reservando'
  const mapaIndisponivel =
    mapa.estado === 'erro'
      ? mapa.erro instanceof SessaoIndisponivel
      : mapa.estado === 'pronto' && (mapa.dado?.length ?? 0) === 0

  function trocarSessao(id: string) {
    setIdPedido(id)
    setSelecao(null)
    setAviso(null)
    setErroReserva(null)
  }

  function alternar(rotulo: string) {
    if (!atual) return
    const r = alternarAssento(selecionados, rotulo, livres)
    setAviso(r.aviso === 'limite' ? `Você pode escolher no máximo ${LIMITE_ASSENTOS} assentos por pedido.` : null)
    setErroReserva(null)
    guardarSelecao(atual.id, r.rotulos)
    setSelecao({ sessaoId: atual.id, rotulos: r.rotulos })
  }

  function continuar() {
    if (!atual || selecionados.length === 0 || reservando) return
    const sessaoId = atual.id
    const rotulos = selecionados
    setReserva({ estado: 'reservando' })
    setAviso(null)
    setErroReserva(null)
    c.reservar(sessaoId, rotulos, token)
      .then((r) => {
        guardarSelecao(sessaoId, [])
        setReserva({ estado: 'reservado', rotulos, expiraEm: new Date(r.expira_em) })
      })
      .catch((erro: unknown) => {
        setReserva({ estado: 'ocioso' })
        if (erro instanceof NaoAutenticado) {
          guardarSelecao(sessaoId, rotulos)
          limparSessao()
          c.irParaLogin(window.location.pathname + window.location.search)
        } else if (erro instanceof ConflitoPoltronas) {
          setErroReserva('Alguns assentos escolhidos acabaram de ser reservados por outra pessoa. O mapa foi atualizado; confira sua seleção.')
          mapa.recarregar()
        } else if (erro instanceof LimiteExcedido) {
          setErroReserva(`Você pode escolher no máximo ${erro.limite ?? LIMITE_ASSENTOS} assentos por pedido.`)
        } else if (erro instanceof SessaoIndisponivel) {
          setErroReserva('Esta sessão não aceita reservas no momento.')
        } else {
          setErroReserva('Não foi possível reservar os assentos. Tente novamente.')
        }
      })
  }

  return (
    <div className="pagina pagina-compra">
      <LinkVoltar cinemaId={cinemaId} />
      <h1>{filme.titulo}</h1>
      <CabecalhoSessao sessao={atual} sessoes={sessoes} onTrocar={trocarSessao} desabilitado={reservando || reservado} />

      <div className="compra-corpo">
        <section aria-label="Assentos" className="compra-mapa">
          {mapa.estado === 'carregando' && !mapa.dado ? (
            <p className="estado" role="status">
              Carregando assentos…
            </p>
          ) : mapaIndisponivel ? (
            <p className="estado" role="alert">
              A seleção de assentos não está disponível para esta sessão.
            </p>
          ) : mapa.estado === 'erro' ? (
            <Erro mensagem="Não foi possível carregar o mapa de assentos." onTentarNovamente={mapa.recarregar} />
          ) : (
            <MapaAssentos
              poltronas={mapa.dado ?? []}
              selecionados={selecionados}
              onAlternar={alternar}
              desabilitado={reservando || reservado}
            />
          )}
          {aviso ? (
            <p className="aviso" role="alert">
              {aviso}
            </p>
          ) : null}
          {erroReserva ? (
            <p className="estado estado-erro" role="alert">
              {erroReserva}
            </p>
          ) : null}
        </section>

        <div className="compra-lateral">
          <ResumoPedido
            filme={filme}
            sessao={atual}
            selecionados={selecionados}
            pedido={pedido}
            reservando={reservando}
            reservado={reservado}
            onContinuar={continuar}
          />
          {reserva.estado === 'reservado' ? (
            <div className="estado reserva-confirmada" role="status">
              <h2>Assentos reservados</h2>
              <p>{reserva.rotulos.join(', ')}</p>
              <p>Reserva válida até {formatarHora(reserva.expiraEm)}.</p>
            </div>
          ) : null}
        </div>
      </div>
    </div>
  )
}
