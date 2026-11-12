# internal/api

As rotas HTTP (listar, ver, marcar, anotar, funil, coletar), o token do dono e a trava do painel local.

## Regras

- **Toda rota do `garimpo serve` passa pelo `LocalOnly`**, inclusive a do painel. Nada de CORS aberto, login ou endereço configurável no `serve`.
- A `garimpo-api` avulsa usa `Token`: pelo menos 32 caracteres, comparado em tempo constante. Sem token, toda rota (menos `/health`) responde 401, exista ou não. A checagem fica atrás de `Authenticator`.
- **Erro do banco vai para o log, nunca para a resposta.** A coleta pelo painel devolve só uma frase.
- **`POST /collect` não lê nada do pedido.** A coleta usa as opções que o `serve` recebeu por flag. Uma por vez: 409 se já há outra.
- Os estados aceitos e o formato do `PATCH` seguem o [`docs/FLOW.md`](../../docs/FLOW.md); as palavras da CLI ("apliquei") também valem.
- Roteador da biblioteca padrão (`GET /jobs/{number}`), sem dependência.

## Ver também

Decisões 032, 033, 037 e 039; [`docs/SECURITY.md`](../../docs/SECURITY.md#painel-local).
