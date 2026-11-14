import { useCallback, useEffect, useRef, useState } from 'react'
import type { EstadoCarga } from './useLista'

export interface Recurso<T> {
  estado: EstadoCarga
  dado: T | undefined
  /** O erro lançado pela carga (quando `estado === 'erro'`). */
  erro: unknown
  recarregar: () => void
}

interface Resultado<T> {
  chave: string
  tentativa: number
  dado?: T
  erro?: unknown
  falhou: boolean
}

/** Carrega um único valor; refaz a carga quando `chave` muda ou em `recarregar()`. */
export function useRecurso<T>(carregar: () => Promise<T>, chave: string): Recurso<T> {
  const [resultado, setResultado] = useState<Resultado<T>>()
  const [tentativa, setTentativa] = useState(0)
  const carregarRef = useRef(carregar)

  useEffect(() => {
    carregarRef.current = carregar
  })

  useEffect(() => {
    let ativo = true
    carregarRef
      .current()
      .then((dado) => {
        if (ativo) setResultado({ chave, tentativa, dado, falhou: false })
      })
      .catch((erro: unknown) => {
        if (ativo) setResultado({ chave, tentativa, erro, falhou: true })
      })
    return () => {
      ativo = false
    }
  }, [chave, tentativa])

  const recarregar = useCallback(() => setTentativa((n) => n + 1), [])

  // Resultado de outra chave/tentativa = carga ainda em andamento.
  const atual = resultado && resultado.chave === chave && resultado.tentativa === tentativa ? resultado : undefined
  const estado: EstadoCarga = !atual ? 'carregando' : atual.falhou ? 'erro' : 'pronto'
  return { estado, dado: atual?.dado, erro: atual?.erro, recarregar }
}
