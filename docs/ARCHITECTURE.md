# Arquitetura

Um job em Go roda uma vez por dia no GitHub Actions, coleta, dá nota e grava no Postgres. O painel em React lê pela API em Go. Nada fica ligado o tempo todo, por isso tudo cabe nos planos grátis.

## Fluxo

1. **Coleta**: cada fonte tem um coletor (`internal/collector`) que devolve vagas novas no formato comum (`internal/job`).
2. **Normalização**: textos limpos, campos padronizados (modelo de trabalho, contrato, nível) e duplicadas entre fontes unidas.
3. **Filtro duro**: descarta o que não serve. Nada aqui depende de IA.
4. **Nota determinística (0 a 70)**: stack exigida e desejável com peso por skill, remoto, contrato, nível, faixa salarial quando houver. Cada ponto guarda o motivo.
5. **Nota da IA (0 a 30)**: compara a vaga com o perfil e devolve uma classe fixa, mais pontos fortes, lacunas e o que destacar. Só roda para o que passou no filtro.
6. **Gravação**: o registro de cada vaga (estado, anotação, nota e motivos da última rodada) e o texto dela no Postgres, ou num arquivo local quando não há banco configurado.
7. **Aviso**: resumo das melhores no Telegram.
8. **Painel**: lista por nota, detalhe com motivos e o funil (nova, vou aplicar, apliquei, entrevista, descartada).

## Cortes da nota

| Nota | O que acontece |
| --- | --- |
| abaixo de 55 | some da lista |
| 55 a 69 | talvez |
| 70 a 84 | boa |
| 85 ou mais | prioridade |

Os cortes são o ponto de partida e mudam depois de duas semanas de uso.

## Peças

| Parte | Escolha |
| --- | --- |
| Coleta, nota e API | Go, biblioteca padrão (o roteador é o `net/http`) |
| Banco | Postgres no Supabase, SQL puro com migrações, pelo driver `pgx` |
| IA | Interface `Scorer`: `claude -p` local ou API do Claude pelo SDK oficial |
| Painel | React + Vite + TypeScript na Vercel |
| Agendamento | GitHub Actions, cron diário |
| Acesso | Só o dono: hoje um token fixo no cabeçalho `Authorization`; o login do painel (Supabase Auth) entra na etapa 4 |

## Banco e API

O que a pessoa fez com cada vaga fica atrás de uma interface (`tracker.Store`) com duas implementações: um arquivo local e o Postgres. As regras do [FLOW.md](FLOW.md) estão escritas uma vez só (`internal/tracker/rules.go`) e as duas implementações passam pelos mesmos testes. A CLI e a API escolhem pela configuração: com `GARIMPO_DATABASE_URL` definida é o banco, sem ela é o arquivo.

| Tabela | O que guarda |
| --- | --- |
| `entries` | uma linha por vaga: o número fixo, o estado, a anotação, a nota e os motivos da última rodada, primeira e última vez vista, se saiu da fonte |
| `entry_keys` | as chaves (`fonte:id`) da vaga e das cópias dela em outras fontes |
| `entry_events` | o histórico: cada troca de estado, com data |
| `jobs` | o texto da vaga como veio da última coleta, para a tela de detalhe |
| `runs` | as rodadas: quando, quantas vagas listadas, quantas novas, quais fontes responderam |
| `schema_migrations` | quais migrações já foram aplicadas |

As migrações são arquivos SQL numerados em `internal/db/migrations/`, embutidos no binário e aplicados pelo `garimpo migrate`, cada um numa transação. Uma rodada grava tudo numa transação e num lote só, e quem escreve pega uma trava do banco antes, para dois processos nunca darem o mesmo número a vagas diferentes.

A API (`cmd/garimpo-api`, rotas em `internal/api`) não coleta e não dá nota: lê e escreve o mesmo registro. A nota que ela mostra é a da última rodada. Erro do banco vai para o log do servidor e nunca para a resposta.

Segurança do acesso:

