# 046. O perfil lê notas locais para a IA (`about_files`)

- **Data:** 2026-11-25
- **Status:** ativa

## Contexto

O dono mantém o currículo e as metas em notas Markdown fora do repositório. O `about` do `profile.yaml` era uma cópia manual dessas notas e ficava velho quando elas mudavam (issue #38). Gerar o perfil inteiro por IA a partir das notas foi descartado: mais peças, e a IA passaria a mexer nas regras duras.

## Decisão

O perfil aceita `about_files`, uma lista de caminhos locais (absolutos ou com `~/`). O `profile.Load` lê os arquivos junto com o YAML e guarda o texto à parte (`AboutTexts`), sem mexer no `about`. O `Parse` não toca no disco.

- O texto só entra no bloco do perfil que a IA lê (nota e carta), depois do `about`, um bloco por arquivo com o nome do arquivo. Filtro duro e nota das regras não leem esses arquivos.
- Caminho relativo é recusado: dependeria da pasta de onde o comando roda.
- Limites: 20 000 bytes por arquivo e 40 000 no total (perto de 5 mil e 10 mil tokens). Passar do limite é erro, não corte: um corte silencioso esconderia a parte do fim da nota, e o prompt é pago em toda vaga avaliada.
- Arquivo que falta, não é arquivo comum, não lê ou não é UTF-8 é erro que cita o caminho como foi escrito, nunca o conteúdo.

## Consequência

- Mudar uma nota muda o prompt de sistema, que é a chave do cache da IA: a próxima rodada reavalia as vagas.
- As notas vão para o modelo como o `about` já ia (plano local ou API). Elas não entram no repositório: só o `profile.example.yaml` mostra o campo, comentado e com caminho genérico.
- O modo demo continua usando só o `profile.example.yaml`, sem `about_files`.
