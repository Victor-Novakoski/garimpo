# 009. Normalização sem dependência externa

- **Data:** 2026-11-10
- **Status:** ativa

## Contexto

`golang.org/x/text` atual exige Go 1.26; a tabela cobre o português e mantém o módulo sem dependências.

## Decisão

Normalização sem dependência externa: acentos tirados por uma tabela própria.

## Consequência

Migrada do antigo `docs/DECISIONS.md`, que guardava a decisão e o porquê; o efeito está descrito acima.
