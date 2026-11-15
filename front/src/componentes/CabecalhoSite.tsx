import type { Sessao } from '../dominio/keycloak'
import { BarraConta } from './BarraConta'
import { LinkInicio } from './LinkInicio'

interface Props {
  sessao: Sessao | null
  onSair: () => void
}

/** Cabeçalho único do site: atalho para o início à esquerda, conta à direita. */
export function CabecalhoSite({ sessao, onSair }: Props) {
  return (
    <header className="cabecalho-site">
      <LinkInicio />
      <BarraConta sessao={sessao} onSair={onSair} />
    </header>
  )
}
