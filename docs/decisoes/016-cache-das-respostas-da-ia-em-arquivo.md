# 016. Cache das respostas da IA em arquivo, com chave de modelo + prompt +…

- **Data:** 2026-11-10
- **Status:** ativa

## Contexto

A mesma vaga aparece por semanas. Sem cache, cada rodada local levaria minutos e gastaria o plano à toa. Vira tabela no Postgres na etapa 3.

## Decisão

Cache das respostas da IA em arquivo, com chave de modelo + prompt + perfil + vaga.

## Consequência

Migrada do antigo `docs/DECISIONS.md`, que guardava a decisão e o porquê; o efeito está descrito acima.
