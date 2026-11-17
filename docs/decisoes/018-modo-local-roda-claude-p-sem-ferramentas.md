# 018. Modo local roda `claude -p` sem ferramentas, sem MCP, em modo seguro…

- **Data:** 2026-11-10
- **Status:** ativa

## Contexto

A descrição da vaga é texto de terceiros. Sem ferramenta, uma vaga maliciosa não lê arquivo nem roda comando; só consegue mexer na própria classe. O modo local também tira `ANTHROPIC_API_KEY` do ambiente do `claude`, para usar o plano e não cobrar a API sem querer.

## Decisão

Modo local roda `claude -p` sem ferramentas, sem MCP, em modo seguro e numa pasta vazia, com a vaga no stdin.

## Consequência

Migrada do antigo `docs/DECISIONS.md`, que guardava a decisão e o porquê; o efeito está descrito acima.
