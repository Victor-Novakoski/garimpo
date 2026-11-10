# Decisões

Registro das escolhas e do porquê. A mais nova em cima.

| Data | Decisão | Por quê |
| --- | --- | --- |
| 10/11/2026 | Depois da coleta real: título vence etiqueta e local; faixa "Pleno/Sênior" vira Pleno; só vagas de tech por padrão (`-all` mostra tudo); duplicadas pela mesma empresa com títulos parecidos | Na primeira coleta (499 vagas), issues tinham etiquetas "Remoto" e "Híbrido" juntas, só ~100 vagas do Greenhouse eram de tech e a mesma vaga aparecia em três repositórios com títulos um pouco diferentes |
| 10/11/2026 | Greenhouse entra já na etapa 1, com Stone, QuintoAndar e Wildlife Studios | É API oficial e pública, e foi a fonte brasileira com mais vagas na pesquisa (Stone sozinha tinha 347) |
| 10/11/2026 | Normalização sem dependência externa: acentos tirados por uma tabela própria | `golang.org/x/text` atual exige Go 1.26; a tabela cobre o português e mantém o módulo sem dependências |
| 10/11/2026 | Gupy entra só para uso pessoal: coletor desligado por padrão, só link para a vaga original, nunca no demo público | É onde está a maior parte das vagas BR, mas os termos proíbem agregar o conteúdo. Assim o risco fica baixo e o projeto público segue limpo |
| 10/11/2026 | IA atrás de uma interface `Scorer`, com duas implementações: API do Claude (Haiku) e `claude -p` local | A API custa uns R$ 5 a 10 por mês; o modo local usa o plano do Claude que o Victor já paga. Trocar não mexe no resto |
| 10/11/2026 | Nota híbrida: até 70 pontos em código e até 30 da IA, com filtro duro antes | Nota explicável e barata: a IA só vê o que passou no filtro e não consegue furar regra. Ideia tirada da friday, do Carlos Junio |
| 10/11/2026 | Foco no Brasil: repositórios de vagas no GitHub e Gupy primeiro; Greenhouse e Lever depois | O Victor busca vaga brasileira por enquanto. A friday já cobre bem o lado internacional |
| 10/11/2026 | Sem LinkedIn e sem candidatura automática | LinkedIn proíbe scraping e bloqueia conta. Candidatura em massa rende menos entrevista; o valor está em escolher bem |
| 10/11/2026 | Job diário no GitHub Actions em vez de um worker sempre ligado | Cabe no plano grátis e o histórico de execuções fica visível no repositório |
| 10/11/2026 | Go para coleta, nota e API; React no painel | É a stack que o Victor quer mostrar nas vagas que procura |
| 10/11/2026 | Nome: garimpo | Curto, em português, e diz o que faz: peneira muita vaga e fica com o ouro |
