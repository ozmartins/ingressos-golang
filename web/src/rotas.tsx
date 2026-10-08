import { Route, Routes } from 'react-router-dom'
import { App } from './App'
import { RotaProtegida } from './auth/RotaProtegida'
import { Entrar } from './paginas/Entrar'
import { Sessoes } from './paginas/Sessoes'
import { Sessao } from './paginas/Sessao'
import { Filmes } from './paginas/Filmes'
import { FilmeDetalhe } from './paginas/FilmeDetalhe'
import { Cinemas } from './paginas/Cinemas'
import { Pagamento } from './paginas/Pagamento'
import { MeusIngressos } from './paginas/MeusIngressos'
import { Portaria } from './paginas/Portaria'
import { Operacao } from './paginas/admin/Operacao'
import { FilmesAdmin } from './paginas/admin/FilmesAdmin'
import { CinemasAdmin } from './paginas/admin/CinemasAdmin'
import { SalasAdmin } from './paginas/admin/SalasAdmin'
import { SessoesAdmin } from './paginas/admin/SessoesAdmin'
import { NaoEncontrada } from './paginas/NaoEncontrada'

export function Rotas() {
  return (
    <Routes>
      <Route element={<App />}>
        <Route index element={<Sessoes />} />
        <Route path="entrar" element={<Entrar />} />
        <Route path="filmes" element={<Filmes />} />
        <Route path="filmes/:id" element={<FilmeDetalhe />} />
        <Route path="cinemas" element={<Cinemas />} />
        {/* A portaria autentica por chave de dispositivo, não por JWT: fica
            fora das rotas protegidas de propósito. */}
        <Route path="portaria" element={<Portaria />} />

        <Route element={<RotaProtegida />}>
          <Route path="sessoes/:id" element={<Sessao />} />
          <Route path="pagamento/:reservaId" element={<Pagamento />} />
          <Route path="meus-ingressos" element={<MeusIngressos />} />
          <Route path="admin" element={<Operacao />}>
            <Route index element={<FilmesAdmin />} />
            <Route path="cinemas" element={<CinemasAdmin />} />
            <Route path="salas" element={<SalasAdmin />} />
            <Route path="sessoes" element={<SessoesAdmin />} />
          </Route>
        </Route>

        <Route path="*" element={<NaoEncontrada />} />
      </Route>
    </Routes>
  )
}
