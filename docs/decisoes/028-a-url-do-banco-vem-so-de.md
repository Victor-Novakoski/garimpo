# 028. A URL do banco vem só de variável de ambiente

- **Data:** 2026-11-11
- **Status:** ativa

## Contexto

A URL leva a senha. Em flag, ela ficaria no histórico do terminal e na lista de processos.

## Decisão

A URL do banco vem só de variável de ambiente; o `-state` escolhe entre arquivo e banco, mas nunca recebe a URL.

## Consequência

Migrada do antigo `docs/DECISIONS.md`, que guardava a decisão e o porquê; o efeito está descrito acima.
