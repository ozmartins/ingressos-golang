import type { jsPDF } from 'jspdf'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { montarIngressos, type IngressoPdf } from '../dominio/ingressoPdf'
import type { ReservaPagamento } from '../dominio/reservaPagamento'
import { baixarIngressoPdf, gerarDocumento } from './gerarPdf'

const base: ReservaPagamento = {
  usuarioId: null,
  cinemaId: 'c1',
  filmeId: 'f1',
  sessaoId: 's1',
  filmeTitulo: 'Ação e Emoção: Coração Valente',
  classificacao: '12',
  posterUrl: null,
  cinemaNome: 'Cine Centro',
  sala: 3,
  inicio: new Date(2026, 10, 15, 16, 0).toISOString(),
  rotulos: ['A1'],
  precoCentavos: 3000,
  taxaCentavos: 250,
  totalCentavos: 3250,
  expiraEm: '2026-11-15T21:10:00.000Z',
}

const com = (n: number, extra: Partial<ReservaPagamento> = {}): IngressoPdf =>
  montarIngressos({ ...base, rotulos: Array.from({ length: n }, (_, i) => `A${i + 1}`), ...extra })

/** Bytes do PDF lidos como latin1 (texto dos objetos é legível; PDF sem compressão). */
const texto = (doc: jsPDF) => new TextDecoder('latin1').decode(doc.output('arraybuffer'))

afterEach(() => {
  vi.unstubAllGlobals()
  vi.restoreAllMocks()
})

describe('gerarDocumento', () => {
  it('produz um PDF com N+1 páginas (ingressos + resumo)', async () => {
    const doc = await gerarDocumento(com(3))
    expect(doc.getNumberOfPages()).toBe(4)
    expect(texto(doc).startsWith('%PDF-')).toBe(true)
  })

  it('repete o aviso de simulação, o código e o nome do cinema nas páginas', async () => {
    const ingresso = com(2)
    const bruto = texto(await gerarDocumento(ingresso))
    const ocorrencias = (trecho: string) => bruto.split(trecho).length - 1
    expect(ocorrencias('Pagamento simulado')).toBe(3)
    expect(ocorrencias(ingresso.codigoPedido)).toBeGreaterThanOrEqual(3)
    expect(ocorrencias('Cine Centro')).toBe(2)
  })

  it('não inclui nenhum dado de cartão', async () => {
    const bruto = texto(await gerarDocumento(com(1))).toLowerCase()
    for (const proibido of ['cvv', 'número do cartão', 'nome impresso']) expect(bruto).not.toContain(proibido)
  })

  it('10 assentos: gera em tempo folgado e sem chamar a rede', async () => {
    const fetchSimulado = vi.fn()
    vi.stubGlobal('fetch', fetchSimulado)
    const inicio = performance.now()
    const doc = await gerarDocumento(com(10))
    expect(doc.getNumberOfPages()).toBe(11)
    expect(performance.now() - inicio).toBeLessThan(3000)
    expect(fetchSimulado).not.toHaveBeenCalled()
  })

  it('textos muito longos e acentuados são gerados sem erro', async () => {
    const longo = 'Ação ç ã é ª '.repeat(30)
    const doc = await gerarDocumento(com(2, { filmeTitulo: longo, cinemaNome: longo }))
    expect(doc.getNumberOfPages()).toBe(3)
    expect(texto(doc).startsWith('%PDF-')).toBe(true)
  })

  it('sem classificação, gera igualmente', async () => {
    const { classificacao: _c, ...semClassificacao } = base
    const doc = await gerarDocumento(montarIngressos(semClassificacao))
    expect(doc.getNumberOfPages()).toBe(2)
  })
})

describe('baixarIngressoPdf', () => {
  it('dispara o download com o nome do arquivo do ingresso e libera a URL', async () => {
    const criar = vi.fn(() => 'blob:ingresso')
    const revogar = vi.fn()
    vi.stubGlobal('URL', Object.assign(URL, { createObjectURL: criar, revokeObjectURL: revogar }))
    const baixados: { nome: string; href: string }[] = []
    vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(function (this: HTMLAnchorElement) {
      baixados.push({ nome: this.download, href: this.getAttribute('href') ?? '' })
    })
    vi.useFakeTimers()
    try {
      const ingresso = com(1)
      await baixarIngressoPdf(ingresso)
      expect(baixados).toEqual([{ nome: ingresso.nomeArquivo, href: 'blob:ingresso' }])
      expect(document.querySelector('a[download]')).toBeNull()
      vi.advanceTimersByTime(1000)
      expect(revogar).toHaveBeenCalledWith('blob:ingresso')
    } finally {
      vi.useRealTimers()
    }
  })
})
