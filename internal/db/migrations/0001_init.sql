-- O que o Garimpo guarda: as vagas marcadas (entries), as chaves de cada
-- cópia, o histórico de estados, o texto da vaga e as rodadas. As regras
-- estão em docs/FLOW.md.

create table entries (
    number     integer primary key check (number > 0),
    status     text not null default 'new'
               check (status in ('new', 'saved', 'applied', 'interview', 'discarded')),
    note       text not null default '',
    title      text not null default '',
    company    text not null default '',
    url        text not null default '',
    -- A nota e os motivos são os da última rodada que listou a vaga. Não são
    -- a verdade: a nota é recalculada a cada rodada.
    score      integer not null default 0,
    reasons    text[] not null default '{}',
    first_seen timestamptz not null,
    last_seen  timestamptz not null,
    gone       boolean not null default false
);

create index entries_status_idx on entries (status);

-- A identidade da vaga é a fonte mais o id na fonte. A mesma vaga em duas
-- fontes tem duas chaves e uma entrada só.
create table entry_keys (
    key    text primary key,
    number integer not null references entries (number) on delete cascade,
    seq    bigint generated always as identity
);

create index entry_keys_number_idx on entry_keys (number);

create table entry_events (
    id          bigint generated always as identity primary key,
    number      integer not null references entries (number) on delete cascade,
    at          timestamptz not null,
    from_status text not null
                check (from_status in ('new', 'saved', 'applied', 'interview', 'discarded')),
    to_status   text not null
                check (to_status in ('new', 'saved', 'applied', 'interview', 'discarded'))
);

create index entry_events_number_idx on entry_events (number);

-- O texto da vaga como veio da última coleta, para a tela de detalhe.
create table jobs (
    key         text primary key,
    number      integer not null references entries (number) on delete cascade,
    source      text not null,
    source_id   text not null,
    url         text not null default '',
    title       text not null default '',
    company     text not null default '',
    location    text not null default '',
    description text not null default '',
    tags        text[] not null default '{}',
    work_mode   text not null default '',
    contract    text not null default '',
    seniority   text not null default '',
    posted_at   timestamptz,
    seen_at     timestamptz not null
);

create index jobs_number_idx on jobs (number);

create table runs (
    id          bigint generated always as identity primary key,
    at          timestamptz not null,
    listed      integer not null,
    new_entries integer not null,
    sources     text[] not null default '{}'
);

-- No Supabase, toda tabela do schema public fica exposta pela API REST
-- deles. Com RLS ligado e nenhuma policy, as chaves anon e authenticated
-- não leem nem escrevem nada. O Garimpo entra como dono das tabelas, pela
-- conexão direta do Postgres, e o dono não passa pelo RLS.
alter table entries enable row level security;
alter table entry_keys enable row level security;
alter table entry_events enable row level security;
alter table jobs enable row level security;
alter table runs enable row level security;
