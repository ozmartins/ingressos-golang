import type { Cinema } from '../api/tipos'
import { caminhoCinema } from '../dominio/rotas'
import { interceptarClique, navegar } from '../hooks/useRota'

export function CartaoCinema({ cinema }: { cinema: Cinema }) {
  const caminho = caminhoCinema(cinema.id)
  return (
    <li className="cartao cartao-cinema cartao-clicavel">
      {cinema.logo_url ? (
        <img className="cartao-logo" src={cinema.logo_url} alt={`Logo de ${cinema.nome}`} loading="lazy" />
      ) : (
        <div className="cartao-logo cartao-logo-vazio" aria-hidden="true">
          🎞️
        </div>
      )}
      <div className="cartao-corpo">
        <h3>
          <a className="cartao-link" href={caminho} onClick={(e) => interceptarClique(e, () => navegar(caminho))}>
            {cinema.nome}
          </a>
        </h3>
        <p className="cartao-meta">
          {cinema.cidade} / {cinema.estado}
        </p>
        <p className="cartao-sinopse">{cinema.endereco}</p>
      </div>
    </li>
  )
}
