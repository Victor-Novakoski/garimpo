# 030. As regras do fluxo ficam num lugar só (`tracker/rules.go`)

- **Data:** 2026-11-11
- **Status:** ativa

## Contexto

Duas implementações das mesmas regras, uma em Go e outra em SQL, iam divergir com o tempo. Uma busca de emprego tem algumas centenas de vagas: carregar tudo numa rodada custa pouco.

## Decisão

As regras do fluxo ficam num lugar só (`tracker/rules.go`); arquivo e banco carregam as entradas, aplicam a mesma função e gravam o que mudou.

## Consequência

Migrada do antigo `docs/DECISIONS.md`, que guardava a decisão e o porquê; o efeito está descrito acima.
