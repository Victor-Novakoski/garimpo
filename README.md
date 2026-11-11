# garimpo

Peneira as vagas de tecnologia do Brasil e fica só com o ouro.

O Garimpo coleta vagas de fontes abertas uma vez por dia, descarta o que não serve, dá a cada vaga uma nota explicável de 0 a 100 e entrega as melhores. Ele **não** se candidata sozinho: acha, filtra e prepara, e a decisão é sempre sua.

> Em construção. Veja o [roadmap](docs/ROADMAP.md).

## Como funciona

1. **Coleta** de fontes abertas: repositórios de vagas no GitHub e as páginas de vagas de empresas brasileiras no Greenhouse e no Lever. Sem LinkedIn.
2. **Filtro duro**: some o que não serve (presencial, nível errado, setor evitado, duplicada).
3. **Nota**: até 70 pontos calculados em código (stack, remoto, modelo de contrato, nível) e até 30 da IA, comparando a vaga com o currículo.
4. **Entrega**: resumo diário e um painel para acompanhar cada vaga até a entrevista.

Detalhes em [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md).

## Rodando

Precisa de Go 1.25 ou mais novo.

```bash
go run ./cmd/garimpo collect          # coleta e lista as vagas dos últimos 30 dias
go run ./cmd/garimpo collect -json    # uma vaga por linha, em JSON
go test ./...
```

Os testes do banco só rodam com `GARIMPO_TEST_DATABASE_URL` apontando para um Postgres de teste (a CI sobe um); sem ela são pulados, e as mesmas regras são testadas pelo arquivo.

Para filtrar e dar nota, copie `profile.example.yaml` para `profile.yaml` (fica fora do git) e ajuste. Com ele, o `collect` corta o que não serve e ordena pela nota:

```bash
go run ./cmd/garimpo collect -why     # mostra o motivo de cada ponto
go run ./cmd/garimpo collect -min 45  # só vagas com nota 45 ou mais
```

### Nota da IA

Os 30 pontos que faltam vêm de uma IA que lê a vaga inteira e compara com o seu perfil. Ela diz o encaixe (forte, bom, parcial, fraco ou nenhum), o que conta a favor, o que falta e o que destacar na candidatura. Preencha o campo `about` do `profile.yaml` com a sua experiência: é o que ela usa para comparar.

```bash
go run ./cmd/garimpo collect -ai local -why   # usa o Claude Code da máquina (claude -p) e o seu plano
go run ./cmd/garimpo collect -ai api -why     # usa a API do Claude, com ANTHROPIC_API_KEY
```

Por padrão a IA avalia as 20 melhores vagas pela nota das regras (`-ai-top`), com o modelo Haiku (`-ai-model` troca). Cada vaga é avaliada uma vez: a resposta fica em cache e a próxima rodada só avalia o que é novo. Se a IA falhar, a vaga continua na lista com a nota das regras.

### Acompanhando as vagas

Cada vaga da lista tem um número que nunca muda. Ela continua aparecendo em toda rodada até você dizer o que fez com ela:

```bash
go run ./cmd/garimpo mark 12 apliquei         # estados: nova, vou aplicar, apliquei, entrevista, descartada
go run ./cmd/garimpo note 12 "falei com a Ana do RH"
go run ./cmd/garimpo list                     # o que está em andamento: vou aplicar, apliquei, entrevista
go run ./cmd/garimpo show 12                  # a vaga, a anotação e o histórico
```

As que você marcou como "apliquei", "entrevista" ou "descartada" saem da lista do dia. As marcadas como "vou aplicar" sobem para o topo, e as que chegaram nesta rodada aparecem como "nova hoje". Tudo fica num arquivo na sua máquina, fora do repositório. O desenho completo está em [docs/FLOW.md](docs/FLOW.md).

Com `GITHUB_TOKEN` definido, a coleta do GitHub usa o limite maior da API.

### Guardando no banco

Por padrão o que você marca fica no arquivo local. Para guardar num Postgres (o plano é o Supabase, no plano grátis), ponha a URL de conexão na variável `GARIMPO_DATABASE_URL`. Ela leva a senha do banco, por isso vai só no ambiente: nunca num arquivo do repositório nem numa flag.

```bash
export GARIMPO_DATABASE_URL='postgres://usuario:senha@host:5432/postgres?sslmode=require'
go run ./cmd/garimpo migrate   # cria as tabelas; pode rodar de novo sem medo
go run ./cmd/garimpo import    # copia para o banco o que já estava marcado no arquivo
```

No Supabase, a URL está no botão **Connect** do projeto. Use a do **Session pooler**, que funciona em rede IPv4.

Com a variável definida, `collect`, `mark`, `note`, `list` e `show` passam a usar o banco, com os mesmos números e estados. O `import` só roda num banco vazio e não mexe no arquivo, que fica como cópia. Para voltar ao arquivo numa rodada, use `-state file`.

As tabelas nascem com RLS ligado e sem nenhuma policy: a API REST que o Supabase publica sozinho não lê nem escreve nada. Só o Garimpo entra, pela conexão do Postgres.

### API

A API serve o mesmo fluxo por HTTP, para o painel. Ela lê e escreve o registro (arquivo ou banco, pela mesma regra da CLI); quem coleta continua sendo o `collect`.

```bash
export GARIMPO_API_TOKEN="$(openssl rand -hex 32)"   # o token do dono, com 32 caracteres ou mais
go run ./cmd/garimpo-api                             # escuta em 127.0.0.1:8080

curl -H "Authorization: Bearer $GARIMPO_API_TOKEN" localhost:8080/jobs
```

| Rota | O que faz |
| --- | --- |
| `GET /jobs` | a lista do dia: "vou aplicar" no topo, depois as novas por nota |
| `GET /jobs?status=saved,applied,interview` | as vagas nesses estados, na ordem pedida (`all` traz todas) |
| `GET /jobs/12` | a vaga, os motivos da nota, a anotação, o histórico e, com o banco, o texto da vaga |
| `PATCH /jobs/12` | `{"status": "applied"}` marca, `{"note": "falei com a Ana"}` anota; podem vir juntos |
| `GET /health` | responde `ok`, sem token |

Toda rota, menos `/health`, exige o token. Os estados são `new`, `saved`, `applied`, `interview` e `discarded`; as palavras da CLI ("apliquei") também valem. A API não fala HTTPS: fora da sua máquina, só atrás de um proxy que fale.

## Por que existe

Candidatura em massa costuma render menos entrevista, não mais. O Garimpo aposta no contrário: poucas vagas, bem escolhidas, com o motivo da escolha à vista.

Feito por [Victor Novakoski](https://github.com/Victor-Novakoski).
