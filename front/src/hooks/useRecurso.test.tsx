import { act, renderHook, waitFor } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { useRecurso } from './useRecurso'

describe('useRecurso', () => {
  it('vai de carregando para pronto', async () => {
    const { result } = renderHook(() => useRecurso(() => Promise.resolve(42), 'a'))
    expect(result.current.estado).toBe('carregando')
    await waitFor(() => expect(result.current.estado).toBe('pronto'))
    expect(result.current.dado).toBe(42)
  })

  it('guarda o erro lançado e permite recarregar', async () => {
    const falha = new Error('x')
    const carregar = vi.fn().mockRejectedValueOnce(falha).mockResolvedValueOnce('ok')
    const { result } = renderHook(() => useRecurso(carregar, 'a'))
    await waitFor(() => expect(result.current.estado).toBe('erro'))
    expect(result.current.erro).toBe(falha)
    act(() => result.current.recarregar())
    await waitFor(() => expect(result.current.estado).toBe('pronto'))
    expect(result.current.dado).toBe('ok')
  })

  it('recarrega quando a chave muda', async () => {
    const carregar = vi.fn((chave: string) => Promise.resolve(chave))
    const { result, rerender } = renderHook(({ chave }) => useRecurso(() => carregar(chave), chave), {
      initialProps: { chave: 'a' },
    })
    await waitFor(() => expect(result.current.dado).toBe('a'))
    rerender({ chave: 'b' })
    await waitFor(() => expect(result.current.dado).toBe('b'))
    expect(carregar).toHaveBeenCalledTimes(2)
  })
})
