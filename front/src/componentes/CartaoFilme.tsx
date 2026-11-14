import type { Filme } from '../api/tipos'
import { formatarDuracao, ROTULO_STATUS } from '../dominio/formatar'

export function CartaoFilme({ filme }: { filme: Filme }) {
  return (
    <li className="cartao cartao-filme">
      {filme.imagem_url ? (
        <img className="cartao-imagem" src={filme.imagem_url} alt={`Pôster de ${filme.titulo}`} loading="lazy" />
      ) : (
        <div className="cartao-imagem cartao-imagem-vazia" aria-hidden="true">
          🎬
        </div>
      )}
      <div className="cartao-corpo">
        <h3>{filme.titulo}</h3>
        <p className="cartao-meta">
          {filme.genero} · {formatarDuracao(filme.duracao_minutos)}
        </p>
        <p className="cartao-selos">
          <span className="selo">Classificação: {filme.classificacao_etaria}</span>
          <span className={`selo selo-${filme.status.toLowerCase()}`}>{ROTULO_STATUS[filme.status]}</span>
        </p>
        {filme.sinopse ? <p className="cartao-sinopse">{filme.sinopse}</p> : null}
      </div>
    </li>
  )
}
