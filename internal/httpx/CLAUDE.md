# internal/httpx

O cliente HTTP educado. É o único caminho da coleta para a rede: coletor nenhum usa `http.Get` ou um `http.Client` próprio.

## Regras

- Só GET, e só para os hosts que o coletor declarou em `Options.Hosts`. Outro host dá `ErrHostNotAllowed`.
- Timeout por pedido, limite de tamanho da resposta e intervalo mínimo entre pedidos à mesma fonte (padrões no topo do `client.go`).
- Todo pedido leva o `UserAgent`, que identifica o projeto e aponta para o repositório.
- A coleta roda quando o dono pede (CLI ou botão do painel), não em laço. Não acrescente repetição automática sem uma decisão nova (ver decisão 020).

## Ver também

- [`docs/SOURCES.md`](../../docs/SOURCES.md): quais fontes podem ser chamadas.
