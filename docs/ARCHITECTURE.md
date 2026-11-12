# Arquitetura

O Garimpo roda na máquina de quem procura a vaga. A coleta em Go busca as fontes, dá nota e grava no Postgres (Supabase); o painel em React lê e marca pela API em Go, servido pelo mesmo comando, só em `127.0.0.1`. Nada fica hospedado nem ligado o tempo todo: o único serviço de fora é o banco, no plano grátis. A coleta roda quando a pessoa pede, pela CLI ou pelo botão do painel.

## O caminho da vaga

1. **Coleta**: cada fonte tem um coletor (`internal/collector`) que devolve vagas no formato comum (`internal/job`), sempre pelo cliente HTTP educado (`internal/httpx`).
2. **Normalização** (`internal/normalize`): textos limpos, modelo de trabalho, contrato e nível padronizados, duplicadas entre fontes unidas.
3. **Filtro duro** (`internal/score`): descarta o que não serve. Nada aqui depende de IA.
4. **Nota das regras, 0 a 70** (`internal/score`): stack com peso por skill, modelo, contrato e nível. Cada ponto guarda o motivo.
5. **Nota da IA, 0 a 30** (`internal/ai`): compara a vaga com o perfil e devolve uma classe fixa, mais pontos fortes, lacunas e o que destacar. Só para as melhores que passaram no filtro.
6. **Registro** (`internal/tracker`): o estado, a anotação, a nota e os motivos da última rodada, no Postgres ou num arquivo local quando não há banco.
7. **Painel** (`web/`, servido por `garimpo serve`): lista do dia, tela da vaga e funil. Os estados estão no [FLOW.md](FLOW.md).

Ainda não existe: o job diário e o aviso no Telegram (adiados, ver decisão 020).

## Cortes da nota

| Nota | O que acontece |
| --- | --- |
| abaixo de 55 | some da lista |
| 55 a 69 | talvez |
| 70 a 84 | boa |
| 85 ou mais | prioridade |

Os cortes são o ponto de partida e mudam depois de duas semanas de uso. Como cada parte soma pontos está em `internal/score/CLAUDE.md` e `internal/ai/CLAUDE.md`.

## Peças

| Parte | Escolha |
| --- | --- |
| Coleta, nota e API | Go, biblioteca padrão (o roteador é o `net/http`) |
| Banco | Postgres no Supabase, SQL puro com migrações, pelo driver `pgx` |
| IA | Interface `Scorer`: `claude -p` local ou API do Claude pelo SDK oficial |
| Painel | React + Vite + TypeScript, embutido no binário e servido pelo `garimpo serve` em `127.0.0.1` |
| Agendamento | Na mão, pelo botão do painel ou pelo `collect` |
| Acesso | Só o dono, na própria máquina. A `garimpo-api` avulsa pede um token fixo (ver [SECURITY.md](SECURITY.md)) |

## Banco e API

O registro fica atrás de uma interface (`tracker.Store`) com duas implementações, um arquivo local e o Postgres, que aplicam as mesmas regras e passam pelos mesmos testes. Com `GARIMPO_DATABASE_URL` definida é o banco; sem ela, o arquivo.

| Tabela | O que guarda |
| --- | --- |
| `entries` | uma linha por vaga: o número fixo, o estado, a anotação, a nota e os motivos da última rodada, modelo, contrato, nível, local, primeira e última vez vista, se saiu da fonte |
| `entry_keys` | as chaves (`fonte:id`) da vaga e das cópias dela em outras fontes |
| `entry_events` | o histórico: cada troca de estado, com data |
| `jobs` | o texto da vaga como veio da última coleta, para a tela de detalhe |
| `runs` | as rodadas: quando, quantas vagas listadas, quantas novas, quais fontes responderam |
| `schema_migrations` | quais migrações já foram aplicadas |

A API (`internal/api`) lê e escreve o mesmo registro e sobe de dois jeitos: dentro do `garimpo serve`, sem token e só para o painel local, ou avulsa (`cmd/garimpo-api`), com o token do dono. As rotas estão no [README](../README.md#api-avulsa) e no [FLOW.md](FLOW.md).

## Perfil

O perfil (skills com pontos, o que aceitar, preferir e evitar, e o `about` que a IA lê) fica num `profile.yaml` fora do repositório. O formato está em [`profile.example.yaml`](../profile.example.yaml); campo desconhecido é erro, para um erro de digitação nunca desligar uma regra calado.
