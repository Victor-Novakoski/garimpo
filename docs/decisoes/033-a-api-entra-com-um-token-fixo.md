# 033. A API entra com um token fixo do dono (`GARIMPO_API_TOKEN`)

- **Data:** 2026-11-11
- **Status:** ativa (o login com Supabase Auth saiu do plano: ver 041)

## Contexto

Ainda não existe painel para fazer login e o Victor roda tudo local. O token funciona hoje sem depender de nada, e a checagem fica atrás de uma interface (`api.Authenticator`) para o login entrar sem mexer nas rotas.

## Decisão

A API entra com um token fixo do dono (`GARIMPO_API_TOKEN`); o Supabase Auth fica para a etapa 4.

## Consequência

Migrada do antigo `docs/DECISIONS.md`, que guardava a decisão e o porquê; o efeito está descrito acima.
