# internal/db

Conexão com o Postgres (Supabase) e as migrações SQL, embutidas no binário e aplicadas pelo `garimpo migrate`.

## Regras

- **Migração aplicada não se edita.** Mudança no banco é um arquivo novo em `migrations/`, com o número seguinte. Cada uma roda numa transação.
- **Tabela nova nasce com RLS ligado**, na mesma migração que a cria, e sem policy. O Garimpo entra como dono e não passa pelo RLS.
- A URL vem só de `GARIMPO_DATABASE_URL` e nunca aparece em mensagem de erro.
- Quem escreve pega `pg_advisory_xact_lock` antes, para dois processos nunca darem o mesmo número a vagas diferentes. Uma rodada grava tudo numa transação.

## Armadilhas conhecidas

- O pooler do Supabase divide as conexões: comando preparado não sobrevive de uma consulta para a outra. Por isso o `pgx` roda em `QueryExecModeExec` (não prepara). Não troque.
- Use a URL do **Session pooler** do Supabase, que funciona em rede IPv4.
- Depois de atualizar o projeto, rode o `migrate` de novo: os outros comandos avisam e esperam por ele.

## Testes

Os testes do banco usam `dbtest`, que só roda com `GARIMPO_TEST_DATABASE_URL` (o Postgres descartável do `ci.yml`, ou um local). Cada teste ganha um schema próprio e apaga no fim. Nunca aponte essa variável para o Supabase.

## Ver também

Decisões 025, 028, 029, 031 e 036.
