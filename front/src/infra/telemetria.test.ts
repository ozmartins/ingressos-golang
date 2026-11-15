import { InMemorySpanExporter } from '@opentelemetry/sdk-trace-base'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { iniciarTelemetria } from './telemetria'

describe('iniciarTelemetria', () => {
  const original = globalThis.fetch

  afterEach(() => {
    globalThis.fetch = original
  })

  it('envia o traceparent W3C nas chamadas de mesma origem e registra um span de cliente', async () => {
    const recebido = vi.fn(async (_url: RequestInfo | URL, _init?: RequestInit) => new Response('{}', { status: 200 }))
    globalThis.fetch = recebido as unknown as typeof fetch

    const exportador = new InMemorySpanExporter()
    const provedor = iniciarTelemetria({ exportador })

    await fetch(`${window.location.origin}/catalogo/api/v1/filmes`)
    // O span fecha quando o corpo da resposta termina de ser lido, depois do fetch.
    await vi.waitFor(() => expect(exportador.getFinishedSpans()).toHaveLength(1))
    await provedor.forceFlush()

    const init = recebido.mock.calls[0]?.[1]
    const cabecalhos = new Headers(init?.headers)
    expect(cabecalhos.get('traceparent')).toMatch(/^00-[0-9a-f]{32}-[0-9a-f]{16}-0[01]$/)

    const spans = exportador.getFinishedSpans()
    expect(spans).toHaveLength(1)
    expect(cabecalhos.get('traceparent')).toContain(spans[0].spanContext().traceId)
  })

  it('não rastreia o envio dos próprios spans', async () => {
    globalThis.fetch = vi.fn(async () => new Response('', { status: 200 })) as unknown as typeof fetch
    const exportador = new InMemorySpanExporter()
    const provedor = iniciarTelemetria({ exportador })

    await fetch(`${window.location.origin}/otlp/v1/traces`, { method: 'POST' })
    await provedor.forceFlush()

    expect(exportador.getFinishedSpans()).toHaveLength(0)
  })
})
