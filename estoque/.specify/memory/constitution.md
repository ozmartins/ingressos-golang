<!--
Sync Impact Report
==================
Version change: (template não preenchido) → 1.0.0
Bump rationale: primeira ratificação deste serviço. Não há versão anterior com
  princípios definidos, portanto não se aplica MAJOR/MINOR/PATCH sobre conteúdo
  prévio.

Origem: os princípios I a V são adotados sem alteração semântica da constituição
  v1.0.0 do Servico-Catalogo (../catalogo/.specify/memory/constitution.md), por
  decisão de governar os dois serviços do mesmo sistema pelas mesmas regras. O
  princípio VI é próprio deste serviço.

Modified principles:
  [PRINCIPLE_1_NAME] → I. Dependências Apontam Para Dentro
  [PRINCIPLE_2_NAME] → II. Configuração Externa, Falha na Largada
  [PRINCIPLE_3_NAME] → III. Fronteira de Estado Explícita
  [PRINCIPLE_4_NAME] → IV. Erro é Contrato
  [PRINCIPLE_5_NAME] → V. Integração Síncrona Tem Orçamento

Added sections:
  VI. Entrega de Fato é Ao Menos Uma Vez (princípio novo, sem equivalente no
    Servico-Catalogo, que não publica nem consome eventos)
  [SECTION_2_NAME] → Restrições Técnicas
  [SECTION_3_NAME] → Fluxo de Desenvolvimento e Portões de Qualidade

Removed sections: nenhuma

Templates requiring updates:
  ✅ .specify/templates/plan-template.md — seção "Constitution Check" já existe e
     é genérica; nenhuma alteração necessária
  ✅ .specify/templates/spec-template.md — não referencia princípios; a
     constituição não impõe seção nova à spec
  ✅ .specify/templates/tasks-template.md — não referencia princípios; os
     princípios I, V e VI geram tarefas de verificação, não categorias novas
  ✅ .specify/templates/commands/ — diretório inexistente neste projeto
  ✅ specs/001-estoque-bloqueio-poltronas/plan.md — seção "Constitution Check"
     atualizada de "informativa" para veredito formal contra esta constituição
  ⚠ README.md — ausente no repositório; quando criado (tarefa T097 de
    specs/001-estoque-bloqueio-poltronas/tasks.md), deve referenciar esta
    constituição

Follow-up TODOs: nenhum. Todos os placeholders foram preenchidos.
-->

# Servico-Estoque Constitution

## Core Principles

### I. Dependências Apontam Para Dentro

O núcleo de domínio e os casos de uso MUST NOT importar adaptadores, drivers,
clientes de rede, framework de servidor ou qualquer biblioteca de infraestrutura.
Toda dependência externa MUST ser expressa como uma interface declarada pelo
núcleo e implementada por um adaptador. A composição MUST ocorrer em um único
ponto de entrada do binário.

Esta regra MUST ser verificada mecanicamente por um analisador de imports na
esteira de build, não por revisão humana.

**Rationale**: a regra existe para manter as regras de negócio testáveis sem
banco, sem rede e sem servidor, e para que trocar um adaptador não custe uma
reescrita. Revisão humana falha em detectar um import errado; um linter não.

### II. Configuração Externa, Falha na Largada

Endereço, credencial, segredo, material criptográfico ou parâmetro de ambiente
MUST NOT ser embutido no código ou no artefato entregue. Toda configuração MUST
ser injetada pelo ambiente, lida e validada uma única vez na inicialização. O
processo MUST recusar subir quando faltar configuração obrigatória ou quando um
valor for malformado.

**Rationale**: falhar alto na largada é barato; descobrir um endereço vazio na
primeira requisição do dia é caro. O mesmo artefato precisa rodar em qualquer
ambiente sem recompilação, e um segredo versionado é um incidente esperando data.

### III. Fronteira de Estado Explícita

Um serviço MUST NOT persistir, cachear ou reconstruir estado cuja fonte de
verdade pertence a outro serviço. Quando uma decisão depende de estado alheio,
ela MUST ser delegada ao dono daquele estado, e o resultado MUST ser repassado
sem reinterpretação.

Quando o serviço É o dono de um estado, a decisão sobre esse estado MUST ser
tomada por ele e MUST NOT depender da disponibilidade de um armazenamento
auxiliar: uma peça de infraestrutura que não seja a fonte de verdade MUST poder
ser perdida sem que nenhuma resposta mude de valor.

Cache de dados de domínio que sirva valor desatualizado ao cliente MUST ser
declarado explicitamente na especificação, com a janela de defasagem tolerada,
ou não existir.

**Rationale**: estado duplicado diverge, e a divergência aparece no pior momento
possível. Do lado de quem é dono, amarrar a correção a um cache torna a garantia
tão durável quanto o cache — e caches são projetados para serem descartáveis.

### IV. Erro é Contrato

Toda resposta de falha MUST ser legível por máquina e carregar uma categoria
estável que o cliente possa inspecionar sem interpretar texto livre. A categoria
MUST ser versionada como parte do contrato da interface; o texto humano MAY mudar
de redação livremente.

Uma resposta de erro MUST NOT expor detalhe interno — mensagem de driver,
endereço de serviço, consulta, nome de fila ou pilha de execução. Esses detalhes
MUST ir para os registros, correlacionados por um identificador de rastreamento
que também acompanha a resposta.

**Rationale**: um cliente que precisa comparar strings de mensagem para decidir o
que fazer quebra na primeira revisão de texto. Detalhe interno em resposta de
erro é superfície de ataque e ruído para quem integra.

