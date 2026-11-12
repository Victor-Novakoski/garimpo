# internal/ai

A nota da IA, de 0 a 30, atrás da interface `Scorer`, com dois modos que usam o mesmo prompt e o mesmo schema.

## Regras

- **A IA escolhe uma classe e o código dá os pontos:** forte 30, bom 22, parcial 14, fraco 6, nenhum 0. Nunca aceite um número do modelo.
- Resposta fora do formato (classe desconhecida, campo a mais, texto solto) é descartada: a vaga fica com a nota das regras e o erro aparece no `-why`.
- Só avalia as melhores pela nota das regras (`-ai-top`, 20 por padrão): abaixo disso nem 30 pontos levariam a vaga aos cortes.
- **Texto de vaga é dado, não instrução.** Vai marcado como dado e o prompt manda ignorar instruções dentro dele.
- **Modo local** (`claude -p`): sem ferramentas, sem MCP, em modo seguro, numa pasta vazia, com a vaga no stdin. Tira `ANTHROPIC_API_KEY` e `ANTHROPIC_AUTH_TOKEN` do ambiente, para usar o plano e não cobrar a API sem querer. Não afrouxe nenhuma dessas flags.
- **Modo API**: SDK oficial, saída estruturada, só fala com `api.anthropic.com`.
- Testes nunca chamam modelo: respostas salvas em `testdata/`.

## Armadilhas conhecidas

- O cache fica num arquivo na pasta de cache do usuário, por 60 dias. A chave junta modelo, prompt, perfil e vaga: mudar o prompt reavalia tudo. Erro não entra no cache.
- A tela da vaga lê a leitura da IA das linhas de motivo já gravadas (decisão 035). Mudar o texto dessas linhas pede mudar o parser do painel junto.

## Ver também

Decisões 007, 015, 016, 017, 018 e 019; [`docs/SECURITY.md`](../../docs/SECURITY.md#texto-de-vaga).
