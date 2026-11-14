import { useEffect, useState, type MouseEvent } from 'react'
import { lerRota, type Rota } from '../dominio/rotas'

const EVENTO = 'navegacao-interna'

/** Navega sem recarregar; marca a entrada do histórico como interna ao app. */
export function navegar(caminho: string): void {
  window.history.pushState({ interno: true }, '', caminho)
  window.dispatchEvent(new Event(EVENTO))
}

/** A entrada atual do histórico foi criada por `navegar` (há para onde voltar no app). */
export function veioDoApp(): boolean {
  const estado: unknown = window.history.state
  return typeof estado === 'object' && estado !== null && (estado as { interno?: boolean }).interno === true
}

/** Clique simples vira navegação interna; Ctrl/Cmd/Shift/botão do meio seguem o padrão do navegador. */
export function interceptarClique(evento: MouseEvent<HTMLElement>, acao: () => void): void {
  if (evento.defaultPrevented || evento.button !== 0) return
  if (evento.metaKey || evento.ctrlKey || evento.shiftKey || evento.altKey) return
  evento.preventDefault()
  acao()
}

/** Rota atual, derivada de `location.pathname`. */
export function useRota(): [Rota, (caminho: string) => void] {
  const [rota, setRota] = useState<Rota>(() => lerRota(window.location.pathname))

  useEffect(() => {
    const atualizar = () => setRota(lerRota(window.location.pathname))
    window.addEventListener('popstate', atualizar)
    window.addEventListener(EVENTO, atualizar)
    return () => {
      window.removeEventListener('popstate', atualizar)
      window.removeEventListener(EVENTO, atualizar)
    }
  }, [])

  return [rota, navegar]
}
