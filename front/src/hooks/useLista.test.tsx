import { act, renderHook, waitFor } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { useLista } from './useLista'

describe('useLista', () => {
  it('vai de carregando a pronto', async () => {
    const carregar = vi.fn().mockResolvedValue(['a', 'b'])
    const { result } = renderHook(() => useLista(carregar))

    expect(result.current.estado).toBe('carregando')
    await waitFor(() => expect(result.current.estado).toBe('pronto'))
    expect(result.current.itens).toEqual(['a', 'b'])
  })

  it('reporta erro e permite tentar de novo', async () => {
    const carregar = vi.fn().mockRejectedValueOnce(new Error('falhou')).mockResolvedValueOnce(['ok'])
    const { result } = renderHook(() => useLista(carregar))

    await waitFor(() => expect(result.current.estado).toBe('erro'))
    expect(result.current.erro).toBe('falhou')

    act(() => result.current.recarregar())
    await waitFor(() => expect(result.current.estado).toBe('pronto'))
    expect(result.current.itens).toEqual(['ok'])
    expect(carregar).toHaveBeenCalledTimes(2)
  })
})
