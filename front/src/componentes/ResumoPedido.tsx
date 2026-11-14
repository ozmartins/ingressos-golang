import type { Filme, SessaoNaGrade } from '../api/tipos'
import { formatarCentavos, formatarDiaDe, formatarHora, ROTULO_IDIOMA } from '../dominio/formatar'
import type { Pedido } from '../dominio/pedido'

interface Props {
  filme: Filme
  sessao: SessaoNaGrade
  selecionados: readonly string[]
  pedido: Pedido
  reservando: boolean
  /** Reserva já feita: a ação "Continuar" some. */
  reservado: boolean
  onContinuar: () => void
}

/** "Resumo do pedido": filme, sessão, assentos escolhidos, taxa e total. */
export function ResumoPedido({ filme, sessao, selecionados, pedido, reservando, reservado, onContinuar }: Props) {
  const inicio = new Date(sessao.data_hora_inicio)
  return (
    <aside className="resumo" aria-labelledby="titulo-resumo">
      <h2 id="titulo-resumo">Resumo do pedido</h2>
      <div className="resumo-filme">
        {filme.imagem_url ? (
          <img className="resumo-poster" src={filme.imagem_url} alt={`Pôster de ${filme.titulo}`} />
        ) : (
          <div className="resumo-poster programacao-imagem-vazia" aria-hidden="true">
            🎬
          </div>
        )}
        <div>
          <h3>{filme.titulo}</h3>
          <p className="cartao-meta">
            Classificação: {filme.classificacao_etaria} · {sessao.tipo_tela} · {ROTULO_IDIOMA[sessao.idioma]}
          </p>
          <p className="cartao-meta">{sessao.cinema_nome}</p>
        </div>
      </div>
      <p className="resumo-sessao">
        <span>Sala {sessao.sala_numero}</span>
        <span>
          {formatarDiaDe(inicio)} <strong>{formatarHora(inicio)}</strong>
        </span>
      </p>

      <p className="resumo-assentos">
        {selecionados.length === 0 ? 'Nenhum assento selecionado.' : `Assentos: ${selecionados.join(', ')}`}
      </p>

      <dl className="resumo-totais">
        <div>
          <dt>Itens</dt>
          <dd>{pedido.itens}</dd>
        </div>
        <div>
          <dt>Total taxa</dt>
          <dd>{formatarCentavos(pedido.taxaCentavos)}</dd>
        </div>
        <div className="resumo-total">
          <dt>Total</dt>
          <dd>{formatarCentavos(pedido.totalCentavos)}</dd>
        </div>
      </dl>

      {reservado ? null : (
        <button type="button" className="botao-comprar resumo-continuar" disabled={pedido.itens === 0 || reservando} onClick={onContinuar}>
          {reservando ? 'Reservando…' : 'Continuar'}
        </button>
      )}
    </aside>
  )
}
