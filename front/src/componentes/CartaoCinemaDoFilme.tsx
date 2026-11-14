import type { Cinema } from '../api/tipos'

interface Props {
  cinema: Cinema
  /** Chamado ao acionar "Comprar Ingressos" (abre a compra do filme neste cinema). */
  onComprar?: (cinema: Cinema) => void
}

/** Item da listagem de cinemas de um filme: dados do cinema e o botão de compra. */
export function CartaoCinemaDoFilme({ cinema, onComprar }: Props) {
  return (
    <li className="cartao cartao-cinema">
      {cinema.logo_url ? (
        <img className="cartao-logo" src={cinema.logo_url} alt={`Logo de ${cinema.nome}`} loading="lazy" />
      ) : (
        <div className="cartao-logo cartao-logo-vazio" aria-hidden="true">
          🎞️
        </div>
      )}
      <div className="cartao-corpo">
        <h3>{cinema.nome}</h3>
        <p className="cartao-meta">
          {cinema.cidade} / {cinema.estado}
        </p>
        <p className="cartao-sinopse">{cinema.endereco}</p>
        <button
          type="button"
          className="botao-comprar"
          aria-label={`Comprar Ingressos em ${cinema.nome}`}
          onClick={() => onComprar?.(cinema)}
        >
          Comprar Ingressos
        </button>
      </div>
    </li>
  )
}
