# internal/tracker

O que a pessoa fez com cada vaga: estados do funil, regras, a interface `Store` e as duas implementações (arquivo local e Postgres). O desenho está no [`docs/FLOW.md`](../../docs/FLOW.md).

## Regras

- **Regra do fluxo mora só em `rules.go`.** Arquivo e banco carregam as entradas, aplicam a mesma função e gravam o que mudou. Regra nova entra ali e no teste que roda nas duas implementações, nunca em uma só.
- A vaga só sai da lista quando a pessoa marca; aparecer numa rodada não conta como vista.
- A identidade é fonte + id na fonte (`job.Key`). O número curto (`#12`) é dado na primeira vez e nunca muda. Vaga repetida em duas fontes é uma só.
- A nota não é guardada como verdade: é recalculada a cada rodada. O que fica é estado, datas, anotação e os motivos da última rodada.
- "Saiu da fonte" só é marcado quando a fonte respondeu: fonte fora do ar não apaga vaga.
- Com `GARIMPO_DATABASE_URL` é o banco, sem ela o arquivo (`open.go`). A URL nunca vem de flag.

## Armadilhas conhecidas

- O `import` só roda num banco vazio e não mexe no arquivo: juntar dois registros daria o mesmo número a duas vagas.

## Ver também

Decisões 021, 022, 026, 027 e 030; `internal/db/CLAUDE.md`.
