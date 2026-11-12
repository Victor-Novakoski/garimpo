# 042. Docs no padrão CLAUDE.md + INDEX + decisões numeradas

- **Data:** 2026-11-12
- **Status:** ativa

## Contexto

Os docs cresciam num formato próprio: um `AGENTS.md` com todas as regras, um `DECISIONS.md` com uma tabela que só aumentava e um `ROADMAP.md` com as tarefas. Para achar a regra de um módulo era preciso ler tudo, e a mesma regra aparecia em dois ou três arquivos.

## Decisão

Docs no padrão usado nos outros projetos do Victor: `CLAUDE.md` raiz curto, `docs/INDEX.md` como roteador, um `CLAUDE.md` por módulo com regras e armadilhas, e uma decisão por arquivo em `docs/decisoes/`. Para pessoas ficam README, ARCHITECTURE (curta), FLOW, SECURITY e SOURCES. `AGENTS.md`, `DECISIONS.md` e `ROADMAP.md` saem; tarefas viram issues com as etiquetas `prioridade` e `ideia`.

## Consequência

Cada regra mora num lugar só e uma sessão nova lê só o que a tarefa pede. Decisão não se edita: uma nova substitui a antiga e marca o status.
