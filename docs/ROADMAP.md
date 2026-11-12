# Roadmap

Seis etapas, um PR por etapa para a `develop`. Ao fim da etapa 2 o Garimpo já entrega vagas boas todo dia; o resto é acabamento e vitrine.

## Etapa 0: fundação (2 a 4 h)

- [x] README, AGENTS.md e docs (arquitetura, decisões, roadmap, fontes)
- [x] Esqueleto Go (`cmd/garimpo`, `internal/`), testes e CI
- [x] Confirmar o acesso da Gupy em [SOURCES.md](SOURCES.md)

## Etapa 1: coleta (8 a 12 h)

- [x] Coletores: backend-br/vagas, frontendbr/vagas, react-brasil/vagas, Gupy (só uso pessoal, desligado por padrão)
- [x] Coletor do Greenhouse para empresas brasileiras (adiantado da etapa 5: é a fonte oficial com mais vagas)
- [x] Cliente HTTP educado (timeout, limite de tamanho, intervalo, `User-Agent`)
- [x] Normalização e remoção de duplicadas, com testes usando respostas salvas
- [x] Comando `garimpo collect`

## Etapa 2: nota e aviso (8 a 12 h)

- [x] Perfil YAML (e `profile.example.yaml`)
- [x] Filtro duro e nota determinística com motivos (incluindo país: o Greenhouse traz vagas do México, Argentina e Chile)
- [ ] Modelo de trabalho das vagas do QuintoAndar e da Wildlife, que hoje saem sem modelo
- [x] Vaga sem modelo mas com cidade no título ou no local ("[Belo Horizonte]") não deve passar como se fosse remota
- [x] Nota com IA atrás da interface `Scorer` (local com `claude -p` e API)
- [ ] Conjunto de vagas rotuladas à mão para medir a IA e comparar modelos
- [x] Registro local dos estados da vaga e comandos `mark`, `list`, `show` e `note` (veja [FLOW.md](FLOW.md))
- [ ] Job diário no GitHub Actions + resumo no Telegram (adiado: por enquanto roda local)

## Etapa 3: banco e API (10 a 15 h)

- [x] Postgres com migrações e o registro de estados atrás da mesma interface do arquivo
- [x] Criar o projeto no Supabase e aplicar a migração nele (a primeira coleta real gravou 121 vagas pelo Session pooler)
- [x] API: listar, detalhar, mudar status e anotar, com os estados do [FLOW.md](FLOW.md)
- [ ] API: gerar carta
- [x] Importar o registro local para o banco (`garimpo import`)
- [x] Proteção da API por token do dono
- [x] ~~Login do painel com Supabase Auth~~ e ~~hospedar a API~~: saíram do plano, o painel roda só na máquina (veja [DECISIONS.md](DECISIONS.md))
- [ ] Cache das notas da IA no banco (hoje é um arquivo)

## Etapa 4: painel local (15 a 20 h)

O painel roda só na máquina do Victor: sem deploy, sem login. O Supabase segue como banco.

- [x] `garimpo serve`: a API e o painel num comando só, em `127.0.0.1`
- [x] Lista do dia por nota, com estado, empresa, modelo e link; filtro por estado e busca
- [x] Tela da vaga: motivos da nota, leitura da IA (com o que destacar na candidatura), anotação e histórico
- [x] Marcar com um clique (vou aplicar, apliquei, entrevista, descartada), com desfazer
- [x] Funil com a contagem de cada estado e as últimas mudanças
- [x] Botão para rodar a coleta pelo painel
- [x] Testado a 390 px, claro e escuro; CI cobrindo o `web/`
- [ ] Carta de apresentação (depende da rota de gerar carta, da etapa 3)

## Etapa 5: vitrine (5 a 8 h)

- [x] Coletor do Lever e mais empresas no Greenhouse (adiantado: 16 empresas no Greenhouse e 3 no Lever)
- [ ] README com prints, diagrama e as decisões
- [ ] Modo demo com dados fictícios, para mostrar o painel sem os dados reais
