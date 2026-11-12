# 038. O build do painel é embutido no binário (`embed`) e não vai para o git

- **Data:** 2026-11-11
- **Status:** ativa

## Contexto

Um comando só serve tudo, sem Node rodando no dia a dia. Build commitado fica velho e polui os diffs; a pasta guarda só um marcador, e sem o build o servidor explica como gerar.

## Decisão

O build do painel é embutido no binário (`embed`) e não vai para o git.

## Consequência

Migrada do antigo `docs/DECISIONS.md`, que guardava a decisão e o porquê; o efeito está descrito acima.
