# Data Model: Página inicial com abas de Filmes e Cinemas

Somente leitura; os dados pertencem ao `catalogo`. O front mantém cópias em memória.

## Filme (da API)
| Campo | Tipo | Observação |
|---|---|---|
| id | string (uuid) | chave de lista |
| titulo | string | busca por texto |
| sinopse | string? | omitida quando ausente |
| duracao_minutos | number | exibida como "2h 05min" |
| classificacao_etaria | string | filtro (opções derivadas) |
| genero | string | filtro (opções derivadas) |
| imagem_url | string? | substituto visual quando ausente |
| status | `EM_CARTAZ` \| `BREVE` \| `FORA_DE_CARTAZ` | `FORA_DE_CARTAZ` descartado (FR-018) |

## Cinema (da API)
| Campo | Tipo | Observação |
|---|---|---|
| id | string (uuid) | chave de lista |
| nome | string | busca por texto |
| cidade | string | filtro dependente do estado |
| estado | string (UF, 2 letras) | filtro |
| endereco | string | exibido |
| logo_url | string \| null | URL remota do logo (como `imagem_url` dos filmes); `null` → substituto neutro |
| ativo | boolean | só `true` (pedido `ativo=true`) |

## Página da API
`{ itens: T[], pagina: { pagina, tamanho, total, tem_proxima } }`

## FiltrosFilmes (estado da UI)
`{ texto: string; genero: string | ''; classificacao: string | ''; status: 'EM_CARTAZ' | 'BREVE' | '' }` — vazio = sem restrição; critérios combinados por E.

## FiltrosCinemas (estado da UI)
`{ texto: string; estado: string | ''; cidade: string | '' }`

Regra: ao mudar `estado`, `cidade` é zerada se não pertencer ao novo estado; opções de cidade = cidades distintas do estado escolhido (ou todas, sem estado).

## Estado de lista
`carregando | erro(mensagem) | pronto(itens)`; derivados: `vazio` (pronto sem itens), `semResultado` (filtrado sem itens).

## Aba
`'filmes' | 'cinemas'`, padrão `filmes`, refletida em `?aba=`.
