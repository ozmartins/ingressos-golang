import { useCallback, useEffect, useRef, useState } from 'react'

export type EstadoCarga = 'carregando' | 'pronto' | 'erro'

export interface Lista<T> {
  estado: EstadoCarga
  itens: T[]
  erro: string
  recarregar: () => void
}

/** Carrega uma lista ao montar e permite repetir a carga após uma falha. */
export function useLista<T>(carregar: () => Promise<T[]>): Lista<T> {
  const [estado, setEstado] = useState<EstadoCarga>('carregando')
  const [itens, setItens] = useState<T[]>([])
  const [erro, setErro] = useState('')
  const [tentativa, setTentativa] = useState(0)
  const carregarRef = useRef(carregar)

  useEffect(() => {
    carregarRef.current = carregar
  })

  useEffect(() => {
    let ativo = true
    carregarRef
      .current()
      .then((resultado) => {
        if (!ativo) return
        setItens(resultado)
        setEstado('pronto')
      })
      .catch((falha: unknown) => {
        if (!ativo) return
        setErro(falha instanceof Error ? falha.message : 'Não foi possível carregar a lista.')
        setEstado('erro')
      })
    return () => {
      ativo = false
    }
  }, [tentativa])

  const recarregar = useCallback(() => {
    setEstado('carregando')
    setTentativa((n) => n + 1)
  }, [])

  return { estado, itens, erro, recarregar }
}