- **Token do dono** em `GARIMPO_API_TOKEN`, com pelo menos 32 caracteres, comparado em tempo constante. Sem ele toda rota responde 401, exista ou não.
- **RLS ligado sem policy** em todas as tabelas: a API REST que o Supabase publica para o schema `public` não enxerga nada. O Garimpo entra como dono, pela conexão do Postgres.
- **A URL do banco só no ambiente.** Nunca em flag (o histórico do terminal e a lista de processos mostram) e nunca em mensagem de erro.
- **A API escuta só em `127.0.0.1`** por padrão e não fala HTTPS; para sair da máquina precisa de um proxy com HTTPS na frente.

## Perfil

O perfil (skills com pontos, o que aceitar, o que preferir, o que evitar) fica num `profile.yaml` fora do repositório. No Actions ele entra por segredo. O formato está em [`profile.example.yaml`](../profile.example.yaml); campo desconhecido é erro, para um erro de digitação nunca desligar uma regra calado.

## Nota determinística (0 a 70)

| Parte | Pontos | Como |
| --- | --- | --- |
| Tecnologias | até 40 | pontos de cada skill do perfil: cheios se está no título, nas etiquetas ou na seção de requisitos da descrição; metade se só no resto da descrição (muita empresa cola o mesmo parágrafo de stack em toda vaga). Links não contam |
| Modelo | 10 / 4 / 2 / 0 | preferido / aceito / não informado (baixo de propósito: vaga sem marcação muitas vezes é presencial) / não informado mas cita uma cidade no título ou no local |
| Contrato | 10 / 4 / 5 | preferido / aceito / não informado |
| Nível | 10 / 6 / 0 / 5 | igual / um acima ou abaixo / distante / não informado. Staff, principal e head são `lead`, um nível à parte |

Vaga sem nenhuma skill do perfil sai no filtro: é vaga de vendas, dados ou segurança que passou pelo resto das regras.

Vaga sem a informação passa no filtro e ganha poucos pontos daquela parte: melhor ver e descartar do que perder uma vaga boa mal etiquetada.

## Nota da IA (0 a 30)

A IA entra depois do filtro e da nota das regras, só nas melhores vagas (20 por padrão). Ela recebe o perfil e a vaga inteira e devolve um JSON com uma de cinco classes, um resumo, pontos a favor, lacunas e o que destacar na candidatura.

| Classe | Pontos | Quando |
| --- | --- | --- |
| forte | 30 | os requisitos principais batem com a experiência central e nenhuma restrição atrapalha |
| bom | 22 | a maior parte bate; falta algo que se aprende rápido |
| parcial | 14 | há base em comum, mas falta um requisito importante |
| fraco | 6 | pouca coisa em comum |
| nenhum | 0 | outra área, ou uma restrição que o perfil não aceita |

A IA escolhe a classe e o código dá os pontos. Assim ela nunca passa do teto, e uma resposta fora do formato (classe desconhecida, campo a mais, texto solto) é descartada: a vaga fica com a nota das regras e o erro aparece no `-why`.

A interface é uma só (`ai.Scorer`), com duas implementações que usam o mesmo prompt e o mesmo schema:

| Modo | Como | Quem paga |
| --- | --- | --- |
| `-ai local` | `claude -p` com `--json-schema`, sem ferramentas, sem MCP, sem arquivos do projeto, numa pasta vazia | o plano do Claude já logado na máquina |
| `-ai api` | SDK oficial em Go, saída estruturada, só fala com `api.anthropic.com` | `ANTHROPIC_API_KEY` (Haiku: centavos por mês) |

O texto da vaga vem da internet, então vai marcado como dado e o prompt manda ignorar instruções dentro dele. Mesmo que uma vaga engane o modelo, o máximo que ela consegue é a própria classe: 30 pontos, numa vaga que já passou no filtro.

As respostas ficam em cache em disco. A chave junta modelo, prompt, perfil e vaga: mudou qualquer um, a vaga é avaliada de novo. Erro não entra no cache.
