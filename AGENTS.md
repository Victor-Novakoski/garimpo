# AGENTS.md

Guia para quem (pessoa ou IA) for mexer neste repositório. Leia antes de começar.

## O projeto

**Garimpo** acha todo dia as vagas brasileiras de tecnologia que combinam com um perfil e entrega as melhores com nota e motivo. Peneira muita vaga e fica só com o ouro.

É um projeto de portfólio do Victor Novakoski (dev full stack, Go + React), e também a ferramenta que ele usa para procurar o segundo emprego: pleno, remoto, PJ ou meio período, fora de logística.

- Arquitetura: [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md)
- Fluxo da vaga e estados do funil: [docs/FLOW.md](docs/FLOW.md). A CLI, a API e o painel seguem este desenho
- Etapas e tarefas: [docs/ROADMAP.md](docs/ROADMAP.md)
- Decisões e o porquê: [docs/DECISIONS.md](docs/DECISIONS.md)
- Fontes de vagas: [docs/SOURCES.md](docs/SOURCES.md)

## Stack

- Go 1.25: coleta, nota e API (`cmd/`, `internal/`). Dependências: `yaml.v3` e o SDK oficial da Anthropic.
- PostgreSQL no Supabase (a partir da etapa 3), SQL puro com migrações, sem ORM.
- React + Vite + TypeScript no `web/` (a partir da etapa 4), deploy na Vercel.
- GitHub Actions: CI e o job diário de coleta.

## Comandos

```bash
go test -race ./...   # testes
go vet ./...          # análise estática
gofmt -l .            # tem que sair vazio
go run ./cmd/garimpo collect          # coleta e lista as vagas
go run ./cmd/garimpo collect -gupy    # inclui a Gupy (só uso pessoal)
go run ./cmd/garimpo collect -ai local -why   # soma a nota da IA pelo claude -p
go run ./cmd/garimpo mark 12 apliquei         # marca o que foi feito com a vaga #12
```

## Regras do repositório

- Branch principal de trabalho: `develop`. Todo PR vai para a `develop`, nunca para a `main`.
- Commits com a identidade do Victor (`Victor-Novakoski`). Sem trailer `Co-Authored-By` de IA, sem link de sessão, sem assinatura de IA.
- Um commit por tarefa, com `gofmt`, `go vet` e testes passando entre eles.
- Textos para pessoas (docs, mensagens da CLI, painel) em português do Brasil. Código, nomes de arquivo e identificadores em inglês.
- Sem serviço pago além da IA. Hospedagem, banco e agendamento ficam nos planos grátis.

## Regras da ferramenta (não quebrar)

- **Nunca se candidata sozinha.** O Garimpo acha, filtra, dá nota e prepara. O clique final é sempre de uma pessoa.
- **Sem LinkedIn.** Nada de scraping de LinkedIn nem de site que proíba coleta automatizada. Cada fonte nova é conferida em [docs/SOURCES.md](docs/SOURCES.md) antes de entrar. Fonte cujos termos proíbem agregar (Gupy) só roda como uso pessoal: desligada por padrão, só link para a vaga original e nunca no demo público.
- **Coleta educada:** só GET, hosts fixos, timeout, limite de tamanho de resposta, intervalo entre pedidos e um `User-Agent` que identifica o projeto.
- **A IA não manda na regra.** O filtro duro roda antes e a IA não consegue tirar uma vaga dele nem trazer uma de volta. A IA só escolhe uma de cinco classes; os pontos de cada classe ficam no código, com teto de 30 de 100, e a saída é validada.
- **Texto de vaga é dado, não instrução.** Vai para a IA marcado como dado, e o modo local roda o `claude -p` sem nenhuma ferramenta, sem MCP e numa pasta vazia.
- **Nota explicável:** todo ponto da nota tem um motivo legível.
- **Dados pessoais fora do repositório.** Currículo e perfil reais nunca são commitados; o repositório traz só um perfil de exemplo.
- **A vaga só sai da lista quando a pessoa marca.** Os estados e as regras estão em [docs/FLOW.md](docs/FLOW.md) e valem igual para a CLI, a API e o painel.
- **Testes sem rede.** Coletores e IA são testados com respostas salvas em `testdata/`, sem chamar modelo nenhum.

## Estrutura

```
cmd/garimpo/          a CLI (collect, mark, note, list, show)
internal/job/         o modelo de vaga
internal/httpx/       o cliente HTTP educado, único caminho da coleta para a rede
internal/normalize/   classificação (modelo, contrato, nível) e duplicadas
internal/profile/     o perfil (o que aceitar, evitar e quanto vale cada skill)
internal/score/       filtro duro e nota de 0 a 70 com motivos
internal/ai/          nota da IA de 0 a 30: interface Scorer, claude -p local e API
internal/tracker/     o que a pessoa fez com cada vaga: estados do funil, interface Store e arquivo local
internal/collector/   a interface dos coletores e um pacote por fonte
docs/                 arquitetura, roadmap, decisões, fontes
```
