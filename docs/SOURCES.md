# Fontes de vagas

Toda fonte passa por aqui antes de virar coletor: como acessar, o que os termos dizem e o risco. Fonte que proíbe coleta automatizada não entra.

| Fonte | Acesso | Status | Risco |
| --- | --- | --- | --- |
| [backend-br/vagas](https://github.com/backend-br/vagas) | Issues pela API REST do GitHub | Aprovada | Baixo: API oficial, vagas publicadas para serem lidas |
| [frontendbr/vagas](https://github.com/frontendbr/vagas) | Issues pela API REST do GitHub | Aprovada | Baixo: mesmo caso |
| Gupy | Busca pública que o portal de vagas usa (`portal.api.gupy.io`) | A verificar | Médio: não é API documentada, pode mudar; ler os termos de uso |
| Greenhouse | Job Board API pública (`boards-api.greenhouse.io`), por empresa | Aprovada, etapa 5 | Baixo: API oficial e pública |
| Lever | Postings API pública (`api.lever.co/v0/postings`), por empresa | Aprovada, etapa 5 | Baixo: API oficial e pública |
| Sólides, Inhire e outros ATS brasileiros | A pesquisar | A verificar | A verificar |
| LinkedIn | Não entra | Recusada | Proíbe scraping e bloqueia conta |

## O que falta verificar

- [ ] **Gupy**: formato da resposta da busca, limite de uso e o que os termos dizem sobre acesso automatizado. Se os termos proibirem, a Gupy sai e o peso vai para as outras fontes.
- [ ] **Sólides e Inhire**: se têm página ou API pública de vagas e o que os termos permitem.
- [ ] **GitHub**: limite de 60 pedidos por hora sem token; no Actions usar o `GITHUB_TOKEN`.

## Regras de coleta

- Só GET, em hosts fixos por coletor.
- Timeout por pedido e limite de tamanho da resposta.
- Intervalo entre pedidos à mesma fonte.
- `User-Agent` que identifica o projeto e aponta para este repositório.
- Uma execução por dia.
