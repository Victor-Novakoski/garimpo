# garimpo

Peneira as vagas de tecnologia do Brasil e fica só com o ouro.

O Garimpo coleta vagas de fontes abertas quando você pede, descarta o que não serve, dá a cada vaga uma nota explicável de 0 a 100 e entrega as melhores. Ele **não** se candidata sozinho: acha, filtra e prepara, e a decisão é sempre sua.

> Em construção. O que falta está nas [issues](https://github.com/Victor-Novakoski/garimpo/issues).

## Como funciona

1. **Coleta** de fontes abertas: repositórios de vagas no GitHub e as páginas de vagas de empresas brasileiras no Greenhouse e no Lever. Sem LinkedIn.
2. **Filtro duro**: some o que não serve (presencial, nível errado, setor evitado, duplicada).
3. **Nota**: até 70 pontos calculados em código (stack, remoto, modelo de contrato, nível) e até 30 da IA, comparando a vaga com o currículo.
4. **Entrega**: um painel, na sua máquina, para ver a lista do dia e acompanhar cada vaga até a entrevista.

## Documentação

- [Arquitetura](docs/ARCHITECTURE.md): o caminho da vaga, as peças e o banco
- [Fluxo da vaga](docs/FLOW.md): os estados do funil e as três telas (CLI, API e painel)
- [Segurança](docs/SECURITY.md): o que protege o painel, o banco e a IA
- [Fontes](docs/SOURCES.md): de onde vêm as vagas e o que os termos de cada uma dizem
- [Decisões](docs/decisoes/): cada escolha e o porquê

## Rodando

Precisa de Go 1.25 ou mais novo.

```bash
go run ./cmd/garimpo collect          # coleta e lista as vagas dos últimos 30 dias
go run ./cmd/garimpo collect -json    # uma vaga por linha, em JSON
go test ./...
npm --prefix web test                 # os testes do painel
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

### Painel

O painel mostra a lista do dia, a tela de cada vaga (motivos da nota, leitura da IA, anotação e histórico) e o funil. Ele roda só na sua máquina: um comando sobe a API e as telas em `127.0.0.1`, sem login e sem nada hospedado.

Precisa de um Node recente, só para montar as telas: o 22 mais novo (22.22 em diante) ou o 24 (24.15 em diante). Com `nvm`, é `nvm install 22`. Com um Node mais antigo o `npm ci` para e avisa, em vez de instalar pela metade. Faça isso na primeira vez e sempre que atualizar o projeto:

```bash
npm --prefix web ci
npm --prefix web run build
```

Depois, no dia a dia, é um comando só:

```bash
go run ./cmd/garimpo serve              # abre em http://127.0.0.1:8080
go run ./cmd/garimpo serve -ai local    # o botão "Buscar vagas" também pede a nota da IA
go run ./cmd/garimpo serve -port 9000   # outra porta
```

- **Lista do dia**: nota, estado, empresa, modelo e link, com "vou aplicar" no topo. Um clique marca (vou aplicar, apliquei, descartar) e o aviso que aparece deixa desfazer.
- **Filtros**: por estado (vou aplicar, apliquei, entrevista, em andamento, descartadas, todas) e busca por vaga ou empresa.
- **Vaga**: por que a nota é aquela, o que a IA disse (encaixe, a favor, o que falta, o que destacar), anotação e histórico com datas.
- **Funil**: quantas vagas em cada estado e as últimas mudanças.
- **Buscar vagas**: roda a coleta dali mesmo, com as opções que você passou ao `serve` (as mesmas do `collect`: `-ai`, `-gupy`, `-days`...), e mostra o andamento. Precisa do `profile.yaml`.

O painel usa o mesmo registro da linha de comando: o banco, se `GARIMPO_DATABASE_URL` estiver definida, ou o arquivo local. O que você marca num aparece no outro.

O servidor só escuta em `127.0.0.1` e só atende a própria página: outro site aberto no navegador não consegue ler nem marcar nada. O Garimpo continua sem se candidatar por você: "Abrir a vaga" leva ao site da empresa, e marcar "apliquei" é você contando o que fez.

### Acompanhando as vagas pelo terminal

Cada vaga da lista tem um número que nunca muda. Ela continua aparecendo em toda rodada até você dizer o que fez com ela:

```bash
go run ./cmd/garimpo mark 12 apliquei         # estados: nova, vou aplicar, apliquei, entrevista, descartada
go run ./cmd/garimpo note 12 "falei com a Ana do RH"
go run ./cmd/garimpo list                     # o que está em andamento: vou aplicar, apliquei, entrevista
go run ./cmd/garimpo show 12                  # a vaga, a anotação e o histórico
```

As que você marcou como "apliquei", "entrevista" ou "descartada" saem da lista do dia. As marcadas como "vou aplicar" sobem para o topo, e as que chegaram nesta rodada aparecem como "nova hoje". Sem banco configurado, tudo fica num arquivo na sua máquina, fora do repositório. O desenho completo está em [docs/FLOW.md](docs/FLOW.md).

Com `GITHUB_TOKEN` definido, a coleta do GitHub usa o limite maior da API.

### Guardando no banco

Por padrão o que você marca fica no arquivo local. Para guardar num Postgres (o plano é o Supabase, no plano grátis), ponha a URL de conexão na variável `GARIMPO_DATABASE_URL`. Ela leva a senha do banco, por isso vai só no ambiente: nunca num arquivo do repositório nem numa flag.

```bash
export GARIMPO_DATABASE_URL='postgres://usuario:senha@host:5432/postgres?sslmode=require'
go run ./cmd/garimpo migrate   # cria as tabelas; pode rodar de novo sem medo
go run ./cmd/garimpo import    # copia para o banco o que já estava marcado no arquivo
```

No Supabase, a URL está no botão **Connect** do projeto. Use a do **Session pooler**, que funciona em rede IPv4.

Depois de atualizar o projeto, rode o `migrate` de novo: quando há tabela para atualizar, os outros comandos avisam e esperam por ele.

Com a variável definida, `collect`, `serve`, `mark`, `note`, `list` e `show` passam a usar o banco, com os mesmos números e estados. O `import` só roda num banco vazio e não mexe no arquivo, que fica como cópia. Para voltar ao arquivo numa rodada, use `-state file`.

As tabelas nascem com RLS ligado e sem nenhuma policy: a API REST que o Supabase publica sozinho não lê nem escreve nada. Só o Garimpo entra, pela conexão do Postgres.

### API avulsa

O `garimpo serve` já traz a API junto com o painel, em `/api/`. Para usar a API sem o painel (um script, outro programa), existe o `garimpo-api`, protegido por token. Ela lê e escreve o registro (arquivo ou banco, pela mesma regra da CLI) e não coleta.

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
| `GET /funnel` | a contagem de cada estado e as últimas mudanças |
| `GET /health` | responde `ok`, sem token |

Toda rota, menos `/health`, exige o token. Os estados são `new`, `saved`, `applied`, `interview` e `discarded`; as palavras da CLI ("apliquei") também valem. A API não fala HTTPS: fora da sua máquina, só atrás de um proxy que fale.

## Por que existe

Candidatura em massa costuma render menos entrevista, não mais. O Garimpo aposta no contrário: poucas vagas, bem escolhidas, com o motivo da escolha à vista.

Feito por [Victor Novakoski](https://github.com/Victor-Novakoski).
