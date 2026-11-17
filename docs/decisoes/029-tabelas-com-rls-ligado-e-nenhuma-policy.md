# 029. Tabelas com RLS ligado e nenhuma policy

- **Data:** 2026-11-11
- **Status:** ativa

## Contexto

O Supabase publica uma API REST para tudo que está no schema `public`. Sem RLS, a chave pública do projeto leria as anotações. O Garimpo entra como dono das tabelas, que não passa pelo RLS.

## Decisão

Tabelas com RLS ligado e nenhuma policy.

## Consequência

Migrada do antigo `docs/DECISIONS.md`, que guardava a decisão e o porquê; o efeito está descrito acima.
