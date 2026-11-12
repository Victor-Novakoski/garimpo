# web

O painel em React + Vite + TypeScript. Não tem deploy: o build vai para `internal/panel/dist` e o Go embute no binário.

## Regras

- **Texto de vaga é texto.** Nunca `dangerouslySetInnerHTML`. Link de vaga só passa por `safeUrl` (`http`/`https`).
- Sem biblioteca de rotas, de estado ou de componentes: as telas ficam no `#` do endereço (`router.ts`), e depois de cada mudança a tela lê de novo do servidor, que é quem sabe as regras.
- Todas as chamadas ficam em `api.ts`; textos e datas em `format.ts`.
- **O build não vai para o git.** `internal/panel/dist/` guarda só o `.gitkeep`.
- Mexeu aqui: `npm run lint`, `npm test` e `npm run build` passando. Testar a 390 px, claro e escuro.

## Armadilhas conhecidas

- Precisa de Node 22.22+ ou 24.15+. Um Node mais antigo pula calado os pacotes nativos do Vite e o build quebra depois sem dizer o motivo; o `.npmrc` com `engine-strict` faz o `npm ci` parar na hora.
- `npm run dev` sobe o Vite e repassa `/api` para um `garimpo serve` rodando ao lado.
- A leitura da IA na tela vem das linhas de motivo gravadas; linha que a tela não conhece aparece como está, nunca some.

## Ver também

Decisões 034, 035 e 038; [`docs/SECURITY.md`](../docs/SECURITY.md).
