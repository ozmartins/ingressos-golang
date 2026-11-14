/// <reference types="vite/client" />

interface ImportMetaEnv {
  /** Base do catálogo; por padrão `/catalogo` (proxy do Vite). */
  readonly VITE_CATALOGO_URL?: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}
