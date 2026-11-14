import { useRef, type KeyboardEvent } from 'react'
import { idAba, idPainel } from './idsAbas'

export interface DefinicaoAba<Id extends string> {
  id: Id
  rotulo: string
}

interface Props<Id extends string> {
  abas: DefinicaoAba<Id>[]
  ativa: Id
  onChange: (id: Id) => void
  rotulo: string
}

/** Lista de abas no padrão WAI-ARIA (setas, Home e End movem a seleção). */
export function Abas<Id extends string>({ abas, ativa, onChange, rotulo }: Props<Id>) {
  const refs = useRef<Record<string, HTMLButtonElement | null>>({})

  function aoTeclar(evento: KeyboardEvent, indice: number) {
    let destino = -1
    if (evento.key === 'ArrowRight') destino = (indice + 1) % abas.length
    else if (evento.key === 'ArrowLeft') destino = (indice - 1 + abas.length) % abas.length
    else if (evento.key === 'Home') destino = 0
    else if (evento.key === 'End') destino = abas.length - 1
    if (destino < 0) return
    evento.preventDefault()
    onChange(abas[destino].id)
    refs.current[abas[destino].id]?.focus()
  }

  return (
    <div role="tablist" aria-label={rotulo} className="abas">
      {abas.map((aba, indice) => {
        const selecionada = aba.id === ativa
        return (
          <button
            key={aba.id}
            ref={(el) => {
              refs.current[aba.id] = el
            }}
            type="button"
            role="tab"
            id={idAba(aba.id)}
            aria-selected={selecionada}
            aria-controls={idPainel(aba.id)}
            tabIndex={selecionada ? 0 : -1}
            className={selecionada ? 'aba aba-ativa' : 'aba'}
            onClick={() => onChange(aba.id)}
            onKeyDown={(e) => aoTeclar(e, indice)}
          >
            {aba.rotulo}
          </button>
        )
      })}
    </div>
  )
}
