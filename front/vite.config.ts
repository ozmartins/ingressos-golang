import react from '@vitejs/plugin-react'
import type { ProxyOptions } from 'vite'
import { defineConfig } from 'vitest/config'

// O catálogo não envia cabeçalhos CORS; o navegador acessa o serviço pelo
// proxy do Vite, na mesma origem da página.
const proxyServicos: Record<string, ProxyOptions> = {
  '/catalogo': {
    target: 'http://localhost:8082',
    changeOrigin: true,
    rewrite: (caminho) => caminho.replace(/^\/catalogo/, ''),
  },
}

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  server: { proxy: proxyServicos },
  preview: { proxy: proxyServicos },
  test: {
    environment: 'jsdom',
    setupFiles: ['./src/testes/setup.ts'],
  },
})
