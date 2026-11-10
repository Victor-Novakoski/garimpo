# AGENTS.md

Guia para quem (pessoa ou IA) for mexer neste repositório. Leia antes de começar.

## O projeto

**Garimpo** acha todo dia as vagas brasileiras de tecnologia que combinam com um perfil e entrega as melhores com nota e motivo. Peneira muita vaga e fica só com o ouro.

É um projeto de portfólio do Victor Novakoski (dev full stack, Go + React), e também a ferramenta que ele usa para procurar o segundo emprego: pleno, remoto, PJ ou meio período, fora de logística.

- Arquitetura: [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md)
- Etapas e tarefas: [docs/ROADMAP.md](docs/ROADMAP.md)
- Decisões e o porquê: [docs/DECISIONS.md](docs/DECISIONS.md)
- Fontes de vagas: [docs/SOURCES.md](docs/SOURCES.md)

## Stack

- Go 1.24: coleta, nota e API (`cmd/`, `internal/`).
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
- **A IA não manda na regra.** O filtro duro roda antes e a IA não consegue tirar uma vaga dele. A nota da IA tem teto (30 de 100) e saída validada.
- **Nota explicável:** todo ponto da nota tem um motivo legível.
- **Dados pessoais fora do repositório.** Currículo e perfil reais nunca são commitados; o repositório traz só um perfil de exemplo.
- **Testes sem rede.** Coletores são testados com respostas salvas em `testdata/`.

## Estrutura

```
cmd/garimpo/          a CLI (garimpo collect)
internal/job/         o modelo de vaga
internal/httpx/       o cliente HTTP educado, único caminho para a rede
internal/normalize/   classificação (modelo, contrato, nível) e duplicadas
internal/collector/   a interface dos coletores e um pacote por fonte
docs/                 arquitetura, roadmap, decisões, fontes
```
