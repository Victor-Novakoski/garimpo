# 039. Sem token no painel local

- **Data:** 2026-11-11
- **Status:** ativa

## Contexto

Escutar no loopback tira as outras máquinas, mas não o navegador: qualquer site aberto consegue chamar `http://127.0.0.1`. Conferir de onde o pedido veio barra outro site e o truque de apontar um domínio para o loopback, sem o Victor ter que colar token. A `garimpo-api` com token continua existindo.

## Decisão

Sem token no painel local: o servidor só escuta em `127.0.0.1` e só atende a própria página (confere `Host`, `Origin` e `Sec-Fetch-Site`).

## Consequência

Migrada do antigo `docs/DECISIONS.md`, que guardava a decisão e o porquê; o efeito está descrito acima.
