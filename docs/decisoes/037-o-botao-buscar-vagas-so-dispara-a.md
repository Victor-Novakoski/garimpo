# 037. O botão "Buscar vagas" só dispara a coleta que o `serve` recebeu por…

- **Data:** 2026-11-11
- **Status:** ativa

## Contexto

A página não escolhe fonte, perfil nem IA: o máximo que um pedido indevido conseguiria é rodar a mesma coleta que o Victor já roda. Uma de cada vez.

## Decisão

O botão "Buscar vagas" só dispara a coleta que o `serve` recebeu por flag; o pedido HTTP não leva opção nenhuma.

## Consequência

Migrada do antigo `docs/DECISIONS.md`, que guardava a decisão e o porquê; o efeito está descrito acima.
