import type { Cinema } from '../api/tipos'

export function CartaoCinema({ cinema }: { cinema: Cinema }) {
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
      </div>
    </li>
  )
}
