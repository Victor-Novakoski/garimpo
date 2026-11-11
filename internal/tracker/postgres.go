package tracker

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/victor-novakoski/garimpo/internal/db"
	"github.com/victor-novakoski/garimpo/internal/job"
)

// Postgres is a Store kept in the database (migrations in internal/db). It
// applies the same rules as File: a run loads the entries, applies
// book.sync and writes back what changed, all in one transaction.
type Postgres struct {
	pool *pgxpool.Pool
}

// timeout bounds every call: Store has no context, and a database that
// stopped answering must not hang the command line or a request.
const timeout = 30 * time.Second

// OpenPostgres returns the store over a pool from db.Open. The caller
// closes the pool.
func OpenPostgres(pool *pgxpool.Pool) *Postgres { return &Postgres{pool: pool} }

// querier is what pgxpool.Pool and pgx.Tx have in common.
type querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

func statusNames(statuses []Status) []string {
	out := make([]string, len(statuses))
	for i, s := range statuses {
		out[i] = string(s)
	}
	return out
}

// load reads entries with their keys and history, ordered by number. With
// numbers it reads only those; with statuses, only the ones in them.
func load(ctx context.Context, q querier, numbers []int32, statuses []Status) ([]Entry, error) {
	const cols = `select number, status, note, title, company, url, score, reasons, first_seen, last_seen, gone from entries`
	var (
		rows pgx.Rows
		err  error
	)
	switch {
	case numbers != nil:
		rows, err = q.Query(ctx, cols+` where number = any($1) order by number`, numbers)
	case len(statuses) > 0:
		rows, err = q.Query(ctx, cols+` where status = any($1) order by number`, statusNames(statuses))
	default:
		rows, err = q.Query(ctx, cols+` order by number`)
	}
	if err != nil {
		return nil, err
	}
	entries, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (Entry, error) {
		var e Entry
		err := row.Scan(&e.Number, &e.Status, &e.Note, &e.Title, &e.Company, &e.URL, &e.Score, &e.Reasons, &e.FirstSeen, &e.LastSeen, &e.Gone)
		if len(e.Reasons) == 0 {
			e.Reasons = nil
		}
		return e, err
	})
	if err != nil || len(entries) == 0 {
		return entries, err
	}
	at := make(map[int]int, len(entries)) // number -> index
	found := make([]int32, len(entries))
	for i, e := range entries {
		at[e.Number] = i
		found[i] = int32(e.Number)
	}

	rows, err = q.Query(ctx, `select number, key from entry_keys where number = any($1) order by seq`, found)
	if err != nil {
		return nil, err
	}
	var (
		number int
		key    string
	)
	if _, err := pgx.ForEachRow(rows, []any{&number, &key}, func() error {
		e := &entries[at[number]]
		e.Keys = append(e.Keys, key)
		return nil
	}); err != nil {
		return nil, err
	}

	rows, err = q.Query(ctx, `select number, at, from_status, to_status from entry_events where number = any($1) order by id`, found)
	if err != nil {
		return nil, err
	}
	var ev Event
	if _, err := pgx.ForEachRow(rows, []any{&number, &ev.At, &ev.From, &ev.To}, func() error {
		e := &entries[at[number]]
		e.History = append(e.History, ev)
		return nil
	}); err != nil {
		return nil, err
	}
	return entries, nil
}

// one reads a single entry.
func one(ctx context.Context, q querier, number int) (Entry, error) {
	entries, err := load(ctx, q, []int32{int32(number)}, nil)
	if err != nil {
		return Entry{}, err
	}
	if len(entries) == 0 {
		return Entry{}, ErrNotFound(number)
	}
	return entries[0], nil
}

