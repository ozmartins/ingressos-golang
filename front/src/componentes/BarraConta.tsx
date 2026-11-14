import { urlAutenticacao, type Sessao, type Tela } from '../dominio/keycloak'

interface Props {
  sessao: Sessao | null
  onSair: () => void
}

function ir(tela: Tela) {
  const retorno = window.location.pathname + window.location.search
  void urlAutenticacao(tela, retorno).then((url) => window.location.assign(url))
}

/** Criar Conta/Login levam às telas do Keycloak; logado, Login vira Log out. */
export function BarraConta({ sessao, onSair }: Props) {
  return (
    <nav className="barra-conta" aria-label="Conta">
      {sessao === null ? (
        <>
          <button type="button" className="botao botao-secundario" onClick={() => ir('registro')}>
            Criar Conta
          </button>
          <button type="button" className="botao" onClick={() => ir('login')}>
            Login
          </button>
        </>
      ) : (
        <button type="button" className="botao" onClick={onSair}>
          Log out
        </button>
      )}
    </nav>
  )
}
