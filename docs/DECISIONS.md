# Decisões

Registro das escolhas e do porquê. A mais nova em cima.

| Data | Decisão | Por quê |
| --- | --- | --- |
| 10/11/2026 | A IA devolve uma classe (forte, bom, parcial, fraco, nenhum) e o código dá os pontos (30, 22, 14, 6, 0) | Um número solto do modelo varia de uma rodada para outra e não dá para validar. Classe fixa é estável, explicável e não passa do teto |
| 10/11/2026 | Modo local roda `claude -p` sem ferramentas, sem MCP, em modo seguro e numa pasta vazia, com a vaga no stdin | A descrição da vaga é texto de terceiros. Sem ferramenta, uma vaga maliciosa não lê arquivo nem roda comando; só consegue mexer na própria classe. O modo local também tira `ANTHROPIC_API_KEY` do ambiente do `claude`, para usar o plano e não cobrar a API sem querer |
| 10/11/2026 | Modo API usa o SDK oficial da Anthropic, e o Go sobe para 1.25 | O SDK cuida de retry, erros tipados e saída estruturada, e é o caminho recomendado. Custa a subida de versão do Go e as dependências dele; a coleta segue só com a biblioteca padrão |
| 10/11/2026 | Cache das respostas da IA em arquivo, com chave de modelo + prompt + perfil + vaga | A mesma vaga aparece por semanas. Sem cache, cada rodada local levaria minutos e gastaria o plano à toa. Vira tabela no Postgres na etapa 3 |
| 10/11/2026 | A IA só avalia as 20 melhores pela nota das regras | Vaga com nota baixa nas regras não chega aos cortes mesmo com 30 da IA, então avaliar todas só gasta tempo |
| 10/11/2026 | Vaga sem modelo que cita uma cidade ("[Belo Horizonte, Brasil]") ganha 0 no modelo, com o motivo à vista, em vez de sair do filtro. Vagas de outras linguagens continuam na lista | O Victor topa aprender outra linguagem para começar logo; esconder a vaga tiraria a escolha dele. A nota baixa e o motivo bastam |
| 10/11/2026 | Skill na seção de requisitos vale ponto cheio; links não contam; vaga sem nenhuma skill do perfil sai no filtro | Na segunda nota real, a Jazida pedia React nos requisitos e levava só metade, a Coodesh ganhava React por um `utm_medium=react-brasil-vagas` e vagas de vendas e ciência de dados com 0 de tecnologia ainda apareciam |
| 10/11/2026 | Depois da primeira nota real: skill só na descrição vale metade; "go-to-market" e "React Native" não contam como Go e React; staff, principal e head viram nível `lead`; modelo não informado vale 2 de 10 | O QuintoAndar ganhava 30 pontos com o mesmo parágrafo de stack em toda vaga, uma vaga de vendas ganhou Go por "go-to-market" e "Head of Engineering" passava como sênior |
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
