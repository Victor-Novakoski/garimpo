# 036. Modelo, contrato, nível e local passam a ficar também na entrada do…

- **Data:** 2026-11-11
- **Status:** ativa

## Contexto

A lista precisa mostrar o modelo de cada vaga, e ele só existia no texto guardado no banco: o arquivo local não tinha, e ler vaga por vaga daria uma consulta por linha.

## Decisão

Modelo, contrato, nível e local passam a ficar também na entrada do registro (migração 0002).

## Consequência

Migrada do antigo `docs/DECISIONS.md`, que guardava a decisão e o porquê; o efeito está descrito acima.
