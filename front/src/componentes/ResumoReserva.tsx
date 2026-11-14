import { formatarCentavos, formatarDiaDe, formatarHora } from '../dominio/formatar'
import { formatarContagem, type ReservaPagamento } from '../dominio/reservaPagamento'

interface Props {
  reserva: ReservaPagamento
  /** Segundos até o fim da reserva; `null` oculta a contagem (pagamento já concluído). */
  segundosRestantes: number | null
}

/** Resumo do pedido reservado, mostrado ao lado do formulário de pagamento. */
export function ResumoReserva({ reserva, segundosRestantes }: Props) {
  const inicio = new Date(reserva.inicio)
  return (
    <aside className="resumo" aria-labelledby="titulo-resumo-reserva">
      <h2 id="titulo-resumo-reserva">Resumo do pedido</h2>
      <div className="resumo-filme">
        {reserva.posterUrl ? (
          <img className="resumo-poster" src={reserva.posterUrl} alt={`Pôster de ${reserva.filmeTitulo}`} />
        ) : (
          <div className="resumo-poster programacao-imagem-vazia" aria-hidden="true">
            🎬
          </div>
        )}
        <div>
          <h3>{reserva.filmeTitulo}</h3>
          <p className="cartao-meta">{reserva.cinemaNome}</p>
        </div>
      </div>
      <p className="resumo-sessao">
        <span>Sala {reserva.sala}</span>
        <span>
          {formatarDiaDe(inicio)} <strong>{formatarHora(inicio)}</strong>
        </span>
      </p>
      <p className="resumo-assentos">Assentos: {reserva.rotulos.join(', ')}</p>
      <dl className="resumo-totais">
        <div>
          <dt>Itens</dt>
          <dd>{reserva.rotulos.length}</dd>
        </div>
        <div>
          <dt>Total taxa</dt>
          <dd>{formatarCentavos(reserva.taxaCentavos)}</dd>
        </div>
        <div className="resumo-total">
          <dt>Total</dt>
          <dd>{formatarCentavos(reserva.totalCentavos)}</dd>
        </div>
      </dl>
      {segundosRestantes === null ? null : (
        <p className="resumo-prazo">
          Reserva expira em{' '}
          <strong role="timer" aria-live="off">
            {formatarContagem(segundosRestantes)}
          </strong>
        </p>
      )}
    </aside>
  )
}
