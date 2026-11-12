# Segurança

O Garimpo guarda dados de uma pessoa só (as vagas que ela marcou, anotações, o perfil) e lê texto da internet que ninguém revisou. As defesas abaixo cobrem esses dois lados. As regras que o código tem que seguir estão no `CLAUDE.md` de cada módulo; aqui fica o porquê.

| Risco | Como é tratado | Onde |
| --- | --- | --- |
| Outro site no navegador chamar o painel | O `serve` só escuta em `127.0.0.1` e toda rota passa pelo `api.LocalOnly` | `internal/api` |
| Alguém de fora usar a `garimpo-api` | Token do dono, comparado em tempo constante | `internal/api` |
| Ler o banco pela API REST do Supabase | RLS ligado sem nenhuma policy | `internal/db` |
| Vazar a URL do banco | Só no ambiente; nunca em flag, log ou erro | `internal/db`, `internal/tracker` |
| Uma vaga manipular a IA | Texto marcado como dado, `claude -p` sem ferramentas, classe fixa com teto | `internal/ai` |
| Uma vaga injetar HTML ou link perigoso no painel | Texto sempre como texto; link só `http`/`https`; política de conteúdo restrita | `web/`, `internal/panel` |
| Abusar das fontes ou ferir os termos delas | Coleta educada e fontes conferidas antes de entrar | `internal/httpx`, [SOURCES.md](SOURCES.md) |
| Dados pessoais no repositório | `profile.yaml`, `.env` e dados privados no `.gitignore` | raiz |

## Painel local

O painel não tem login, então a proteção é conferir de onde vem cada pedido. Escutar no loopback deixa as outras máquinas de fora, mas qualquer site aberto no navegador consegue chamar `http://127.0.0.1`. Por isso um pedido só passa quando:

- o `Host` é `localhost` ou um endereço de loopback (um site que aponta o próprio domínio para `127.0.0.1` chega com o domínio dele);
- o `Origin`, quando o navegador manda, é o próprio servidor;
- o `Sec-Fetch-Site`, quando o navegador manda, diz que o pedido saiu da própria página (ou que o endereço foi digitado, e aí só para leitura).

Não existe flag para trocar o endereço do `serve`, só a porta. As páginas saem com uma política de conteúdo que só carrega o que vem do próprio servidor e não deixa a página ser posta num `iframe`.

O botão "Buscar vagas" (`POST /api/collect`) não leva nenhuma opção: a coleta é a que o `serve` recebeu por flag ao subir. O máximo que um pedido indevido conseguiria é rodar a mesma coleta que o dono já roda, uma por vez.

## API avulsa e banco

- **Token do dono** em `GARIMPO_API_TOKEN`, com pelo menos 32 caracteres. Sem ele toda rota, menos `/health`, responde 401, exista ou não.
- **A `garimpo-api` escuta em `127.0.0.1` por padrão e não fala HTTPS.** Para sair da máquina, só atrás de um proxy com HTTPS.
- **RLS ligado sem policy** em todas as tabelas: o Supabase publica uma API REST para o schema `public`, e a chave pública não enxerga nada. O Garimpo entra como dono das tabelas, pela conexão do Postgres, que não passa pelo RLS.
- **A URL do banco leva a senha**, por isso só vem de `GARIMPO_DATABASE_URL`. Em flag ela ficaria no histórico do terminal e na lista de processos. Erro do banco vai para o log do servidor e nunca para a resposta.

## Texto de vaga

O texto vem da internet e é tratado como dado em todo lugar:

- **Na IA**, vai marcado como dado e o prompt manda ignorar instruções dentro dele. O modo local roda o `claude -p` sem ferramentas, sem MCP, em modo seguro, numa pasta vazia e sem `ANTHROPIC_API_KEY` no ambiente. Mesmo que uma vaga engane o modelo, o máximo que ela consegue é a própria classe: 30 pontos, numa vaga que já passou no filtro. Resposta fora do formato é descartada.
- **Na tela**, nunca vira HTML, e o link da vaga só vira link se for `http` ou `https`.
