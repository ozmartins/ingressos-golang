import { formatarDia, formatarDuracao, formatarHora, formatarPreco } from '../dominio/formatar'
import type { Filme } from '../api/tipos'
import type { ProgramacaoFilme as Programacao } from '../dominio/programacao'

const ROTULO_IDIOMA = { DUBLADO: 'Dublado', LEGENDADO: 'Legendado' } as const

interface Props {
  programacao: Programacao
  /** Chamado ao acionar "Comprar Ingressos"; o fluxo de compra ainda não existe nesta página. */
  onComprar?: (filme: Filme) => void
}

export function ProgramacaoFilme({ programacao, onComprar }: Props) {
  const { filme, sessoes } = programacao
  return (
    <li className="programacao-filme">
      {filme.imagem_url ? (
        <img className="programacao-imagem" src={filme.imagem_url} alt={`Pôster de ${filme.titulo}`} loading="lazy" />
      ) : (
        <div className="programacao-imagem programacao-imagem-vazia" aria-hidden="true">
          🎬
        </div>
      )}
      <div className="programacao-corpo">
        <h3>{filme.titulo}</h3>
        <p className="cartao-meta">
          {filme.genero} · {formatarDuracao(filme.duracao_minutos)} · Classificação: {filme.classificacao_etaria}
        </p>
        <ul className="sessoes">
          {sessoes.map((sessao) => (
            <li key={sessao.id} className="sessao">
              <time dateTime={sessao.inicio.toISOString()}>
                {formatarDia(sessao.dia)} · {formatarHora(sessao.inicio)}
              </time>
              <span>
                Sala {sessao.salaNumero} · {sessao.tipoTela}
              </span>
              <span>{ROTULO_IDIOMA[sessao.idioma]}</span>
              <strong>{formatarPreco(sessao.preco)}</strong>
            </li>
          ))}
        </ul>
        <button type="button" className="botao-comprar" onClick={() => onComprar?.(filme)}>
          Comprar Ingressos
        </button>
      </div>
    </li>
  )
}
