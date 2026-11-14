<!--
Sync Impact Report
- Version change: (template não preenchido) → 1.0.0
- Princípios adicionados: I. Simplicidade e YAGNI; II. Isolamento do Frontend; III. Tipagem Estrita;
  IV. Verificação Obrigatória; V. Preservação do Workspace
- Seções adicionadas: Restrições Adicionais; Fluxo de Desenvolvimento; Governance
- Seções removidas: nenhuma
- Templates: plan-template.md ✅ (Constitution Check genérico, sem alteração necessária);
  spec-template.md ✅; tasks-template.md ✅
- TODOs adiados: nenhum
-->

# Constituição do Frontend Web (ingressos-golang/web)

## Core Principles

### I. Simplicidade e YAGNI
O frontend MUST conter apenas o que a funcionalidade atual exige. Dependências, bibliotecas
de UI, roteamento, estado global e abstrações MUST ser adicionados somente quando uma
funcionalidade especificada os requerer, e a justificativa MUST constar no plano.

### II. Isolamento do Frontend
`web/` é um projeto independente, com seu próprio `package.json`, sem acoplamento ao
código Go dos serviços (`catalogo`, `estoque`, `pagamento`, `notificacao`). A comunicação
com os serviços MUST ocorrer apenas por seus contratos REST publicados (OpenAPI em `/docs`)
e somente quando uma especificação a exigir.

### III. Tipagem Estrita
O código-fonte MUST ser TypeScript com `strict` habilitado. Erros de tipo MUST falhar o
build; `any` implícito é proibido.

### IV. Verificação Obrigatória
Toda entrega MUST passar em `npm run build` (checagem de tipos + build de produção) com
código de saída 0 e abrir sem erros ou avisos no console. Quando houver comportamento de
negócio, testes automatizados MUST acompanhar a funcionalidade.

### V. Preservação do Workspace
Mudanças MUST NOT remover ou sobrescrever arquivos existentes não relacionados à tarefa
(`.claude/`, `.specify/`, `.gitignore`, `.dockerignore`, `specs/`). `node_modules/` e
`dist/` MUST permanecer fora do controle de versão.

## Restrições Adicionais

- Stack base: Node.js LTS, React, Vite, TypeScript e npm (com `package-lock.json` versionado).
- Documentação, especificações e comentários de commit em português, conforme o workspace.
- Autenticação, quando existir, MUST usar o realm Keycloak `cinema` já provido pela plataforma.

## Fluxo de Desenvolvimento

- O fluxo spec-kit (specify → clarify → plan → tasks → analyze → implement) é o caminho
  padrão para novas funcionalidades.
- Commits seguem Conventional Commits e são feitos diretamente na `master` (sem branches de
  feature neste workspace).
- Todo plano MUST registrar a verificação de conformidade com esta constituição na seção
  "Constitution Check".

## Governance

Esta constituição prevalece sobre outras práticas do frontend. Emendas exigem edição deste
arquivo via `/speckit-constitution`, com Sync Impact Report e atualização da versão:
MAJOR para remoção ou redefinição incompatível de princípios, MINOR para novo princípio ou
ampliação material, PATCH para esclarecimentos. Planos e revisões MUST verificar a
conformidade; complexidade adicional MUST ser justificada em "Complexity Tracking".

**Version**: 1.0.0 | **Ratified**: 2026-11-14 | **Last Amended**: 2026-11-14
