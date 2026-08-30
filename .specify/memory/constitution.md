<!--
Sync Impact Report
==================
Version change: TEMPLATE (não ratificada) → 1.0.0
Bump rationale: MAJOR inicial. Este arquivo estava com todos os placeholders do
  template intactos — nunca houve uma v0.x aqui. A primeira ratificação de uma
  constituição é 1.0.0 por definição.

Escopo: esta é a constituição do WORKSPACE (a plataforma de ingressos como um
  todo), não a de um serviço. Ela contém apenas os quatro princípios ditados pelo
  mantenedor em 2026-08-30, que valem para qualquer serviço e para qualquer agente
  trabalhando neste repositório. Regras técnicas de um serviço específico
  (arquitetura hexagonal, eventos, orçamento de integração) permanecem nas
  constituições de cada serviço.

Origem: os quatro princípios já existem, com texto idêntico, nas constituições
  v1.1.0 dos dois serviços implementados. Correspondência de numeração:
    Workspace I   ↔ Catalogo VI   ↔ Estoque VII    (complexidade)
    Workspace II  ↔ Catalogo VII  ↔ Estoque VIII   (testes de domínio e API)
    Workspace III ↔ Catalogo VIII ↔ Estoque IX     (código é a fonte da verdade)
    Workspace IV  ↔ Catalogo IX   ↔ Estoque X      (divergência é pergunta)
  A partir desta ratificação, eles passam a valer também para os serviços ainda
  não implementados (Notificacao, Pagamento) sem precisar ser recopiados.

Modified principles: nenhum. Não havia princípio anterior neste arquivo.

Added sections: todas — o arquivo era um template vazio.
  I. Complexidade Só Entra Se For Necessária ou Pedida
  II. Domínio e API Têm Teste Automatizado
  III. O Código é a Fonte da Verdade
  IV. Divergência Entre Código e Spec é Pergunta, Não Decisão
  Fluxo de Desenvolvimento e Portões de Qualidade
  Relação Com as Constituições de Serviço
  Governance

Removed sections: nenhuma.

Templates requiring updates:
  ✅ .specify/templates/tasks-template.md — atualizado: tarefas de teste para
     domínio e para interfaces expostas deixaram de ser OPTIONAL (princípio II).
     Mesma edição já aplicada nos dois serviços.
  ✅ .specify/templates/plan-template.md — o "Constitution Check" é genérico e
     continua válido; a tabela "Complexity Tracking" já materializa o princípio I.
     Nenhuma alteração necessária.
  ✅ .specify/templates/spec-template.md — não referencia princípios; nenhum
     princípio novo impõe seção nova à spec.
  ✅ .specify/templates/checklist-template.md — genérico, sem referência a
     princípios. Nenhuma alteração necessária.
  ✅ .specify/templates/commands/ — diretório inexistente neste projeto.
  ✅ README.md — inexistente na raiz do workspace. Nenhuma alteração possível.
  ⚠ servico-notificacao-spec.md e servico-pagamento-spec.md — specs de rascunho na
     raiz, escritas antes desta ratificação. Ao virarem features de spec-kit, MUST
     receber veredito contra os quatro princípios.

Follow-up TODOs: nenhum. Todos os placeholders foram preenchidos.
-->

# Plataforma de Ingressos Constitution

Esta constituição governa o workspace inteiro — todos os serviços, presentes e
futuros, e todo agente que trabalhe neste repositório. Ela contém apenas o que vale
em qualquer serviço; regra técnica específica de um serviço mora na constituição
daquele serviço.

## Core Principles

### I. Complexidade Só Entra Se For Necessária ou Pedida

Uma abstração, camada, padrão, dependência, opção de configuração ou ponto de
extensão MUST NOT ser introduzido sem uma necessidade demonstrada no escopo
corrente ou um pedido explícito do mantenedor. Antecipação de requisito futuro
NÃO é necessidade demonstrada.

Diante de duas soluções que atendem ao requisito, MUST ser escolhida a mais
simples — a que tem menos partes móveis, menos indireção e menos conceitos a
aprender. Quando a mais simples for rejeitada, o plano MUST registrar qual era e
por que não serviu.

Escopo MUST NOT ser ampliado por conta própria: o que foi pedido é o que se
entrega. Melhoria percebida fora do escopo MUST ser proposta, não implementada.

**Rationale**: cada abstração que não paga o próprio custo é dívida cobrada em
toda leitura futura do código. O custo aparece meses depois, na cabeça de quem não
escreveu a linha, e é invisível na hora em que parece elegante escrevê-la.

### II. Domínio e API Têm Teste Automatizado

O núcleo de domínio MUST ter teste automatizado cobrindo suas regras — cada
invariante, cada transição de estado e cada condição de erro que o domínio decide.
Esses testes MUST rodar sem banco, sem rede e sem servidor.

Toda operação de interface exposta a terceiros — síncrona ou por evento — MUST ter
teste automatizado exercitando o caminho de sucesso e cada categoria de erro
declarada no contrato.

Um teste que não pode falhar por causa do defeito que ele alega cobrir não conta
como cobertura. Percentual de linhas cobertas NÃO é critério de aceitação e MUST
NOT ser usado como portão.

