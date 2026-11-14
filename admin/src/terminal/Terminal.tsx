import { useEffect, useRef, useState, type KeyboardEvent } from 'react'
import { concluirLogin, temRetorno, type ConfigOidc } from './api/oidc'
import { SEM_SESSAO, mascarar, type Sessao } from './api/sessao'
import type { Base } from './api/cliente'
import { cancelar, registrarNoHistorico, type Dialogo } from './dialogo'
import { continuarDialogo, despachar } from './despachar'
import { adicionar, anterior, criar, proximo } from './historico'
import { adicionarLinhas, type Linha } from './linhas'

const PROMPT = '>'
const BASE: Base = { catalogo: '/catalogo', identidade: '/identidade' }

const URL_KEYCLOAK: string =
  import.meta.env.VITE_KEYCLOAK_URL ?? 'http://keycloak:8081'

function configOidc(): ConfigOidc {
  return {
    urlKeycloak: URL_KEYCLOAK,
    redirectUri: `${window.location.origin}/`,
    armazem: window.sessionStorage,
  }
}

const BOAS_VINDAS: Linha[] = [
  { tipo: 'sistema', texto: 'Digite "help" para ver os comandos disponiveis.' },
  { tipo: 'sistema', texto: '' },
]

export default function Terminal() {
  const [linhas, setLinhas] = useState<Linha[]>(BOAS_VINDAS)
  const [valor, setValor] = useState('')
  const [historico, setHistorico] = useState(criar)
  const [sessao, setSessao] = useState<Sessao>(SEM_SESSAO)
  const [ocupado, setOcupado] = useState(false)
  const [dialogo, setDialogo] = useState<Dialogo | null>(null)
  const campo = useRef<HTMLInputElement>(null)
  const saida = useRef<HTMLDivElement>(null)

  const retornoTratado = useRef(false)

  // Volta da página de login do Keycloak: troca o code pelo token.
  useEffect(() => {
    if (retornoTratado.current || !temRetorno(window.location.search)) return
    retornoTratado.current = true
    const consulta = window.location.search
    window.history.replaceState(null, '', window.location.pathname)
    setOcupado(true)
    void concluirLogin(fetch.bind(globalThis), BASE, configOidc(), consulta)
      .then((r) => {
        if (r.sessao) setSessao(r.sessao)
        setLinhas((atuais) => adicionarLinhas(atuais, r.linhas))
      })
      .finally(() => setOcupado(false))
  }, [])

  useEffect(() => {
    const el = saida.current
    if (el) el.scrollTop = el.scrollHeight
  }, [linhas])

  async function enviar() {
    if (ocupado) return
    const entrada = valor
    const eco: Linha = { tipo: 'entrada', texto: `${PROMPT} ${mascarar(entrada)}` }
    setValor('')
    if (registrarNoHistorico(dialogo)) setHistorico((h) => adicionar(h, mascarar(entrada)))
    setLinhas((atuais) => adicionarLinhas(atuais, [eco]))
    if (dialogo === null && entrada.trim() === '') return

    setOcupado(true)
    try {
      const contexto = {
        fetchFn: fetch.bind(globalThis),
        base: BASE,
        sessao,
        oidc: configOidc(),
      }
      const resultado =
        dialogo === null
          ? await despachar(entrada, contexto)
          : await continuarDialogo(dialogo, entrada, contexto)
      if (resultado.dialogo !== undefined) setDialogo(resultado.dialogo)
      if (resultado.redirecionar) {
        window.location.assign(resultado.redirecionar)
        return
      }
      if (resultado.fechar) {
        setLinhas((atuais) => adicionarLinhas(atuais, resultado.linhas))
        window.close()
        // O navegador só deixa fechar abas abertas por script; nas demais, avisa.
        setLinhas((atuais) =>
          adicionarLinhas(atuais, [
            { tipo: 'erro', texto: 'o navegador bloqueou o fechamento: feche a aba manualmente (Ctrl+W).' },
          ]),
        )
        return
      }
      if (resultado.sessao) setSessao(resultado.sessao)
      setLinhas((atuais) =>
        resultado.limpar ? [] : adicionarLinhas(atuais, resultado.linhas),
      )
    } finally {
      setOcupado(false)
    }
  }

  function aoPressionar(e: KeyboardEvent<HTMLInputElement>) {
    if (ocupado) {
      e.preventDefault()
      return
    }
    if (e.key === 'Escape' && dialogo !== null) {
      e.preventDefault()
      setDialogo(null)
      setLinhas((atuais) => adicionarLinhas(atuais, cancelar().linhas))
    } else if (e.key === 'Enter') {
      e.preventDefault()
      void enviar()
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
        <span className="prompt">{ocupado ? '…' : dialogo === null ? PROMPT : '?'}</span>
        <span className="espelho" aria-hidden="true">
          {valor}
        </span>
        <span className="cursor" aria-hidden="true" />
        <input
          ref={campo}
          className="campo"
          value={valor}
          onChange={(e) => setValor(e.target.value)}
          readOnly={ocupado}
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
