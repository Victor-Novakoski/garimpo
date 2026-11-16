# 043. O modo demo passa pela coleta de verdade

- **Data:** 2026-11-16
- **Status:** ativa

## Contexto

O projeto é de portfólio, mas o painel só mostra algo com o perfil e as vagas de quem usa, que não podem aparecer em print nem em demonstração. Faltava um jeito de subir o painel com dados de mentira, sem banco e sem rede (issue #17).

O caminho curto era um arquivo com as entradas prontas (nota, motivos, estados). Só que aí a nota do demo seria um texto escrito à mão, que deixa de bater com o código na primeira mudança de regra.

## Decisão

O `garimpo serve -demo` lê vagas inventadas de `demo/vagas.json` e o `profile.example.yaml`, e passa as vagas pelos mesmos passos de uma coleta: classificar, juntar repetidas, filtro duro, nota das regras e registro. O arquivo entra como um coletor a mais (`exemplo`), no lugar dos que usam a rede.

Só a leitura da IA vem pronta no arquivo, no formato que o modelo devolveria, e é validada pelo mesmo `ai.Parse`.

O registro do demo fica só na memória (`tracker.Memory`): nada é gravado, e o modo recusa `-state`, `-profile`, `-ai` e qualquer outra opção que aponte para os dados reais.

## Consequência

- O demo nunca mostra uma nota que o código não daria, e uma mudança de regra aparece nele sem ninguém editar o arquivo.
- As datas do arquivo são contadas para trás a partir da hora em que o demo sobe, então ele parece recente em qualquer dia.
- O que se marca no demo some ao fechar. O botão "Buscar vagas" fica desligado.
- Vaga de exemplo usa fonte `exemplo`, empresa inventada e link em `example.com`. Nada da Gupy (decisão 008).
- `tracker.Store` ganhou a terceira implementação, que passa pelos mesmos testes das outras duas.
