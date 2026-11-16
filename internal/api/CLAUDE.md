# internal/api

As rotas HTTP (listar, ver, marcar, anotar, funil, coletar, carta), o token do dono e a trava do painel local.

## Regras

- **Toda rota do `garimpo serve` passa pelo `LocalOnly`**, inclusive a do painel. Nada de CORS aberto, login ou endereço configurável no `serve`.
- A `garimpo-api` avulsa usa `Token`: pelo menos 32 caracteres, comparado em tempo constante. Sem token, toda rota (menos `/health`) responde 401, exista ou não. A checagem fica atrás de `Authenticator`.
- **Erro do banco vai para o log, nunca para a resposta.** A coleta pelo painel devolve só uma frase.
- **`POST /search` não lê nada do pedido.** A coleta usa as opções que o `serve` recebeu por flag. Uma por vez: 409 se já há outra.
- **`POST /jobs/{number}/letter` também não lê nada do pedido** e não grava nada: devolve o rascunho da carta em texto. Quem escreve é o `LetterFunc` que o `serve` monta com `-ai`; sem ele a rota responde 409 e o detalhe da vaga diz o motivo em `letter_off`. Uma por vez (409), com prazo próprio de 2 minutos, maior que o `WriteTimeout` do servidor.
- Os estados aceitos e o formato do `PATCH` seguem o [`docs/FLOW.md`](../../docs/FLOW.md); as palavras da CLI ("apliquei") também valem.
- Roteador da biblioteca padrão (`GET /jobs/{number}`), sem dependência.
- **Nome de rota não pode parecer rastreador.** A coleta fica em `/search`, não em `/collect`: bloqueador de anúncio (o do Opera, uBlock) barra pedidos com `collect`, `track`, `analytics` ou `beacon` no caminho, e o painel fica mudo sem o servidor ver nada.

## Ver também

Decisões 032, 033, 037, 039 e 044; [`docs/SECURITY.md`](../../docs/SECURITY.md#painel-local).
