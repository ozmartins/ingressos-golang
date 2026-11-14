# Contrato: saídas de ajuda

## `help` (e `help <qualquer texto>`)

```text
Comandos disponíveis:
  help           lista os comandos disponíveis
  clear          limpa a tela

Comandos da API do catálogo:
  filmes
  cinemas
  salas
  sessoes
  health
Digite "<controller> help" para ver as opções (ex.: filmes help).
Sessão
  login
      abre a página de login do Keycloak (usuário e senha)
  login <client_id> <client_secret>
      autentica por client credentials, sem sair do terminal
  logout
      encerra a sessão
```
Máximo de 20 linhas (SC-001). Cabeçalho e itens de "Comandos disponíveis" e "Sessão" como hoje.

## `<controller> help` (filmes | cinemas | salas | sessoes; qualquer caixa/espaçamento)

```text
Filmes
  filmes listar [status=EM_CARTAZ|BREVE|FORA_DE_CARTAZ] [page=N] [page_size=N]
      lista os filmes, com filtros e paginação
  filmes criar
      cadastra um filme, perguntando os campos (exige login)
  filmes obter <id>
      mostra um filme pelo id
  filmes atualizar <id>
      atualiza um filme, perguntando os campos (exige login)
  filmes remover <id>
      remove um filme (exige login)
criar e atualizar perguntam os campos um a um; digite "cancelar" (ou Esc) para abortar.
```
Só as ações do controller pedido. Nenhuma requisição.

## `health help`

```text
Operacional
  health
      mostra se o catálogo está apto a receber tráfego
```

## `<controller>` sem ação / ação desconhecida

```text
ação desconhecida: xpto            (apenas quando há ação)
ações de filmes: listar, criar, obter, atualizar, remover
use: filmes help para ver os detalhes.
```
