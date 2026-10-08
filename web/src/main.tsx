import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { BrowserRouter } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { ProvedorAuth } from './auth/ContextoAuth'
import { Rotas } from './rotas'
import './estilos.css'

const clienteDeConsultas = new QueryClient({
  defaultOptions: {
    queries: {
      // Boa parte do que se lê aqui muda por mensageria, fora do controle da
      // tela. Repetir sozinho esconderia o estado real das poltronas.
      retry: false,
      refetchOnWindowFocus: false,
      staleTime: 15_000,
    },
  },
})

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <QueryClientProvider client={clienteDeConsultas}>
      <BrowserRouter>
        <ProvedorAuth>
          <Rotas />
        </ProvedorAuth>
      </BrowserRouter>
    </QueryClientProvider>
  </StrictMode>,
)
