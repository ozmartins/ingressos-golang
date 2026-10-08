import { Link } from 'react-router-dom'
import { EstadoVazio } from '../componentes/ui'

export function NaoEncontrada() {
  return (
    <div className="space-y-4">
      <EstadoVazio titulo="Página não encontrada" descricao="O endereço não corresponde a nenhuma tela." />
      <p className="text-center">
        <Link to="/" className="text-sm text-destaque hover:underline">
          Voltar para a grade de sessões
        </Link>
      </p>
    </div>
  )
}