### V. Integração Síncrona Tem Orçamento

Toda chamada a serviço ou armazenamento externo no caminho de uma requisição MUST
declarar um tempo máximo de espera e uma política de falha explícita. Retentativa
automática MUST NOT ser aplicada a operação não idempotente. Falhas consecutivas
MUST levar a interrupção temporária das chamadas, com retomada automática, sem
intervenção manual.

A indisponibilidade de uma dependência MUST NOT degradar funcionalidades que não
dependem dela.

**Rationale**: sem orçamento declarado, a latência de um parceiro vira a latência
do produto. Retentar uma operação que já pode ter sido efetivada do outro lado
cria estado órfão — um bloqueio, uma cobrança, um registro que ninguém reivindica.

### VI. Entrega de Fato é Ao Menos Uma Vez

Todo fato publicado por este serviço MUST ser persistido na mesma transação que
produziu o fato, e reenviado até ser aceito pelo intermediário de mensagens. A
resposta síncrona MUST NOT esperar pela publicação.

Todo consumo de fato MUST ser idempotente por uma chave declarada no contrato do
evento, MUST ser confirmado somente após o efeito estar durável, e MUST tolerar
duplicata e ordem invertida sem produzir efeito adicional. Mensagem que não possa
ser processada por erro definitivo MUST ser retirada do fluxo normal para
inspeção, nunca reprocessada indefinidamente.

Nenhum contrato de evento MUST prometer entrega exatamente-uma-vez.

**Rationale**: intermediários de mensagens duplicam e reordenam — é o contrato
real deles, e desenhar contra isso é desenhar contra a realidade. Publicar fora
da transação perde o fato quando o processo morre entre as duas; esperar pela
publicação coloca a latência do broker dentro do orçamento da requisição. A
idempotência no consumidor é o que torna as duas coisas seguras.

## Restrições Técnicas

Aplicam-se a todo serviço governado por esta constituição:

- **Observabilidade**: cada requisição e cada mensagem consumida MUST emitir
  registro estruturado com identificador de correlação, operação, desfecho e
  duração. Operações expostas e consumos MUST expor métricas de volume, latência
  e desfecho. O contexto de rastreamento recebido MUST ser propagado às chamadas
  de saída **e aos fatos publicados**, de modo que um fluxo atravessando
  mensageria permaneça reconstituível de ponta a ponta.
- **Contrato antes da implementação**: interfaces expostas a terceiros — síncronas
  ou por evento — MUST ter contrato escrito e versionado antes do código que as
  serve.
- **Mudança incompatível é versão nova**: alterar a forma de uma resposta ou de um
  evento já publicado MUST resultar em nova versão da interface, nunca em
  substituição silenciosa. Adição de campo ou operação MUST ser compatível com
  clientes já compilados.
- **Segredos**: segredos e material criptográfico MUST NOT aparecer em código, em
  registros ou em resposta de erro. Credenciais recebidas MUST NOT ser
  registradas, nem mesmo truncadas.

## Fluxo de Desenvolvimento e Portões de Qualidade

- Toda feature MUST passar por especificação antes do planejamento, e por
  planejamento antes da implementação. A especificação descreve o quê e o porquê;
  o plano descreve o como.
- O plano de cada feature MUST conter uma seção de verificação contra esta
  constituição, com o veredito por princípio.
- Violação de princípio MUST ser resolvida corrigindo a especificação, o plano ou
  as tarefas — nunca diluindo, reinterpretando ou ignorando o princípio.
- Uma violação que se prove necessária MUST ser registrada explicitamente, com a
  necessidade que a motiva e a alternativa mais simples que foi rejeitada, junto
  com o motivo da rejeição.
- Requisitos e critérios de sucesso MUST ser verificáveis. Um critério que nenhuma
  implementação plausível possa violar não é um critério — MUST ser reescrito ou
  removido.
- Invariante de concorrência e garantia de idempotência MUST ter teste
  automatizado com infraestrutura real, não com dublê. Um teste que não pode
  falhar por causa do defeito que ele alega cobrir não conta como cobertura.

## Governance

Esta constituição prevalece sobre qualquer outra prática, convenção ou preferência
adotada no projeto. Em conflito entre a constituição e um documento de feature, a
constituição vence e o documento MUST ser corrigido.

**Emendas**: qualquer alteração MUST ser proposta por escrito, com a justificativa
e o impacto sobre artefatos existentes, e MUST ser aplicada em uma mudança dedicada
— nunca como efeito colateral de uma feature.

**Versionamento**: MAJOR para remoção ou redefinição incompatível de princípio;
MINOR para princípio ou seção nova, ou ampliação material de orientação existente;
PATCH para esclarecimento, redação ou correção sem efeito semântico.

**Conformidade**: toda revisão de código e todo plano de feature MUST verificar
aderência aos princípios. Os princípios I, V e VI MUST ter verificação
automatizada na esteira; os demais são verificados em revisão. Complexidade
adicional MUST ser justificada contra a alternativa mais simples.

**Relação com o Servico-Catalogo**: os princípios I a V são idênticos aos da
constituição v1.0.0 daquele serviço, por decisão deliberada. Emenda que altere um
princípio compartilhado MUST declarar se o serviço irmão acompanha a mudança ou se
as constituições passam a divergir.

**Version**: 1.0.0 | **Ratified**: 2026-08-29 | **Last Amended**: 2026-08-29
