# Research: Aplicação React Vazia

## Ferramenta de build

- **Decision**: Vite com `@vitejs/plugin-react`.
- **Rationale**: Padrão atual para SPAs React (Create React App está descontinuado); inicialização rápida, build simples, e é o que o frontend anterior deste repositório usava.
- **Alternatives considered**: Next.js (SSR/roteamento desnecessários para app vazia); Parcel/webpack (mais configuração); CRA (obsoleto).

## Linguagem

- **Decision**: TypeScript em modo estrito.
- **Rationale**: O frontend anterior já era TS (`.tsx`); detecta erros cedo sem custo para uma app vazia.
- **Alternatives considered**: JavaScript puro (menos seguro para evolução futura).

## Gerenciador de pacotes

- **Decision**: npm (v11 instalado) com `package-lock.json` versionado.
- **Rationale**: Já disponível com Node, sem ferramentas extras.
- **Alternatives considered**: pnpm/yarn (não instalados/necessários).

## Testes

- **Decision**: Não adicionar framework de testes agora.
- **Rationale**: "Vazia" exclui lógica a testar; o build com checagem de tipos valida a base. Vitest pode ser adicionado quando houver comportamento.
- **Alternatives considered**: Vitest + Testing Library (excesso para este escopo).

## Preservação de arquivos

- **Decision**: Gerar o scaffold em diretório temporário e copiar apenas os arquivos necessários, ou criar os arquivos manualmente, sem sobrescrever `.gitignore`/`.dockerignore` existentes.
- **Rationale**: Atende FR-005; scaffolders em diretório não vazio podem pedir confirmação ou apagar conteúdo.
