# internal/score

Filtro duro e a nota das regras, de 0 a 70, com um motivo para cada ponto.

## Regras

- **O filtro roda antes de tudo e não depende de IA.** A IA não consegue tirar uma vaga dele nem trazer uma de volta.
- Todo ponto tem um motivo legível (é o que o `-why` e a tela da vaga mostram).
- Vaga sem a informação passa no filtro e ganha poucos pontos daquela parte: melhor ver e descartar do que perder uma vaga boa mal etiquetada.
- Vaga sem nenhuma skill do perfil sai no filtro (é vaga de vendas, dados ou segurança que passou pelo resto).

## A nota

| Parte | Pontos | Como |
| --- | --- | --- |
| Tecnologias | até 40 | pontos de cada skill do perfil: cheios se está no título, nas etiquetas ou na seção de requisitos; metade se só no resto da descrição. Links não contam |
| Modelo | 10 / 4 / 2 / 0 | preferido / aceito / não informado / não informado mas cita uma cidade no título ou no local |
| Contrato | 10 / 4 / 5 | preferido / aceito / não informado |
| Nível | 10 / 6 / 0 / 5 | igual / um acima ou abaixo / distante / não informado |

Os cortes (55, 70, 85) estão no [ARCHITECTURE.md](../../docs/ARCHITECTURE.md#cortes-da-nota).

## Armadilhas conhecidas

- Muita empresa cola o mesmo parágrafo de stack em toda vaga: por isso skill só na descrição vale metade.
- "go-to-market" não é Go e "React Native" não é React. Um `utm_medium=react-brasil-vagas` num link já deu React a uma vaga: links não contam.
- Modelo "não informado" vale pouco de propósito: vaga sem marcação muitas vezes é presencial.
- O Greenhouse traz vagas do México, da Argentina e do Chile: o país é conferido aqui.

## Ver também

Decisões 006, 012, 013 e 014.
