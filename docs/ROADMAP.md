# Roadmap

Seis etapas, um PR por etapa para a `develop`. Ao fim da etapa 2 o Garimpo já entrega vagas boas todo dia; o resto é acabamento e vitrine.

## Etapa 0: fundação (2 a 4 h)

- [x] README, AGENTS.md e docs (arquitetura, decisões, roadmap, fontes)
- [x] Esqueleto Go (`cmd/garimpo`, `internal/`), testes e CI
- [x] Confirmar o acesso da Gupy em [SOURCES.md](SOURCES.md)

## Etapa 1: coleta (8 a 12 h)

- [ ] Coletores: backend-br/vagas, frontendbr/vagas, Gupy (só uso pessoal, desligado por padrão)
- [ ] Cliente HTTP educado (timeout, limite de tamanho, intervalo, `User-Agent`)
- [ ] Normalização e remoção de duplicadas, com testes usando respostas salvas

## Etapa 2: nota e aviso (8 a 12 h)

- [ ] Perfil YAML (e `profile.example.yaml`)
- [ ] Filtro duro e nota determinística com motivos
- [ ] Nota com IA atrás da interface `Scorer`
- [ ] Job diário no GitHub Actions + resumo no Telegram

## Etapa 3: banco e API (10 a 15 h)

- [ ] Postgres no Supabase com migrações
- [ ] API: listar, detalhar, mudar status, gerar carta
- [ ] Login e proteção da API

## Etapa 4: painel (15 a 20 h)

- [ ] Lista por nota, filtros, detalhe com motivos
- [ ] Funil: nova, vou aplicar, apliquei, entrevista, descartada
- [ ] Carta e pontos do currículo para destacar
- [ ] Deploy na Vercel, testado a 390 px

## Etapa 5: vitrine (5 a 8 h)

- [ ] Coletores de Greenhouse e Lever
- [ ] README com prints, diagrama e as decisões
- [ ] Modo demo com dados fictícios, para ver sem login
