import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

// Em produção quem repassa as chamadas é o nginx (ver `nginx.conf`), e os
// caminhos abaixo existem só para `npm run dev` fora do compose. Os destinos
// são as portas do host publicadas pelo docker-compose.yml da raiz.
export default defineConfig({
  plugins: [react(), tailwindcss()],
  server: {
    port: 3000,
    proxy: {
      '/api/catalogo': {
        target: 'http://localhost:8082',
        changeOrigin: true,
        rewrite: (p) => p.replace(/^\/api\/catalogo/, '/api/v1'),
      },
      '/api/estoque': {
        target: 'http://localhost:8085',
        changeOrigin: true,
        rewrite: (p) => p.replace(/^\/api\/estoque/, '/api/v1'),
      },
      '/api/pagamento': {
        target: 'http://localhost:8083',
        changeOrigin: true,
        rewrite: (p) => p.replace(/^\/api\/pagamento/, '/api/v1'),
      },
      '/api/notificacao': {
        target: 'http://localhost:8084',
        changeOrigin: true,
        rewrite: (p) => p.replace(/^\/api\/notificacao/, '/api/v1'),
      },
      '/auth': {
        target: 'http://localhost:8081',
        changeOrigin: true,
        rewrite: (p) => p.replace(/^\/auth/, ''),
      },
    },
  },
})
