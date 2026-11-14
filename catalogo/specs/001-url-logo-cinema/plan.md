# Implementation Plan: URL do logo do cinema

**Branch**: `001-url-logo-cinema` (master, sem branch de feature) | **Date**: 2026-11-14 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `/specs/001-url-logo-cinema/spec.md`

## Summary

Adicionar ao cinema o atributo opcional `logo_url`: coluna nova em `catalogo.cinemas` (migração 000007, reversível), campo no domínio com validação (aparar, vazio = sem logo, http/https, ≤ 2048), propagação por casos de uso → repositório → DTOs HTTP (entrada e saída) e atualização do `openapi.yaml`. O PUT é substituição completa, então omitir o campo remove o logo.

## Technical Context

**Language/Version**: Go (módulo `github.com/oseias/ingressos-golang/catalogo`, ver `go.mod`)

**Primary Dependencies**: net/http (ServeMux com `PathValue`), GORM + pgx, golang-migrate

**Storage**: PostgreSQL, schema `catalogo`, tabela `cinemas`

**Testing**: `go test -race ./...` (unitários de domínio/casos de uso, contrato em `test/contract`, integração em `test/integration` com `-tags=integration`)

**Target Platform**: serviço Linux em Docker

**Project Type**: web-service (arquitetura hexagonal)

**Performance Goals**: N/A — coluna de texto acrescentada a linhas já lidas; sem impacto relevante

**Constraints**: núcleo (`domain`, `usecase`) sem importar adaptadores (`depguard` `nucleo-sem-adaptadores`); contrato vive em `openapi.yaml`; migração com SQL qualificado pelo schema

**Scale/Scope**: 1 coluna, 1 campo de domínio, 2 DTOs, 1 migração

## Constitution Check

A constituição em `.specify/memory/constitution.md` é o template não preenchido (sem princípios ratificados): nenhum gate aplicável. Restrições relevantes vêm do `CLAUDE.md` e foram respeitadas: arquitetura hexagonal, contrato em `openapi.yaml`, migração reversível. **Gate: PASS** (antes e depois do desenho).

## Project Structure

### Documentation (this feature)

```text
specs/001-url-logo-cinema/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/cinema-logo-url.md
└── tasks.md             # gerado por /speckit-tasks
```

### Source Code (arquivos tocados)

```text
migrations/
├── 000007_logo_do_cinema.up.sql        # novo
└── 000007_logo_do_cinema.down.sql      # novo
internal/domain/catalogo/cinema.go      # campo LogoURL + validação
internal/domain/catalogo/cinema_test.go # testes de validação (novo, se não existir)
internal/adapter/postgres/modelos.go    # cinemaRow.LogoURL
internal/adapter/postgres/cinema_repository.go  # leitura, insert e update
internal/adapter/http/dto.go            # cinemaDTO / cinemaEntradaDTO
internal/adapter/http/openapi/openapi.yaml      # schemas Cinema e CinemaEntrada
test/contract/cinemas_test.go           # cenários do campo
test/integration/catalogo_test.go       # ida e volta no banco
```

**Structure Decision**: serviço único existente (`catalogo`); nenhuma estrutura nova. Casos de uso `criar_cinema`/`atualizar_cinema` já repassam `DadosCinema` inteiro, então não precisam de mudança de lógica.

## Complexity Tracking

Sem violações.
