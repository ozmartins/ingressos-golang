/// <reference types="vite/client" />

interface ImportMetaEnv {
  /** Base do catálogo; por padrão `/catalogo` (proxy do Vite). */
  readonly VITE_CATALOGO_URL?: string
  /** Keycloak alcançável pelo navegador; por padrão `http://keycloak:8081`. */
  readonly VITE_KEYCLOAK_URL?: string
  /** Endpoint OTLP/HTTP dos spans; por padrão `/otlp/v1/traces` (proxy do Vite até o Jaeger). */
  readonly VITE_OTLP_URL?: string
  /** `off` desliga o tracing do navegador. */
  readonly VITE_TELEMETRIA?: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}
