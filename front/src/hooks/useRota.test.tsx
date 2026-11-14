import { act, renderHook } from '@testing-library/react'
import { afterEach, describe, expect, it } from 'vitest'
import { navegar, useRota, veioDoApp } from './useRota'

afterEach(() => {
  window.history.replaceState(null, '', '/')
})

describe('useRota', () => {
  it('lê a rota do pathname', () => {
    window.history.replaceState(null, '', '/cinemas/abc')
    const { result } = renderHook(() => useRota())
    expect(result.current[0]).toEqual({ tipo: 'cinema', id: 'abc' })
  })

  it('navegar faz pushState, atualiza a rota e marca a entrada como interna', () => {
    window.history.replaceState(null, '', '/')
    const { result } = renderHook(() => useRota())
    expect(veioDoApp()).toBe(false)
    act(() => result.current[1]('/cinemas/x'))
    expect(window.location.pathname).toBe('/cinemas/x')
    expect(result.current[0]).toEqual({ tipo: 'cinema', id: 'x' })
    expect(veioDoApp()).toBe(true)
  })

  it('acompanha popstate', () => {
    window.history.replaceState(null, '', '/cinemas/a')
    const { result } = renderHook(() => useRota())
    act(() => {
      window.history.replaceState(null, '', '/')
      window.dispatchEvent(new PopStateEvent('popstate'))
    })
    expect(result.current[0]).toEqual({ tipo: 'inicio' })
  })

  it('navegar fora do hook também notifica', () => {
    const { result } = renderHook(() => useRota())
    act(() => navegar('/cinemas/z'))
    expect(result.current[0]).toEqual({ tipo: 'cinema', id: 'z' })
  })
})
