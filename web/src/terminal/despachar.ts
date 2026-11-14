import { destacar, ehErro, lerArgumentos } from './api/argumentos'
import { FalhaRequisicao, requisitar, type Base } from './api/cliente'
import {
  formatarAcoes,
  formatarAjuda,
  formatarAjudaController,
  formatarErro,
  formatarFalha,
  formatarResposta,
  formatarUso,
} from './api/formatar'
import { iniciarLogin, type ConfigOidc } from './api/oidc'
import { SEM_SESSAO, login, logout, type Sessao } from './api/sessao'
import { obterFormulario } from './api/formularios'
import { buscarDefinicao, ehControllerCrud, TABELA, type Definicao } from './api/tabela'
import { iniciar, responder, type Dialogo } from './dialogo'
import { executar, linhasAjudaLocal, type Resultado } from './comandos'
import type { Linha } from './linhas'

export interface Contexto {
  fetchFn: typeof fetch
  base: Base
  sessao: Sessao
  /** Login pela página do Keycloak; sem isso, só o login por client_id/secret. */
  oidc?: ConfigOidc
}

const erro = (texto: string): Linha => ({ tipo: 'erro', texto })

function resultado(linhas: Linha[], sessao?: Sessao): Resultado {
  return sessao === undefined ? { linhas, limpar: false } : { linhas, limpar: false, sessao }
}

async function executarLoginKeycloak(oidc: ConfigOidc): Promise<Resultado> {
  const url = await iniciarLogin(oidc)
  return {
    linhas: [{ tipo: 'sistema', texto: 'Redirecionando para a página de login do Keycloak...' }],
    limpar: false,
    redirecionar: url,
  }
}

async function executarLogin(resto: string, ctx: Contexto): Promise<Resultado> {
  if (resto.trim() === '' && ctx.oidc !== undefined) return executarLoginKeycloak(ctx.oidc)
  const [clientId, segredo] = destacar(resto)
  if (clientId === '' || segredo === '') {
    return resultado([erro('uso: login | login <client_id> <client_secret>')])
  }
  const r = await login(ctx.fetchFn, ctx.base, clientId, segredo)
  return resultado(r.linhas, r.sessao)
}

async function enviarDefinicao(
  def: Definicao,
  id: string | undefined,
  query: Record<string, string>,
  corpo: unknown,
  ctx: Contexto,
): Promise<Resultado> {
  const caminho = def.caminho.replace('{id}', encodeURIComponent(id ?? ''))
  try {
    const resposta = await requisitar(ctx.fetchFn, ctx.base, {
      metodo: def.metodo,
      caminho,
      raiz: def.raiz,
      query,
      corpo,
      token: ctx.sessao.token,
    })
    if (resposta.status === 401) {
      return resultado(
        [
          ...formatarErro(resposta),
          erro('  sessão ausente ou expirada: use login para autenticar novamente.'),
        ],
        SEM_SESSAO,
      )
    }
    if (resposta.status >= 400) return resultado(formatarErro(resposta))
    return resultado(formatarResposta(def, resposta))
  } catch (e) {
    if (e instanceof FalhaRequisicao) return resultado(formatarFalha(e))
    throw e
  }
}

async function executarDefinicao(def: Definicao, resto: string, ctx: Contexto): Promise<Resultado> {
  const args = lerArgumentos(def, resto)
  if (ehErro(args)) return resultado([erro(args.erro), ...formatarUso(def)])

  if (def.protegido && ctx.sessao.token === null) {
    return resultado([
      erro('este comando exige autenticação.'),
      erro('use: login'),
    ])
  }

  if (def.usaCorpo) {
    const campos = obterFormulario(def)
    if (campos === undefined) return resultado([erro(`comando ainda não disponível: ${def.uso}`)])
    const passo = iniciar(def, campos, args.id)
    return { linhas: passo.linhas, limpar: false, dialogo: passo.dialogo }
  }

  return enviarDefinicao(def, args.id, args.query, undefined, ctx)
}

/** Recebe uma linha digitada enquanto há um diálogo de preenchimento ativo. */
export async function continuarDialogo(
  dialogo: Dialogo,
  entrada: string,
  ctx: Contexto,
): Promise<Resultado> {
  const passo = responder(dialogo, entrada)
  if (passo.enviar === undefined) {
    return { linhas: passo.linhas, limpar: false, dialogo: passo.dialogo }
  }
  const r = await enviarDefinicao(dialogo.definicao, dialogo.id, {}, passo.enviar.corpo, ctx)
  return { ...r, linhas: [...passo.linhas, ...r.linhas], dialogo: null }
}

export async function despachar(entrada: string, ctx: Contexto): Promise<Resultado> {
  const texto = entrada.trim()
  if (texto === '') return executar(entrada)

  const [nome, resto] = destacar(texto)
  const chave = nome.toLowerCase()

  if (chave === 'help') {
    return resultado([...linhasAjudaLocal(), ...formatarAjuda()])
  }
  if (chave === 'login') return executarLogin(resto, ctx)
  if (chave === 'logout') {
    const r = logout()
    return resultado(r.linhas, r.sessao)
  }
  if (chave === 'health') {
    if (resto.trim().toLowerCase() === 'help') return resultado(formatarAjudaController('operacional'))
    const def = TABELA.find((d) => d.acao === 'health')
    if (def) return executarDefinicao(def, resto, ctx)
  }
  if (ehControllerCrud(chave)) {
    const [acao, argumentos] = destacar(resto)
    if (acao.toLowerCase() === 'help') return resultado(formatarAjudaController(chave))
    const def = acao === '' ? undefined : buscarDefinicao(chave, acao)
    if (!def) {
      const prefixo = acao === '' ? [] : [erro(`ação desconhecida: ${acao}`)]
      return resultado([...prefixo, ...formatarAcoes(chave)])
    }
    return executarDefinicao(def, argumentos, ctx)
  }
  return executar(texto)
}
