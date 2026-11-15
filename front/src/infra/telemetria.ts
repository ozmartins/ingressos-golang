import { ZoneContextManager } from '@opentelemetry/context-zone'
import { OTLPTraceExporter } from '@opentelemetry/exporter-trace-otlp-http'
import { FetchInstrumentation } from '@opentelemetry/instrumentation-fetch'
import { registerInstrumentations } from '@opentelemetry/instrumentation'
import { resourceFromAttributes } from '@opentelemetry/resources'
import { BatchSpanProcessor, SimpleSpanProcessor, type SpanExporter } from '@opentelemetry/sdk-trace-base'
import { WebTracerProvider } from '@opentelemetry/sdk-trace-web'

/** Onde o navegador envia os spans: um proxy do Vite (`/otlp`) até o Jaeger, na mesma origem. */
const URL_OTLP_PADRAO = '/otlp/v1/traces'

export interface OpcoesTelemetria {
  /** Substitui o exportador OTLP (testes). Com ele, os spans saem sem lote. */
  exportador?: SpanExporter
}

/**
 * Liga o tracing do navegador. Cada `fetch` vira um span de cliente e leva o
 * cabeçalho W3C `traceparent`: o catálogo e o estoque (acessados pelo proxy, na
 * mesma origem) continuam o trace aberto aqui, e o trace passa a começar no clique
 * do usuário, não na primeira linha do backend.
 *
 * Chamadas a outras origens (o Keycloak) geram span, mas não recebem o cabeçalho:
 * mandá-lo exigiria liberá-lo no CORS desse servidor.
 */
export function iniciarTelemetria(opcoes: OpcoesTelemetria = {}): WebTracerProvider {
  const url = import.meta.env.VITE_OTLP_URL ?? URL_OTLP_PADRAO
  const processador = opcoes.exportador
    ? new SimpleSpanProcessor(opcoes.exportador)
    : new BatchSpanProcessor(new OTLPTraceExporter({ url }))

  const provedor = new WebTracerProvider({
    resource: resourceFromAttributes({ 'service.name': 'front' }),
    spanProcessors: [processador],
  })
  provedor.register({ contextManager: new ZoneContextManager() })

  registerInstrumentations({
    tracerProvider: provedor,
    instrumentations: [
      new FetchInstrumentation({
        // Não rastreia o próprio envio de spans, que viraria um laço.
        ignoreUrls: [/\/otlp\//],
      }),
    ],
  })
  return provedor
}