// inTx runs change in a transaction that holds the database lock, so two
// writers never hand out the same number or overwrite each other.
func (p *Postgres) inTx(change func(ctx context.Context, tx pgx.Tx) error) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err := db.Lock(ctx, tx); err != nil {
		return err
	}
	if err := change(ctx, tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// loadBook reads every entry. Numbers are never reused, so the next one
// comes after the highest.
func loadBook(ctx context.Context, tx pgx.Tx) (book, error) {
	entries, err := load(ctx, tx, nil, nil)
	if err != nil {
		return book{}, err
	}
	b := book{Next: 1, Entries: entries}
	if n := len(entries); n > 0 {
		b.Next = entries[n-1].Number + 1
	}
	return b, nil
}

const upsertEntry = `
insert into entries (number, status, note, title, company, url, score, reasons, first_seen, last_seen, gone)
values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
on conflict (number) do update set
	status = excluded.status, note = excluded.note, title = excluded.title,
	company = excluded.company, url = excluded.url, score = excluded.score,
	reasons = excluded.reasons, first_seen = excluded.first_seen,
	last_seen = excluded.last_seen, gone = excluded.gone`

// A key belongs to one entry. When a run says a posting that had its own
// entry is a copy of another, lookups by that key follow the newer answer.
const upsertKey = `
insert into entry_keys (key, number) values ($1, $2)
on conflict (key) do update set number = excluded.number where entry_keys.number <> excluded.number`

// The writes of a run or an import go in one batch: a run lists a few
// hundred postings, and one round trip per row would take minutes on a
// database far away.

func queueEntry(b *pgx.Batch, e Entry) {
	reasons := e.Reasons
	if reasons == nil {
		reasons = []string{}
	}
	b.Queue(upsertEntry, e.Number, string(e.Status), e.Note, e.Title, e.Company, e.URL,
		e.Score, reasons, e.FirstSeen, e.LastSeen, e.Gone)
	for _, k := range e.Keys {
		b.Queue(upsertKey, k, e.Number)
	}
}

const insertEvent = `insert into entry_events (number, at, from_status, to_status) values ($1, $2, $3, $4)`

const upsertJob = `
insert into jobs (key, number, source, source_id, url, title, company, location, description, tags, work_mode, contract, seniority, posted_at, seen_at)
values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
on conflict (key) do update set
	number = excluded.number, url = excluded.url, title = excluded.title, company = excluded.company,
	location = excluded.location, description = excluded.description, tags = excluded.tags,
	work_mode = excluded.work_mode, contract = excluded.contract, seniority = excluded.seniority,
	posted_at = excluded.posted_at, seen_at = excluded.seen_at`

func queueJob(b *pgx.Batch, number int, j job.Job, seen time.Time) {
	tags := j.Tags
	if tags == nil {
		tags = []string{}
	}
	var posted *time.Time
	if !j.PostedAt.IsZero() {
		posted = &j.PostedAt
	}
	b.Queue(upsertJob, j.Key(), number, j.Source, j.SourceID, j.URL, j.Title, j.Company,
		j.Location, j.Description, tags, string(j.WorkMode), string(j.Contract), string(j.Seniority), posted, seen)
}

func (p *Postgres) Sync(run Run) ([]Synced, error) {
	var out []Synced
	err := p.inTx(func(ctx context.Context, tx pgx.Tx) error {
		b, err := loadBook(ctx, tx)
		if err != nil {
			return err
		}
		var changed []int
		out, changed = b.sync(run)
		batch := &pgx.Batch{}
		for _, i := range changed {
			queueEntry(batch, b.Entries[i])
		}
		fresh := 0
		for n, s := range out {
			if s.NewToday {
				fresh++
			}
			queueJob(batch, s.Entry.Number, run.Listed[n].Job, run.At)
		}
		sources := make([]string, 0, len(run.Sources))
		for name, answered := range run.Sources {
			if answered {
				sources = append(sources, name)
			}
		}
		slices.Sort(sources)
		batch.Queue(`insert into runs (at, listed, new_entries, sources) values ($1, $2, $3, $4)`,
			run.At, len(run.Listed), fresh, sources)
		return tx.SendBatch(ctx, batch).Close()
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (p *Postgres) Get(number int) (Entry, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return one(ctx, p.pool, number)
}

func (p *Postgres) Mark(number int, to Status, at time.Time) (Entry, error) {
	var e Entry
	err := p.inTx(func(ctx context.Context, tx pgx.Tx) error {
		var err error
		if e, err = one(ctx, tx, number); err != nil {
			return err
		}
		from := e.Status
		if !e.mark(to, at) {
			return nil
		}
		if _, err := tx.Exec(ctx, `update entries set status = $2 where number = $1`, number, string(to)); err != nil {
			return err
		}
		_, err = tx.Exec(ctx, insertEvent, number, at, string(from), string(to))
		return err
	})
	if err != nil {
		return Entry{}, err
	}
	return e, nil
}

func (p *Postgres) SetNote(number int, note string) (Entry, error) {
	var e Entry
	err := p.inTx(func(ctx context.Context, tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `update entries set note = $2 where number = $1`, number, cleanNote(note))
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return ErrNotFound(number)
		}
		e, err = one(ctx, tx, number)
		return err
	})
	if err != nil {
		return Entry{}, err
	}
	return e, nil
}

func (p *Postgres) List(statuses ...Status) ([]Entry, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return load(ctx, p.pool, nil, statuses)
}

// ErrNotEmpty is returned by Import when the database already has entries.
var ErrNotEmpty = errors.New("o banco já tem vagas marcadas; a importação só roda num banco vazio, para não misturar os números")

// Import copies entries from another store (the local file) into an empty
// database, keeping numbers, statuses, notes and history. It refuses a
// database that already has entries: the numbers would clash.
func (p *Postgres) Import(entries []Entry) error {
	return p.inTx(func(ctx context.Context, tx pgx.Tx) error {
		var have int
		if err := tx.QueryRow(ctx, `select count(*) from entries`).Scan(&have); err != nil {
			return err
		}
		if have > 0 {
			return ErrNotEmpty
		}
		seen := make(map[int]bool)
		batch := &pgx.Batch{}
		for _, e := range entries {
			if e.Number <= 0 || seen[e.Number] {
				return fmt.Errorf("o arquivo tem um número de vaga inválido ou repetido: #%d", e.Number)
			}
			seen[e.Number] = true
			queueEntry(batch, e)
			for _, ev := range e.History {
				batch.Queue(insertEvent, e.Number, ev.At, string(ev.From), string(ev.To))
			}
		}
		return tx.SendBatch(ctx, batch).Close()
	})
}

// Posting returns the text of a posting as of the last run that listed it.
// The second result is false when the database has no text for it, as
// happens with entries that came from the local file.
func (p *Postgres) Posting(number int) (job.Job, bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	var (
		j      job.Job
		posted *time.Time
	)
	err := p.pool.QueryRow(ctx, `
		select source, source_id, url, title, company, location, description, tags, work_mode, contract, seniority, posted_at
		from jobs where number = $1 order by seen_at desc, key limit 1`, number).
		Scan(&j.Source, &j.SourceID, &j.URL, &j.Title, &j.Company, &j.Location, &j.Description, &j.Tags,
			&j.WorkMode, &j.Contract, &j.Seniority, &posted)
	if errors.Is(err, pgx.ErrNoRows) {
		return job.Job{}, false, nil
	}
	if err != nil {
		return job.Job{}, false, err
	}
	if posted != nil {
		j.PostedAt = *posted
	}
	return j, true, nil
}

// PostingReader is implemented by stores that keep the text of the
// postings, not only what was marked.
type PostingReader interface {
	Posting(number int) (job.Job, bool, error)
}
