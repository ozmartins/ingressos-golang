import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import './index.css'
import App from './App.tsx'
import { iniciarTelemetria } from './infra/telemetria'

// Falha de telemetria nunca derruba a aplicação.
try {
  if (import.meta.env.VITE_TELEMETRIA !== 'off') iniciarTelemetria()
} catch (erro) {
  console.warn('telemetria desativada:', erro)
}

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <App />
  </StrictMode>,
)
