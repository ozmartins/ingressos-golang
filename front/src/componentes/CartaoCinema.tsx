import type { Cinema } from '../api/tipos'

export function CartaoCinema({ cinema }: { cinema: Cinema }) {
  return (
    <li className="cartao cartao-cinema">
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
