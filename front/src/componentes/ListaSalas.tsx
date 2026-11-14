import type { SalaComSessoes } from '../dominio/programacao'

export function ListaSalas({ salas }: { salas: SalaComSessoes[] }) {
  if (salas.length === 0) {
    return <p className="estado">Este cinema ainda não tem salas cadastradas.</p>
  }
  return (
    <ul className="lista-salas">
      {salas.map(({ sala, quantidadeSessoes }) => (
        <li key={sala.id} className="cartao sala">
          <h3>Sala {sala.numero}</h3>
          <p className="cartao-meta">
            {sala.tipo_tela} · {sala.capacidade_total} lugares
          </p>
          <p className="cartao-meta">
            {quantidadeSessoes === 0
              ? 'Sem sessões programadas'
              : `${quantidadeSessoes} ${quantidadeSessoes === 1 ? 'sessão' : 'sessões'}`}
          </p>
        </li>
      ))}
    </ul>
  )
}
