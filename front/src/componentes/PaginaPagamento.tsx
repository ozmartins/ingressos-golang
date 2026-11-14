import { useEffect, useRef, useState } from 'react'
import { cartaoRecusado, type DadosCartao } from '../dominio/cartao'
import { montarIngressos, type IngressoPdf } from '../dominio/ingressoPdf'
import { idDoUsuario, type Sessao } from '../dominio/keycloak'
import { limparReserva, lerReserva, segundosRestantes, type ReservaPagamento } from '../dominio/reservaPagamento'
import { caminhoCompra } from '../dominio/rotas'
import { irParaLogin, useGuardaLogin } from '../hooks/useGuardaLogin'
import { interceptarClique, navegar } from '../hooks/useRota'
import { baixarIngressoPdf } from '../infra/gerarPdf'
import { FormularioCartao } from './FormularioCartao'
import { ResumoReserva } from './ResumoReserva'

/** Tempo simulado de processamento do pagamento. */
export const ATRASO_PAGAMENTO_MS = 2000

interface Props {
  cinemaId: string
  filmeId: string
  /** Sessão de login; `null` redireciona ao login. */
  sessao: Sessao | null
  /** Injetáveis para testes. */
  irParaLogin?: (retorno: string) => void
  atrasoMs?: number
  agora?: () => number
  /** Gera e baixa o PDF do ingresso (por padrão, `baixarIngressoPdf`). */
  gerarPdf?: (ingresso: IngressoPdf) => Promise<void>
}

type Estado = 'ocioso' | 'processando' | 'aprovado' | 'recusado'

const link = (href: string, texto: string, className?: string) => (
  <a className={className} href={href} onClick={(e) => interceptarClique(e, () => navegar(href))}>
    {texto}
  </a>
)

/** Pagamento com cartão (simulado): exige login e uma reserva em andamento. Nenhum dado de cartão sai daqui. */
export function PaginaPagamento({ cinemaId, filmeId, sessao, irParaLogin: ir = irParaLogin, atrasoMs = ATRASO_PAGAMENTO_MS, agora = Date.now, gerarPdf = baixarIngressoPdf }: Props) {
  const exigeLogin = useGuardaLogin(sessao, ir)
  const [reserva] = useState<ReservaPagamento | null>(() => lerReserva(cinemaId, filmeId, idDoUsuario(sessao?.accessToken ?? null)))
  const [estado, setEstado] = useState<Estado>('ocioso')
  const [pdf, setPdf] = useState<'ocioso' | 'gerando' | 'erro'>('ocioso')
  const gerandoPdf = useRef(false)
  const [restante, setRestante] = useState(() => (reserva ? segundosRestantes(reserva.expiraEm, agora()) : 0))
  const agoraRef = useRef(agora)
  const temporizador = useRef<ReturnType<typeof setTimeout> | undefined>(undefined)

  useEffect(() => {
    agoraRef.current = agora
  })

  // Contagem regressiva do prazo da reserva (para ao concluir o pagamento).
  const contando = reserva !== null && estado !== 'aprovado'
  useEffect(() => {
    if (!contando) return
    const id = setInterval(() => setRestante(segundosRestantes(reserva.expiraEm, agoraRef.current())), 1000)
    return () => clearInterval(id)
  }, [contando, reserva])

  useEffect(() => () => clearTimeout(temporizador.current), [])

  if (exigeLogin || sessao === null) {
    return (
      <div className="pagina">
        <p className="estado" role="status">
          Redirecionando para o login…
        </p>
      </div>
    )
  }

  if (reserva === null) {
    return (
      <div className="pagina">
        <div className="estado" role="alert">
          <h1>Nenhuma reserva em andamento</h1>
          <p>Escolha seus assentos para poder pagar.</p>
          <p>
            {link(caminhoCompra(cinemaId, filmeId), 'Escolher assentos')} · {link('/', 'Página inicial')}
          </p>
        </div>
      </div>
    )
  }

  const expirada = estado !== 'aprovado' && restante <= 0
  const voltar = caminhoCompra(cinemaId, filmeId, reserva.sessaoId)

  async function baixarPdf() {
    if (!reserva || gerandoPdf.current) return
    gerandoPdf.current = true
    setPdf('gerando')
    try {
      await gerarPdf(montarIngressos(reserva))
      setPdf('ocioso')
    } catch {
      setPdf('erro')
    } finally {
      gerandoPdf.current = false
    }
  }

  function pagar(dados: DadosCartao) {
    if (!reserva || estado === 'processando' || estado === 'aprovado') return
    if (segundosRestantes(reserva.expiraEm, agoraRef.current()) <= 0) {
      setRestante(0)
      return
    }
    setEstado('processando')
    temporizador.current = setTimeout(() => {
      if (segundosRestantes(reserva.expiraEm, agoraRef.current()) <= 0) {
        setRestante(0)
        setEstado('ocioso')
      } else if (cartaoRecusado(dados.numero)) {
        setEstado('recusado')
      } else {
        limparReserva()
        setEstado('aprovado')
      }
    }, atrasoMs)
  }

  return (
    <div className="pagina pagina-pagamento">
      {link(voltar, '← Voltar aos assentos', 'link-voltar')}
      <h1>Pagamento</h1>

      <div className="compra-corpo">
        <section aria-label="Pagamento" className="compra-mapa">
          {estado === 'aprovado' ? (
            <div className="estado reserva-confirmada" role="status">
              <h2>Pagamento aprovado</h2>
              <p>Assentos {reserva.rotulos.join(', ')} confirmados (simulação).</p>
              <p>
                <button type="button" className="botao-comprar" onClick={() => void baixarPdf()} disabled={pdf === 'gerando'} aria-busy={pdf === 'gerando'}>
                  {pdf === 'gerando' ? 'Gerando PDF…' : 'Baixar ingresso (PDF)'}
                </button>
              </p>
              {pdf === 'erro' ? (
                <p className="estado-erro" role="alert">
                  Não foi possível gerar o PDF. Tente novamente.
                </p>
              ) : null}
              <p>{link('/', 'Voltar à página inicial')}</p>
            </div>
          ) : expirada ? (
            <div className="estado estado-erro" role="alert">
              <h2>Reserva expirada</h2>
              <p>O prazo da reserva terminou e os assentos foram liberados.</p>
              <p>{link(caminhoCompra(cinemaId, filmeId), 'Escolher assentos')}</p>
            </div>
          ) : (
            <>
              {estado === 'recusado' ? (
                <div className="estado estado-erro" role="alert">
                  <h2>Pagamento recusado</h2>
                  <p>O cartão informado foi recusado.</p>
                  <button type="button" onClick={() => setEstado('ocioso')}>
                    Tentar outro cartão
                  </button>
                </div>
              ) : (
                <FormularioCartao onPagar={pagar} desabilitado={false} processando={estado === 'processando'} />
              )}
            </>
          )}
        </section>
        <div className="compra-lateral">
          <ResumoReserva reserva={reserva} segundosRestantes={estado === 'aprovado' ? null : restante} />
        </div>
      </div>
    </div>
  )
}
