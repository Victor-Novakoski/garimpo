# 032. Roteador da API é o `net/http` da biblioteca padrão, e não o `chi`

- **Data:** 2026-11-11
- **Status:** ativa

## Contexto

Desde o Go 1.22 o roteador padrão entende método e parâmetro no caminho (`GET /jobs/{number}`). Para quatro rotas, uma dependência a menos.

## Decisão

Roteador da API é o `net/http` da biblioteca padrão, e não o `chi`.

## Consequência

Migrada do antigo `docs/DECISIONS.md`, que guardava a decisão e o porquê; o efeito está descrito acima.
