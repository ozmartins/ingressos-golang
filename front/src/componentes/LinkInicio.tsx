import { interceptarClique, navegar } from '../hooks/useRota'

const CAMINHO_INICIO = '/'

function irParaInicio() {
  // Já na raiz: não recarrega nem cria entrada repetida no histórico.
  if (window.location.pathname === CAMINHO_INICIO) return
  navegar(CAMINHO_INICIO)
}

/** Ícone de casa que leva à página inicial; presente em todas as páginas pelo `CabecalhoSite`. */
export function LinkInicio() {
  return (
    <a
      className="link-inicio"
      href={CAMINHO_INICIO}
      aria-label="Página inicial"
      title="Página inicial"
      onClick={(e) => interceptarClique(e, irParaInicio)}
    >
      <svg viewBox="0 0 24 24" width="24" height="24" aria-hidden="true" focusable="false" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
        <path d="M3 11.5 12 4l9 7.5" />
        <path d="M5 10v10h5v-6h4v6h5V10" />
      </svg>
    </a>
  )
}