Adaptadores, infraestrutura, código gerado e fiação de composição NÃO estão
sujeitos a esta obrigação; testá-los é decisão de custo-benefício, não regra.

**Rationale**: domínio e contrato de API são as duas coisas que outros dependem e
que mudam sob pressão. Teste automatizado nesses dois lugares é o que permite
alterar o código com confiança — que é exatamente o que o princípio III exige que
se faça.

### III. O Código é a Fonte da Verdade

O comportamento do sistema é o que o código executa. Especificação, plano, tarefas
e demais artefatos de spec-kit são instrumentos de projeto: existem para produzir
código melhor, MUST NOT ser tratados como descrição autoritativa do sistema, e
MUST NOT ser citados como evidência de que algo funciona.

Uma afirmação sobre o comportamento atual MUST ser verificada no código ou por
execução — nunca inferida da spec. Quando spec e código discordarem sobre o que o
sistema faz, o código está certo por definição sobre o *que é*; qual dos dois está
certo sobre o *que deveria ser* é a pergunta do princípio IV.

**Rationale**: a spec é adotada aqui porque produz resultado melhor, não porque se
acredite que ela virará a fonte da verdade. Tratar documento como verdade leva a
afirmar com confiança coisas que o binário não faz — o modo de falha mais caro que
existe, porque a correção só chega em produção.

### IV. Divergência Entre Código e Spec é Pergunta, Não Decisão

Ao encontrar incoerência entre código e qualquer artefato de spec, o agente MUST
parar e perguntar ao mantenedor qual dos dois deve ser modificado, apresentando: o
que a spec diz, o que o código faz, e o caminho mais curto para cada uma das duas
resoluções.

O agente MUST NOT escolher sozinho — nem "corrigindo" o código para casar com a
spec, nem atualizando a spec para casar com o código. MUST NOT silenciar a
divergência prosseguindo como se ela não existisse.

Enquanto a resposta não vier, todo trabalho que não depende dela MUST seguir; só a
parte dependente fica bloqueada.

**Rationale**: uma divergência é sempre um de dois defeitos — implementação errada
ou spec desatualizada — e só o mantenedor sabe qual. Adivinhar acerta metade das
vezes e apaga a evidência nas duas.

## Fluxo de Desenvolvimento e Portões de Qualidade

- O plano de cada feature MUST conter uma seção de verificação contra esta
  constituição, com o veredito por princípio.
- Violação de princípio MUST ser resolvida corrigindo a especificação, o plano ou
  as tarefas — nunca diluindo, reinterpretando ou ignorando o princípio.
- Uma violação que se prove necessária MUST ser registrada explicitamente, com a
  necessidade que a motiva e a alternativa mais simples que foi rejeitada, junto
  com o motivo da rejeição (princípio I).
- Uma feature MUST NOT ser dada por concluída enquanto o domínio e as operações
  expostas que ela introduz ou altera não tiverem os testes exigidos pelo
  princípio II, verdes.
- O estado de conclusão de uma tarefa MUST ser aferido no código, nunca na marcação
  do artefato de tarefas: uma tarefa marcada como feita cujo efeito não existe no
  código é uma divergência e MUST ser tratada pelo princípio IV.
- Um relatório de progresso MUST descrever o que foi verificado em execução e o que
  não foi. Dizer que algo funciona sem ter rodado é violação do princípio III.

## Relação Com as Constituições de Serviço

Cada serviço MAY ter a sua própria constituição em `<servico>/.specify/memory/`.
Ela MUST ser lida como acréscimo a esta, nunca como substituição: um serviço PODE
impor regra adicional, MUST NOT afrouxar princípio daqui.

Os quatro princípios acima já constam, com texto idêntico e numeração própria, das
constituições v1.1.0 do Servico-Catalogo (VI a IX) e do Servico-Estoque (VII a X).
Essa duplicação é deliberada e MUST ser mantida em sincronia: emenda a um princípio
daqui MUST declarar, no Sync Impact Report, se as constituições de serviço
acompanham a mudança ou se passam a divergir. Serviço novo MUST NOT recopiar estes
princípios — herda-os deste arquivo.

## Governance

Esta constituição prevalece sobre qualquer outra prática, convenção ou preferência
adotada no workspace. Em conflito entre a constituição e um documento de feature, a
constituição vence e o documento MUST ser corrigido.

**Emendas**: qualquer alteração MUST ser proposta por escrito, com a justificativa
e o impacto sobre artefatos existentes, e MUST ser aplicada em uma mudança dedicada
— nunca como efeito colateral de uma feature.

**Versionamento**: MAJOR para remoção ou redefinição incompatível de princípio;
MINOR para princípio ou seção nova, ou ampliação material de orientação existente;
PATCH para esclarecimento, redação ou correção sem efeito semântico.

**Conformidade**: toda revisão de código e todo plano de feature MUST verificar
aderência aos quatro princípios. O princípio II MUST ter verificação automatizada
na esteira; os princípios I, III e IV são verificados em revisão, porque nenhum
deles é decidível por ferramenta.

**Version**: 1.0.0 | **Ratified**: 2026-08-30 | **Last Amended**: 2026-08-30
