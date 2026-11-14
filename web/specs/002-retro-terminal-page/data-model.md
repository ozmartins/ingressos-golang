# Data Model: Página de Terminal Retrô

Estado vive apenas em memória, durante a sessão.

## Linha

| Campo | Tipo | Descrição |
|-------|------|-----------|
| `tipo` | `'sistema' \| 'entrada' \| 'saida' \| 'erro'` | Papel da linha na saída |
| `texto` | `string` | Conteúdo em texto puro (nunca marcação) |

- Eco de comando: `tipo: 'entrada'`, texto com prompt (`> comando`).
- Retenção: no máximo 500 linhas; as mais antigas são descartadas.

## Histórico

- Lista de strings (comandos não vazios, mais antigo → mais recente) e um ponteiro de navegação.
- ↑ move para o anterior (para no primeiro); ↓ move para o seguinte e, passando do último, devolve campo vazio.
- Entradas vazias não são registradas.

## Comandos embutidos (contrato interno)

`executar(entrada: string, agora: Date): { linhas: Linha[]; limpar: boolean }`

| Comando | Resultado |
|---------|-----------|
| `help` | Lista os comandos disponíveis e suas descrições |
| `clear` | `limpar: true`; esvazia a saída |
| `echo <texto>` | Repete `<texto>` (vazio produz linha vazia) |
| `date` | Data/hora atual em pt-BR |
| `whoami` | Nome do usuário simulado (`visitante`) |
| `about` | Descrição breve do terminal |
| outro | `erro`: `comando não encontrado: <nome>` |
| vazio | Nenhuma linha de resultado |

Nomes de comando não diferenciam maiúsculas/minúsculas; espaços extras são ignorados.
