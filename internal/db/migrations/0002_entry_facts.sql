-- O modelo de trabalho, o contrato, o nível e o local passam a ficar também
-- na entrada, para a lista do painel mostrar sem ler o texto de cada vaga.
-- São da última rodada que listou a vaga, como o título e a empresa.

alter table entries
    add column work_mode text not null default '',
    add column contract  text not null default '',
    add column seniority text not null default '',
    add column location  text not null default '';

-- As vagas que já estavam no banco ganham os valores do texto guardado.
update entries e
set work_mode = j.work_mode, contract = j.contract, seniority = j.seniority, location = j.location
from (
    select distinct on (number) number, work_mode, contract, seniority, location
    from jobs
    order by number, seen_at desc, key
) j
where j.number = e.number;
