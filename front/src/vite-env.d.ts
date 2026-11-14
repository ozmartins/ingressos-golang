/// <reference types="vite/client" />

interface ImportMetaEnv {
  /** Base do catálogo; por padrão `/catalogo` (proxy do Vite). */
  readonly VITE_CATALOGO_URL?: string
  /** Keycloak alcançável pelo navegador; por padrão `http://keycloak:8081`. */
  readonly VITE_KEYCLOAK_URL?: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}
