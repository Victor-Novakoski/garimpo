# 043. Modelo de trabalho lido da descrição, só por frase inteira

- **Data:** 2026-11-16
- **Status:** ativa

## Contexto

As vagas do Greenhouse de QuintoAndar, Wildlife, BTG, Inter, C6 e XP chegavam sem modelo de trabalho e ganhavam 2 ou 0 de 10 nessa parte da nota (issue #18). A API não tem um campo padrão para isso. Conferindo as seis em 16/11/2026: a Wildlife usa um campo próprio em `metadata` ("Work Model"); o QuintoAndar e a XP dizem o modelo num parágrafo da descrição, igual em toda vaga ("remote-first", "Presencial Flexível"); o Inter e o C6 dizem numa linha dos requisitos, só em parte das vagas ("Disponibilidade para trabalho presencial", "modelo 100% presencial"); o BTG não diz.

Até aqui a descrição não era lida para classificar, porque palavras soltas nela citam outros modelos e níveis ("reuniões presenciais", "equipes remotas").

## Decisão

- O coletor do Greenhouse lê o modelo do campo próprio da empresa quando o nome do campo é de modelo de trabalho ("Work Model", "Modelo de Trabalho").
- O `normalize` passa a ler a descrição para o modelo de trabalho, por último (depois de título, local e etiquetas) e só por frase inteira, de uma lista curta. Vale para todas as fontes.
- Quando a vaga diz mais de uma coisa: "100% presencial" vence, depois híbrido, depois remoto, depois as outras frases de presencial. "Presencial Flexível" da XP conta como híbrido.
- Contrato e nível continuam sem ler a descrição.
- Vaga que não diz o modelo em lugar nenhum continua "não informado". Nada de chutar pelo nome da empresa.

## Consequência

As vagas de tecnologia do QuintoAndar passam a remoto e ganham os 10 pontos. As da Wildlife, da XP e as do Inter e do C6 que pedem presencial passam a híbrido ou presencial, e para um perfil que só aceita remoto saem no filtro, com o motivo, em vez de ficar na lista com nota baixa. BTG e parte do Inter e do C6 seguem sem modelo.

Um "presencial" lido errado tira uma vaga boa do filtro, então frase nova só entra depois de vista numa vaga real e com caso no teste (`internal/collector/greenhouse/testdata/work_mode.json`).
