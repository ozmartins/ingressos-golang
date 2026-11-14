import { act, renderHook } from '@testing-library/react'
import { beforeEach, describe, expect, it } from 'vitest'
import { useAbaNaUrl } from './useAbaNaUrl'

beforeEach(() => {
  window.history.replaceState(null, '', '/')
})

describe('useAbaNaUrl', () => {
  it('usa filmes quando não há parâmetro', () => {
    const { result } = renderHook(() => useAbaNaUrl())
    expect(result.current[0]).toBe('filmes')
  })

  it('lê a aba da URL', () => {
    window.history.replaceState(null, '', '/?aba=cinemas')
    const { result } = renderHook(() => useAbaNaUrl())
    expect(result.current[0]).toBe('cinemas')
  })

  it('trata valor inválido como filmes', () => {
    window.history.replaceState(null, '', '/?aba=outra')
    const { result } = renderHook(() => useAbaNaUrl())
    expect(result.current[0]).toBe('filmes')
  })

  it('escreve a aba na URL ao trocar', () => {
    const { result } = renderHook(() => useAbaNaUrl())
    act(() => result.current[1]('cinemas'))
    expect(result.current[0]).toBe('cinemas')
    expect(window.location.search).toBe('?aba=cinemas')
  })

  it('reage ao popstate', () => {
    const { result } = renderHook(() => useAbaNaUrl())
    act(() => {
      window.history.replaceState(null, '', '/?aba=cinemas')
      window.dispatchEvent(new PopStateEvent('popstate'))
    })
    expect(result.current[0]).toBe('cinemas')
  })
})
