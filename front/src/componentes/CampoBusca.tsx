import { useId } from 'react'

interface Props {
  rotulo: string
  valor: string
  onChange: (valor: string) => void
}

export function CampoBusca({ rotulo, valor, onChange }: Props) {
  const id = useId()
  return (
    <div className="campo campo-busca">
      <label htmlFor={id}>{rotulo}</label>
      <input id={id} type="search" value={valor} autoComplete="off" onChange={(e) => onChange(e.target.value)} />
    </div>
  )
}
