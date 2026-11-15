import react from '@vitejs/plugin-react'
import type { ProxyOptions } from 'vite'
import { defineConfig } from 'vitest/config'

// O catálogo e o estoque não enviam cabeçalhos CORS; o navegador acessa o serviço pelo
// proxy do Vite, na mesma origem da página.
const proxyServicos: Record<string, ProxyOptions> = {
  '/catalogo': {
    target: 'http://localhost:8082',
    changeOrigin: true,
    rewrite: (caminho) => caminho.replace(/^\/catalogo/, ''),
  },
  '/estoque': {
    target: 'http://localhost:8085',
    changeOrigin: true,
    rewrite: (caminho) => caminho.replace(/^\/estoque/, ''),
  },
  // Spans do navegador (OTLP/HTTP) até o Jaeger; o Jaeger também não envia CORS.
  '/otlp': {
    target: 'http://localhost:4318',
    changeOrigin: true,
    rewrite: (caminho) => caminho.replace(/^\/otlp/, ''),
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
