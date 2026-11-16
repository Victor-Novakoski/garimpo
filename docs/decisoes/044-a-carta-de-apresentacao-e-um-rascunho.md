# 044. A carta de apresentação é um rascunho, escrito na hora e não guardado

- **Data:** 2026-11-16
- **Status:** ativa

## Contexto

Depois de decidir se candidatar, a pessoa ainda escreve uma carta para cada vaga. O Garimpo já tem o que a carta precisa: o perfil, o texto da vaga e o que a IA apontou (a favor, o que falta, o que destacar). Mas uma carta é texto que vai para fora, com o nome da pessoa, e o Garimpo nunca se candidata sozinho (decisão 004).

## Decisão

- `POST /jobs/{number}/letter` devolve um rascunho em texto puro. O pedido não leva nada: o servidor monta tudo a partir do perfil, da vaga e das linhas de motivo já gravadas.
- A carta passa pelo `internal/ai`, numa segunda interface (`Writer`) que os modos `local` e `api` implementam com as mesmas travas da nota: texto da vaga e leitura da IA marcados como dado, `claude -p` sem ferramentas, saída estruturada e validada.
- O prompt só deixa afirmar sobre a pessoa o que está no perfil.
- O rascunho não é gravado no banco nem em cache, e nada é enviado a lugar nenhum: a pessoa copia, ajusta e envia.
- Só existe no painel iniciado com `-ai`, uma carta por vez. A `garimpo-api` avulsa não escreve carta.

## Consequência

Sem tabela nem migração nova. Fechar a tela perde o rascunho; gerar de novo custa outra chamada ao modelo e pode dar outro texto. A carta fica tão boa quanto o `about` do perfil: com ele vazio, sai genérica. Vaga guardada só no arquivo local (sem banco) não tem a descrição, e a carta parte do título, da empresa e da leitura da IA.
