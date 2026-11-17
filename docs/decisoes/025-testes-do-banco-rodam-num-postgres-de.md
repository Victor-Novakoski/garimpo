# 025. Testes do banco rodam num Postgres de verdade, só quando…

- **Data:** 2026-11-11
- **Status:** ativa

## Contexto

Simular o banco testaria a simulação. Cada teste ganha um schema próprio e apaga no fim. Sem a variável eles são pulados, e as mesmas regras continuam testadas pelo arquivo, sem rede.

## Decisão

Testes do banco rodam num Postgres de verdade, só quando `GARIMPO_TEST_DATABASE_URL` existe; a CI sobe um.

## Consequência

Migrada do antigo `docs/DECISIONS.md`, que guardava a decisão e o porquê; o efeito está descrito acima.
