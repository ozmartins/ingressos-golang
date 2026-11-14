import { useId, useMemo, useState } from 'react'
import type { Poltrona } from '../api/tipos'
import { agruparPorFileira, rotuloAcessivel, situacaoDoAssento } from '../dominio/mapaAssentos'

interface Props {
  poltronas: Poltrona[]
  selecionados: readonly string[]
  onAlternar: (rotulo: string) => void
  /** Bloqueia todos os assentos (reserva feita ou em andamento). */
  desabilitado?: boolean
}

const SIMBOLO_TIPO = { NORMAL: '', PCD: '♿', NAMORADEIRA: '♥' } as const

const LEGENDA = [
  { classe: 'livre', texto: 'Livre' },
  { classe: 'selecionado', texto: 'Selecionado' },
  { classe: 'ocupado', texto: 'Ocupado' },
  { classe: 'livre tipo-pcd', texto: 'Acessível', simbolo: '♿' },
  { classe: 'livre tipo-namoradeira', texto: 'Namoradeira', simbolo: '♥' },
]

/** Mapa da sala: fileiras (A mais perto da tela), tela, legenda e alternância dos números. */
export function MapaAssentos({ poltronas, selecionados, onAlternar, desabilitado = false }: Props) {
  const idSwitch = useId()
  const [numeros, setNumeros] = useState(false)
  // A última fileira no topo; a A, junto da tela.
  const fileiras = useMemo(() => agruparPorFileira(poltronas).reverse(), [poltronas])

  return (
    <div className="mapa-assentos">
      <div className="mapa-rolagem">
        <div className="mapa-grade" role="group" aria-label="Mapa de assentos">
          {fileiras.map((fileira) => (
            <div key={fileira.letra} className="mapa-fileira" role="group" aria-label={`Fileira ${fileira.letra}`}>
              <span className="mapa-letra" aria-hidden="true">
                {fileira.letra}
              </span>
              <div className="mapa-assentos-fileira">
                {fileira.poltronas.map((p) => {
                  const situacao = situacaoDoAssento(p, selecionados)
                  const bloqueado = desabilitado || (situacao !== 'livre' && situacao !== 'selecionado')
                  return (
                    <button
                      key={p.rotulo}
                      type="button"
                      className={`assento assento-${situacao} tipo-${p.tipo.toLowerCase()}`}
                      aria-label={rotuloAcessivel(p, situacao)}
                      aria-pressed={situacao === 'selecionado'}
                      disabled={bloqueado}
                      onClick={() => onAlternar(p.rotulo)}
                    >
                      {numeros ? p.numero : situacao === 'ocupado' ? '✕' : SIMBOLO_TIPO[p.tipo]}
                    </button>
                  )
                })}
              </div>
              <span className="mapa-letra" aria-hidden="true">
                {fileira.letra}
              </span>
            </div>
          ))}
        </div>
      </div>

      <div className="mapa-tela" aria-hidden="true">
        TELA
      </div>

      <label className="mapa-numeros" htmlFor={idSwitch}>
        <input id={idSwitch} type="checkbox" role="switch" checked={numeros} onChange={(e) => setNumeros(e.target.checked)} />
        Exibir números dos assentos
      </label>

      <ul className="mapa-legenda" aria-label="Legenda">
        {LEGENDA.map((item) => (
          <li key={item.texto}>
            <span className={`assento assento-amostra assento-${item.classe}`} aria-hidden="true">
              {item.simbolo ?? (item.texto === 'Ocupado' ? '✕' : '')}
            </span>
            {item.texto}
          </li>
        ))}
      </ul>
    </div>
  )
}
