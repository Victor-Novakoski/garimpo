# Arquitetura

Um job em Go roda uma vez por dia no GitHub Actions, coleta, dá nota e grava no Postgres. O painel em React lê pela API em Go. Nada fica ligado o tempo todo, por isso tudo cabe nos planos grátis.

## Fluxo

1. **Coleta**: cada fonte tem um coletor (`internal/collector`) que devolve vagas novas no formato comum (`internal/job`).
2. **Normalização**: textos limpos, campos padronizados (modelo de trabalho, contrato, nível) e duplicadas entre fontes unidas.
3. **Filtro duro**: descarta o que não serve. Nada aqui depende de IA.
4. **Nota determinística (0 a 70)**: stack exigida e desejável com peso por skill, remoto, contrato, nível, faixa salarial quando houver. Cada ponto guarda o motivo.
5. **Nota da IA (0 a 30)**: compara a vaga com o perfil e devolve uma classe fixa, mais pontos fortes, lacunas e o que destacar. Só roda para o que passou no filtro.
6. **Gravação**: vagas, notas e motivos no Postgres.
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
| Coleta, nota e API | Go, biblioteca padrão + `chi` na API |
| Banco | Postgres no Supabase, SQL puro com migrações |
| IA | Interface `Scorer`: API do Claude ou `claude -p` local |
| Painel | React + Vite + TypeScript na Vercel |
| Agendamento | GitHub Actions, cron diário |
| Acesso | Login só do dono (Supabase Auth); a API não fica aberta |

## Perfil

O perfil (skills com pontos, o que aceitar, o que preferir, o que evitar) fica num `profile.yaml` fora do repositório. No Actions ele entra por segredo. O formato está em [`profile.example.yaml`](../profile.example.yaml); campo desconhecido é erro, para um erro de digitação nunca desligar uma regra calado.

## Nota determinística (0 a 70)

| Parte | Pontos | Como |
| --- | --- | --- |
| Tecnologias | até 40 | pontos de cada skill do perfil: cheios se está no título ou nas etiquetas, metade se só na descrição (muita empresa cola o mesmo parágrafo de stack em toda vaga) |
| Modelo | 10 / 4 / 2 | preferido / aceito / não informado (baixo de propósito: vaga sem marcação muitas vezes é presencial) |
| Contrato | 10 / 4 / 5 | preferido / aceito / não informado |
| Nível | 10 / 6 / 0 / 5 | igual / um acima ou abaixo / distante / não informado. Staff, principal e head são `lead`, um nível à parte |

Vaga sem a informação passa no filtro e ganha poucos pontos daquela parte: melhor ver e descartar do que perder uma vaga boa mal etiquetada.
