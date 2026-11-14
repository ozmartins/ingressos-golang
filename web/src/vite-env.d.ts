/// <reference types="vite/client" />

interface ImportMetaEnv {
  /** Endereço do Keycloak alcançável pelo navegador (padrão: http://keycloak:8081). */
  readonly VITE_KEYCLOAK_URL?: string
}
