# Data Model: URL do logo do cinema

## Cinema (`catalogo.cinemas`)

| Campo | Tipo (banco) | Obrigatório | Regras |
|---|---|---|---|
| id, nome, cidade, estado, endereco, ativo | (inalterados) | — | (inalteradas) |
| **logo_url** | VARCHAR(2048), NULL | não | aparado; vazio = sem logo (gravado como NULL); esquema http/https; host presente; sem espaços; ≤ 2048 caracteres |

**Domínio**: `Cinema.LogoURL string` / `DadosCinema.LogoURL string`; `""` representa ausência.

**Transições**: criar (opcional) → editar (substitui; omitir remove) → desativar (logo preservado).

**Migração**: linhas existentes recebem `NULL`. Down remove a coluna (perde apenas os logos).
