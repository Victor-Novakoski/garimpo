# internal/normalize

Limpa o texto, classifica modelo de trabalho, contrato e nível, e une duplicadas entre fontes.

## Regras

- Sem dependência externa: os acentos saem por uma tabela própria (o `golang.org/x/text` atual exige Go 1.26).
- O título vence a etiqueta e o local quando dizem coisas diferentes.
- Faixa "Pleno/Sênior" vira Pleno. Staff, principal e head são `lead`, um nível à parte.
- Duplicadas: a mesma empresa com títulos parecidos é uma vaga só, mesmo vinda de repositórios diferentes.

## Armadilhas conhecidas

- Issues dos repositórios de vagas trazem etiquetas contraditórias ("Remoto" e "Híbrido" juntas).
- "Head of Engineering" passava como sênior antes do nível `lead`.

## Ver também

Decisões 009, 011 e 012.
