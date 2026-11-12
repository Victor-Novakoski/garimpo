# 031. Driver do Postgres é o `pgx`, sem preparar comandos

- **Data:** 2026-11-11
- **Status:** ativa

## Contexto

É o driver mais usado em Go e fala o protocolo nativo. O pooler do Supabase divide as conexões entre clientes, e comando preparado não sobrevive de uma consulta para a outra.

## Decisão

Driver do Postgres é o `pgx`, sem preparar comandos.

## Consequência

Migrada do antigo `docs/DECISIONS.md`, que guardava a decisão e o porquê; o efeito está descrito acima.
