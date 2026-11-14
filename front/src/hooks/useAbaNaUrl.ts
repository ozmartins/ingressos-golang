import { useCallback, useEffect, useState } from 'react'

export type Aba = 'filmes' | 'cinemas'

function lerAba(): Aba {
  const valor = new URLSearchParams(window.location.search).get('aba')
  return valor === 'cinemas' ? 'cinemas' : 'filmes'
}

/** Mantém a aba ativa sincronizada com `?aba=` (padrão: filmes). */
export function useAbaNaUrl(): [Aba, (aba: Aba) => void] {
  const [aba, setAbaEstado] = useState<Aba>(lerAba)

  useEffect(() => {
    const aoNavegar = () => setAbaEstado(lerAba())
    window.addEventListener('popstate', aoNavegar)
    return () => window.removeEventListener('popstate', aoNavegar)
  }, [])

  const setAba = useCallback((nova: Aba) => {
    const url = new URL(window.location.href)
    url.searchParams.set('aba', nova)
    window.history.pushState(null, '', url)
    setAbaEstado(nova)
  }, [])

  return [aba, setAba]
}
