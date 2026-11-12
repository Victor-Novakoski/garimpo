# internal/collector

A interface dos coletores e um pacote por fonte (`githubissues`, `greenhouse`, `lever`, `gupy`). Cada um devolve vagas no formato de `internal/job`.

## Regras

- **Fonte nova passa antes pelo [`docs/SOURCES.md`](../../docs/SOURCES.md)**: acesso, termos de uso e risco. Fonte que proíbe coleta automatizada não entra.
- Toda chamada de rede vai pelo `internal/httpx`, com os hosts da fonte fixos no coletor.
- **Gupy só para uso pessoal:** desligada por padrão (flag `-gupy`), guarda o mínimo, sempre aponta para o link original e nunca aparece no demo nem em nada publicado. Para no primeiro erro de formato.
- Empresa nova no Greenhouse ou no Lever: confira na própria API se há vaga aberta no Brasil e acrescente uma linha em `greenhouse.Boards` ou `lever.Sites`.
- Testes sem rede: respostas salvas no `testdata/` de cada pacote.

## Armadilhas conhecidas

- No Lever, o campo `country` é confiável: vaga de outro país sai já na coleta, e o modelo de trabalho vem do campo da API. No Greenhouse não existe esse campo, então o país é filtrado pelo local, no `score`.
- Nome de board igual não é a mesma empresa: `flash`, `juno`, `cortex`, `blip` e `remote` existem no Greenhouse e não são as brasileiras.
- A Gupy mudou de formato antes (a API antiga dá 404); os dados vêm do `__NEXT_DATA__` da página de busca.
- Sem `GITHUB_TOKEN` a API do GitHub limita a 60 pedidos por hora.

## Ver também

Decisões 005, 008, 010, 023 e 024.
