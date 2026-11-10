# Fontes de vagas

Toda fonte passa por aqui antes de virar coletor: como acessar, o que os termos dizem e o risco. Fonte que proíbe coleta automatizada não entra; fonte cujos termos proíbem agregar o conteúdo só entra como uso pessoal, desligada por padrão (caso da Gupy, abaixo).

| Fonte | Acesso | Status | Risco |
| --- | --- | --- | --- |
| [backend-br/vagas](https://github.com/backend-br/vagas) | Issues pela API REST do GitHub | Aprovada | Baixo: API oficial, vagas publicadas para serem lidas |
| [frontendbr/vagas](https://github.com/frontendbr/vagas) | Issues pela API REST do GitHub | Aprovada | Baixo: mesmo caso |
| [react-brasil/vagas](https://github.com/react-brasil/vagas) | Issues pela API REST do GitHub | Aprovada | Baixo: mesmo caso |
| Gupy | Dados embutidos na página de busca (`portal.gupy.io/job-search`, `__NEXT_DATA__`) | Só uso pessoal, desligada por padrão | Alto: os termos proíbem "agregar, copiar ou duplicar"; a API antiga já dá 404 |
| Greenhouse | Job Board API pública (`boards-api.greenhouse.io`), por empresa: Stone, QuintoAndar, Wildlife Studios | Aprovada | Baixo: API oficial e pública. Muitas vagas fora de tech; o filtro da etapa 2 cuida disso |
| Lever | Postings API pública (`api.lever.co/v0/postings`), por empresa | Aprovada, etapa 5 | Baixo: API oficial e pública |
| Sólides | Dados embutidos na página (`vagas.solides.com.br`, Next.js); API pública não encontrada | A verificar | robots.txt libera tudo; só há termos B2B; a busca devolveu vagas fora de tech |
| Inhire | A pesquisar | A verificar | robots.txt libera tudo; termos não lidos |
| LinkedIn | Não entra | Recusada | Proíbe scraping e bloqueia conta |

## Gupy: regras do uso pessoal

Verificado em 10/11/2026. Os [termos de uso da pessoa candidata](https://www.gupy.io/hubfs/%5BJur%C3%ADdico%5D%20Time%20de%20Contratos/Termos%20de%20Uso%20do%20Gupy%20Recrutamento%20e%20Sele%C3%A7%C3%A3o%20-%20Pessoa%20Candidata%2020.03.2024.pdf) (20/03/2024) proíbem "agregar, copiar ou duplicar partes do Gupy Recrutamento e Seleção". A API pública da Gupy é para empresas clientes, não para candidatos. O `robots.txt` de `portal.gupy.io` não bloqueia nada.

Por isso o coletor da Gupy:

- vem **desligado por padrão** e só liga por configuração local;
- guarda o mínimo para dar nota e **sempre aponta para o link original** da vaga;
- **nunca aparece no demo público** nem em nada publicado;
- faz poucos pedidos por execução, com intervalo, e para ao primeiro erro de formato (a página muda sem aviso).

Formato observado: a busca devolve até 12 vagas em `props.pageProps.initialJobList` (`data` + `pagination`), com `id`, `name`, `careerPageName`, `workplaceType`, `type`, `publishedDate`, `applicationDeadline`, `description` e `jobUrl`.

## O que falta verificar

- [ ] **Sólides e Inhire**: se têm API pública de vagas e o que os termos de candidato dizem.
- [ ] **Greenhouse e Lever**: mais empresas brasileiras. Nubank tem board mas sem vagas; iFood, Creditas, CloudWalk e Loft não usam Greenhouse.
- [ ] **GitHub**: limite de 60 pedidos por hora sem token; no Actions usar o `GITHUB_TOKEN`.

## Regras de coleta

- Só GET, em hosts fixos por coletor.
- Timeout por pedido e limite de tamanho da resposta.
- Intervalo entre pedidos à mesma fonte.
- `User-Agent` que identifica o projeto e aponta para este repositório.
- Uma execução por dia.
