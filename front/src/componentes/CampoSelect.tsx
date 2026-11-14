import { useId } from 'react'

interface Opcao {
  valor: string
  rotulo: string
}

interface Props {
  rotulo: string
  valor: string
  /** Texto da opção que representa "sem restrição". */
  todas: string
  opcoes: Opcao[]
  onChange: (valor: string) => void
  desabilitado?: boolean
}

export function CampoSelect({ rotulo, valor, todas, opcoes, onChange, desabilitado }: Props) {
  const id = useId()
  return (
    <div className="campo">
      <label htmlFor={id}>{rotulo}</label>
      <select id={id} value={valor} disabled={desabilitado} onChange={(e) => onChange(e.target.value)}>
        <option value="">{todas}</option>
        {opcoes.map((opcao) => (
          <option key={opcao.valor} value={opcao.valor}>
            {opcao.rotulo}
          </option>
        ))}
      </select>
    </div>
  )
}
