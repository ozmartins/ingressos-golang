# Contrato: campos pedidos por comando

Fonte: schemas do `catalogo/internal/adapter/http/openapi/openapi.yaml`. O corpo enviado contém só as chaves respondidas; opcionais em branco são omitidos (o contrato usa `additionalProperties: false`).

## `filmes criar` / `filmes atualizar <id>` — `FilmeEntrada`

| Chave | Tipo | Obrig. | Restrições / padrão |
|-------|------|--------|---------------------|
| `titulo` | texto | sim | 1–255 |
| `sinopse` | texto | não | — |
| `duracao_minutos` | inteiro | sim | ≥ 1 |
| `classificacao_etaria` | texto | sim | 1–50 |
| `genero` | texto | sim | 1–100 |
| `imagem_url` | texto (URI) | não | ≤ 500 |
| `status` | opção | não | `EM_CARTAZ` (padrão) \| `BREVE` \| `FORA_DE_CARTAZ` |

## `cinemas criar` / `cinemas atualizar <id>` — `CinemaEntrada`

| Chave | Tipo | Obrig. | Restrições / padrão |
|-------|------|--------|---------------------|
| `nome` | texto | sim | 1–255 |
| `cidade` | texto | sim | 1–100 |
| `estado` | texto | sim | exatamente 2 letras (enviado em maiúsculas) |
| `endereco` | texto | sim | ≥ 1 |
| `ativo` | lógico | não | `true` (padrão) |

## `salas criar` — `SalaEntrada`

| Chave | Tipo | Obrig. | Restrições / padrão |
|-------|------|--------|---------------------|
| `cinema_id` | uuid | sim | cinema existente |
| `numero` | inteiro | sim | ≥ 1 |
| `tipo_tela` | opção | sim | `2D` \| `3D` \| `IMAX` \| `VIP` |
| `fileiras` | lista | sim | ≥ 1 item; cada item: `fileira` (`^[A-Z]{1,5}$`), `assentos` (≥ 1), `tipo` (`NORMAL` padrão \| `PCD` \| `NAMORADEIRA`) |
| `ativo` | lógico | não | `true` (padrão) |

## `salas atualizar <id>` — `SalaAtualizacao`

Pede apenas `numero` (sim), `tipo_tela` (sim) e `ativo` (não). `cinema_id` e `fileiras` não são pedidos (opcionais e imutáveis no contrato).

## `sessoes criar` — `SessaoEntrada`

| Chave | Tipo | Obrig. | Restrições / padrão |
|-------|------|--------|---------------------|
| `filme_id` | uuid | sim | — |
| `sala_id` | uuid | sim | — |
| `data_hora_inicio` | data/hora | sim | informada como `AAAA-MM-DD HH:MM` (ou ISO-8601); enviada em ISO-8601 com fuso |
| `idioma` | opção | sim | `DUBLADO` \| `LEGENDADO` |
| `preco_base` | decimal | sim | até 2 casas; **enviado como string** (ex.: `"42.00"`) |
| `status` | opção | não | `AGENDADA` (padrão) \| `EM_ANDAMENTO` \| `FINALIZADA` \| `CANCELADA` |

## `sessoes atualizar <id>` — `SessaoAtualizacao`

Pede `filme_id`, `data_hora_inicio`, `idioma`, `preco_base` (obrigatórios) e `status` (opcional). `sala_id` não é pedido (movê-la responde 409).

## Sintaxe de linha de comando

```
<controller> criar                 # sem argumentos
<controller> atualizar <id>        # somente o identificador
```
Qualquer texto adicional (ex.: um JSON) é recusado com orientação para o fluxo guiado.

## Palavras de controle do diálogo

| Entrada | Efeito |
|---------|--------|
| (Enter em branco) | omite campo opcional; rejeitado em campo obrigatório |
| `cancelar` ou Esc | abandona sem enviar |
| `s` / `n` | responde perguntas lógicas, "mais uma fileira?" e a confirmação final |
