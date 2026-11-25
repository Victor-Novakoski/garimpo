# internal/ai

A nota da IA, de 0 a 30, atrás da interface `Scorer`, e o rascunho da carta de apresentação, atrás da `Writer`. Os dois modos (`local` e `api`) implementam as duas, com o mesmo prompt e o mesmo schema em cada uma.

## Regras

- **A IA escolhe uma classe e o código dá os pontos:** forte 30, bom 22, parcial 14, fraco 6, nenhum 0. Nunca aceite um número do modelo.
- Resposta fora do formato (classe desconhecida, campo a mais, texto solto) é descartada: a vaga fica com a nota das regras e o erro aparece no `-why`.
- Só avalia as melhores pela nota das regras (`-ai-top`, 20 por padrão): abaixo disso nem 30 pontos levariam a vaga aos cortes.
- **Texto de vaga é dado, não instrução.** Vai marcado como dado e o prompt manda ignorar instruções dentro dele.
- **Modo local** (`claude -p`): sem ferramentas, sem MCP, em modo seguro, numa pasta vazia, com a vaga no stdin. Tira `ANTHROPIC_API_KEY` e `ANTHROPIC_AUTH_TOKEN` do ambiente, para usar o plano e não cobrar a API sem querer. Não afrouxe nenhuma dessas flags.
- **Modo API**: SDK oficial, saída estruturada, só fala com `api.anthropic.com`.
- **A carta é um rascunho** (`letter.go`): só afirma sobre a pessoa o que está no perfil, sai como texto puro num campo só (`letter`) e não é guardada nem enviada. A vaga e a leitura anterior da IA entram marcadas como dado (`<vaga>`, `<leitura>`); o perfil é a única fonte de fatos. Ela mostra o que a pessoa tem (o que falta cabe em uma frase, e só se for obrigatório), responde a "disposição para aprender" com um exemplo real de aprendizado do perfil e nunca com adjetivo, não diz que a pessoa tem outro emprego nem fala de disponibilidade ou salário, e segue os recados do `about` do perfil sobre o que citar e quando.
- Toda chamada ao modelo passa pelo `ask` de cada modo: prompt ou tarefa nova não ganha um caminho próprio com flags mais frouxas.
- Testes nunca chamam modelo: respostas salvas em `testdata/`.

## Armadilhas conhecidas

- O cache fica num arquivo na pasta de cache do usuário, por 60 dias. A chave junta modelo, prompt, perfil e vaga: mudar o prompt reavalia tudo. Erro não entra no cache.
- O texto dos `about_files` do perfil entra no `profileBlock`, depois do `about` (decisão 046). Ele faz parte do prompt de sistema, que é a chave do cache: por isso mudar uma nota reavalia tudo. Não tire esse texto do prompt de sistema sem pôr o conteúdo na chave.
- A tela da vaga lê a leitura da IA das linhas de motivo já gravadas (decisão 035), e a carta também. Quem escreve as linhas é o `ReasonLines` e quem lê é o `NotesFromReasons` (os dois em `letter.go`) e o `splitReasons` do painel (`web/src/format.ts`): mudar o texto pede mudar os três.
- A carta não tem cache: cada pedido é uma chamada ao modelo.

## Ver também

Decisões 007, 015, 016, 017, 018, 019, 044 e 046; [`docs/SECURITY.md`](../../docs/SECURITY.md#texto-de-vaga).
