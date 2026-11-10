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

Precisa de Go 1.24 ou mais novo.

```bash
go run ./cmd/garimpo collect          # coleta e lista as vagas dos últimos 30 dias
go run ./cmd/garimpo collect -json    # uma vaga por linha, em JSON
go test ./...
```

Com `GITHUB_TOKEN` definido, a coleta do GitHub usa o limite maior da API.

## Por que existe

Candidatura em massa costuma render menos entrevista, não mais. O Garimpo aposta no contrário: poucas vagas, bem escolhidas, com o motivo da escolha à vista.

Feito por [Victor Novakoski](https://github.com/Victor-Novakoski).
