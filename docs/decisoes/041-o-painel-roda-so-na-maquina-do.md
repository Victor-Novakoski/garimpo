# 041. O painel roda só na máquina do Victor

- **Data:** 2026-11-11
- **Status:** ativa

## Contexto

Só ele usa, e todo dia no próprio PC. Hospedar pedia login, HTTPS, CORS e um servidor para cuidar, para proteger dados que não precisam sair da máquina. A coleta com IA local (`claude -p`) também só existe ali.

## Decisão

O painel roda só na máquina do Victor: `garimpo serve` sobe a API e as telas em `127.0.0.1`. Sem hospedar a API, sem Vercel e sem login.

## Consequência

Migrada do antigo `docs/DECISIONS.md`, que guardava a decisão e o porquê; o efeito está descrito acima.
