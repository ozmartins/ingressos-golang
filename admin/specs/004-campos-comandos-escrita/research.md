# Research: Comandos de Escrita Guiados por Campos

## D1 — Onde vive o estado do diálogo
- **Decisão**: máquina de estados pura em `dialogo.ts` (`iniciar`, `responder`, `cancelar`), com o estado guardado em `useState` no `Terminal`. O `despachar` ganha `continuarDialogo` para o envio final.
- **Racional**: mantém a regra testável sem React e reaproveita o executor de requisições (autenticação, tratamento de 401, formatação de erros) já existente.
- **Alternativas**: `window.prompt` (bloqueia e foge do estilo do terminal; descartado); modal/formulário HTML (fora do conceito de terminal; descartado); estado em módulo global (difícil de testar; descartado).

## D2 — Descrição dos campos
- **Decisão**: formulários declarativos escritos à mão em `api/formularios.ts`, espelhando `FilmeEntrada`, `CinemaEntrada`, `SalaEntrada`, `SalaAtualizacao`, `SessaoEntrada` e `SessaoAtualizacao` do OpenAPI.
- **Racional**: são 6 schemas pequenos e estáveis; gerar a partir do OpenAPI exigiria parser de YAML/`$ref` no navegador (viola YAGNI).
- **Alternativas**: ler `/openapi.yaml` em tempo de execução (dependência nova + acoplamento); gerar código no build (complexidade sem ganho).

## D3 — Tipos de campo e conversão
- **Decisão**: `texto`, `inteiro` (com mínimo), `logico` (aceita `s/n`, `sim/não`, `true/false`), `opcao` (enum, case-insensitive, normalizado para o valor canônico), `uuid`, `decimal` (texto com até 2 casas; a API espera string), `datahora` (aceita ISO-8601; convertido para o formato com fuso) e `fileiras` (lista composta).
- **Racional**: cobre exatamente os tipos dos schemas. `preco_base` é string decimal no contrato, então é enviado como texto, não número. A validação local usa mínimos, tamanhos e o padrão `^[A-Z]{1,5}$` do contrato; o serviço continua sendo a autoridade final (erros 400/409 exibidos como hoje).
- **Alternativas**: validar só no servidor (viola FR-006/SC-003).

## D4 — Fileiras da sala
- **Decisão**: sub-diálogo repetido: letra → assentos → tipo (padrão `NORMAL`) → "adicionar outra fileira? (s/n)". Letra normalizada para maiúsculas; ao menos uma exigida na criação. Na atualização de sala as fileiras **não** são pedidas (o contrato as trata como imutáveis e opcionais).
- **Racional**: reflete a regra do contrato e US3.

## D5 — Cancelar, resumo e confirmação
- **Decisão**: a palavra `cancelar` (e a tecla Esc) abandona o diálogo; ao final, exibe resumo `campo: valor` e pede `s/n`. `n` encerra sem enviar.
- **Racional**: FR-009/FR-010; Esc é conveniência, `cancelar` funciona em qualquer ambiente.
- **Alternativas**: Ctrl+C (conflita com copiar texto no navegador).

## D6 — Histórico e segredo
- **Decisão**: com diálogo ativo, a linha digitada é ecoada na saída, mas **não** vai ao histórico (`adicionar` é pulado).
- **Racional**: FR-012. Nenhum campo é secreto, mas o requisito vale para qualquer resposta.

## D7 — Comandos com JSON na linha
- **Decisão**: `criar` com qualquer argumento e `atualizar <id> <extra>` retornam erro explicando o novo uso (`filmes criar` / `filmes atualizar <id>`), sem requisição e sem abrir diálogo.
- **Racional**: FR-011; evita enviar silenciosamente um corpo ambíguo.

## D8 — Ordem das verificações
- **Decisão**: argumentos → autenticação → abertura do diálogo (FR-008: não desperdiçar o preenchimento).
