# garimpo

Peneira as vagas de tecnologia do Brasil e fica só com o ouro.

O Garimpo coleta vagas de fontes abertas uma vez por dia, descarta o que não serve, dá a cada vaga uma nota explicável de 0 a 100 e entrega as melhores. Ele **não** se candidata sozinho: acha, filtra e prepara, e a decisão é sempre sua.

> Em construção. Veja o [roadmap](docs/ROADMAP.md).

## Como funciona

1. **Coleta** de fontes abertas: repositórios de vagas no GitHub e boards do Greenhouse de empresas brasileiras. Sem LinkedIn.
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

Com `GITHUB_TOKEN` definido, a coleta do GitHub usa o limite maior da API.

## Por que existe

Candidatura em massa costuma render menos entrevista, não mais. O Garimpo aposta no contrário: poucas vagas, bem escolhidas, com o motivo da escolha à vista.

Feito por [Victor Novakoski](https://github.com/Victor-Novakoski).
