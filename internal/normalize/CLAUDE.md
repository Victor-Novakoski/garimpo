# internal/normalize

Limpa o texto, classifica modelo de trabalho, contrato e nível, e une duplicadas entre fontes.

## Regras

- Sem dependência externa: os acentos saem por uma tabela própria (o `golang.org/x/text` atual exige Go 1.26).
- O título vence a etiqueta e o local quando dizem coisas diferentes.
- A descrição só entra para o modelo de trabalho, por último e só por frase inteira (`workModePhrases`): "remote-first", "100% presencial", "trabalho presencial". Palavra solta na descrição não conta. Contrato e nível continuam sem ler a descrição.
- Frase nova só entra depois de vista numa vaga real, com um caso no teste. Para quem só aceita remoto, um "presencial" lido errado tira a vaga no filtro.
- Faixa "Pleno/Sênior" vira Pleno. Staff, principal e head são `lead`, um nível à parte.
- Duplicadas: a mesma empresa com títulos parecidos é uma vaga só, mesmo vinda de repositórios diferentes.

## Armadilhas conhecidas

- Issues dos repositórios de vagas trazem etiquetas contraditórias ("Remoto" e "Híbrido" juntas).
- Empresa cola o mesmo parágrafo de modelo em toda vaga (XP: "Presencial Flexível"). Por isso "100% presencial", que vem nos requisitos da vaga, é conferido antes dele.
- "Head of Engineering" passava como sênior antes do nível `lead`.

## Ver também

Decisões 009, 011, 012 e 043.
