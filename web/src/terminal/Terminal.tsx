import { useEffect, useRef, useState, type KeyboardEvent } from 'react'
import { executar } from './comandos'
import { adicionar, anterior, criar, proximo } from './historico'
import { adicionarLinhas, type Linha } from './linhas'

const PROMPT = '>'

const BOAS_VINDAS: Linha[] = [
  { tipo: 'sistema', texto: 'TERMINAL RETRO v1.0' },
  { tipo: 'sistema', texto: 'Digite "help" para ver os comandos disponiveis.' },
  { tipo: 'sistema', texto: '' },
]

export default function Terminal() {
  const [linhas, setLinhas] = useState<Linha[]>(BOAS_VINDAS)
  const [valor, setValor] = useState('')
  const [historico, setHistorico] = useState(criar)
  const campo = useRef<HTMLInputElement>(null)
  const saida = useRef<HTMLDivElement>(null)

  useEffect(() => {
    const el = saida.current
    if (el) el.scrollTop = el.scrollHeight
  }, [linhas])

  function enviar() {
    const resultado = executar(valor, new Date())
    const eco: Linha = { tipo: 'entrada', texto: `${PROMPT} ${valor}` }
    setLinhas((atuais) =>
      resultado.limpar
        ? []
        : adicionarLinhas(atuais, valor.trim() === '' ? [eco] : [eco, ...resultado.linhas]),
    )
    setHistorico((h) => adicionar(h, valor))
    setValor('')
  }

  function aoPressionar(e: KeyboardEvent<HTMLInputElement>) {
    if (e.key === 'Enter') {
      e.preventDefault()
      enviar()
    } else if (e.key === 'ArrowUp') {
      e.preventDefault()
      const r = anterior(historico)
      setHistorico(r.historico)
      setValor(r.valor)
    } else if (e.key === 'ArrowDown') {
      e.preventDefault()
      const r = proximo(historico)
      setHistorico(r.historico)
      setValor(r.valor)
    }
  }

  function devolverFoco() {
    if (window.getSelection()?.toString()) return
    campo.current?.focus()
  }

  return (
    <div className="terminal" onClick={devolverFoco}>
      <div className="saida" ref={saida} role="log" aria-live="polite">
        {linhas.map((l, i) => (
          <div key={i} className={`linha linha-${l.tipo}`}>
            {l.texto === '' ? ' ' : l.texto}
          </div>
        ))}
      </div>
      <div className="entrada">
        <span className="prompt">{PROMPT}</span>
        <span className="espelho" aria-hidden="true">
          {valor}
        </span>
        <span className="cursor" aria-hidden="true" />
        <input
          ref={campo}
          className="campo"
          value={valor}
          onChange={(e) => setValor(e.target.value)}
          onKeyDown={aoPressionar}
          autoFocus
          autoCapitalize="off"
          autoComplete="off"
          autoCorrect="off"
          spellCheck={false}
          aria-label="Comando"
        />
      </div>
    </div>
  )
}
