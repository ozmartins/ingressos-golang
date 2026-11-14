import type { Cinema } from '../api/tipos'

export function CabecalhoCinema({ cinema }: { cinema: Cinema }) {
  return (
    <header className="cinema-cabecalho">
      {cinema.logo_url ? (
        <img className="cinema-logo" src={cinema.logo_url} alt={`Logo de ${cinema.nome}`} />
      ) : (
        <div className="cinema-logo cinema-logo-vazio" aria-hidden="true">
          🎞️
        </div>
      )}
      <div>
        <h1>{cinema.nome}</h1>
        <p className="cartao-meta">
          {cinema.cidade} / {cinema.estado}
        </p>
        <p>{cinema.endereco}</p>
      </div>
    </header>
  )
}
