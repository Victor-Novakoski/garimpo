package tracker

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"net/url"
	"os"
	"testing"

	"github.com/victor-novakoski/garimpo/internal/db"
)

// TestDatabaseEnv points the tests at a Postgres they may fill and drop
// (a local one, or the service the CI starts). Without it, the database
// tests are skipped and the File store still runs the same checks.
const TestDatabaseEnv = "GARIMPO_TEST_DATABASE_URL"

// openTestPostgres returns a store over a fresh schema that goes away with
// the test, so tests never see each other's rows.
func openTestPostgres(t *testing.T) *Postgres {
	t.Helper()
	base := os.Getenv(TestDatabaseEnv)
	if base == "" {
		t.Skipf("%s não definido: teste do Postgres pulado", TestDatabaseEnv)
	}
	ctx := context.Background()
	admin, err := db.Open(ctx, base)
	if err != nil {
		t.Fatal(err)
	}
	schema := fmt.Sprintf("garimpo_test_%d", rand.Uint64())
	if _, err := admin.Exec(ctx, "create schema "+schema); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		admin.Exec(ctx, "drop schema "+schema+" cascade")
		admin.Close()
	})
	u, err := url.Parse(base)
	if err != nil {
		t.Fatal(err)
	}
	q := u.Query()
	q.Set("search_path", schema)
	u.RawQuery = q.Encode()
	pool, err := db.Open(ctx, u.String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	if err := db.Check(ctx, pool); !errors.Is(err, db.ErrNotMigrated) {
		t.Fatalf("check before migrating = %v, want ErrNotMigrated", err)
	}
	if applied, err := db.Migrate(ctx, pool); err != nil || len(applied) == 0 {
		t.Fatalf("migrate = %v, %v", applied, err)
	}
	if applied, err := db.Migrate(ctx, pool); err != nil || len(applied) != 0 {
		t.Fatalf("second migrate = %v, %v; want nothing to apply", applied, err)
	}
	if err := db.Check(ctx, pool); err != nil {
		t.Fatalf("check after migrating: %v", err)
	}
	return OpenPostgres(pool)
}

func TestPostgresKeepsThePosting(t *testing.T) {
	s := openTestPostgres(t)
	a := posting("gh:repo", "1", "Dev Go")
	a.Description, a.Tags, a.WorkMode, a.PostedAt = "Vaga de Go", []string{"go", "remoto"}, "remote", day1
	if _, err := s.Sync(run(day1, Seen{Job: a, Score: 50})); err != nil {
		t.Fatal(err)
	}
	a.Description = "Vaga de Go, atualizada"
	if _, err := s.Sync(run(day2, Seen{Job: a, Score: 50})); err != nil {
		t.Fatal(err)
	}
	got, ok, err := s.Posting(1)
	if err != nil || !ok || got.Description != "Vaga de Go, atualizada" || len(got.Tags) != 2 || got.WorkMode != "remote" || !got.PostedAt.Equal(day1) || got.Key() != a.Key() {
		t.Fatalf("posting = %+v, %v, %v", got, ok, err)
	}
	if _, ok, err := s.Posting(9); ok || err != nil {
		t.Errorf("missing posting = %v, %v", ok, err)
	}
	var runs int
	if err := s.pool.QueryRow(context.Background(), "select count(*) from runs").Scan(&runs); err != nil || runs != 2 {
		t.Errorf("runs = %d, %v", runs, err)
	}
}

func TestPostgresImport(t *testing.T) {
	s := openTestPostgres(t)
	entries := []Entry{
		{Number: 2, Keys: []string{"s:2"}, Title: "B", URL: "https://x/2", Status: StatusNew, FirstSeen: day1, LastSeen: day1},
		{Number: 5, Keys: []string{"s:5", "t:9"}, Title: "E", Company: "Acme", URL: "https://x/5", Status: StatusApplied,
			Note: "falei com a Ana", Score: 61, Reasons: []string{"+20 Go"}, FirstSeen: day1, LastSeen: day2, Gone: true,
			History: []Event{{At: day1, From: StatusNew, To: StatusSaved}, {At: day2, From: StatusSaved, To: StatusApplied}}},
	}
	if err := s.Import(entries); err != nil {
		t.Fatal(err)
	}
	e, err := s.Get(5)
	if err != nil || e.Note != "falei com a Ana" || e.Status != StatusApplied || e.Score != 61 || !e.Gone ||
		len(e.Keys) != 2 || e.Keys[1] != "t:9" || len(e.History) != 2 || e.History[1].To != StatusApplied ||
		!e.Since().Equal(day2) || !e.FirstSeen.Equal(day1) || len(e.Reasons) != 1 {
		t.Fatalf("imported = %+v, %v", e, err)
	}
	// Numbers go on after the highest imported one, and a known key keeps
	// its entry.
	got, err := s.Sync(run(day2, Seen{Job: posting("s", "7", "New")}, Seen{Job: posting("t", "9", "E")}))
	if err != nil || got[0].Entry.Number != 6 || got[1].Entry.Number != 5 || got[1].NewToday || got[1].Entry.Status != StatusApplied {
		t.Fatalf("sync after import = %+v, %v", got, err)
	}
	if err := s.Import(entries); !errors.Is(err, ErrNotEmpty) {
		t.Errorf("second import = %v, want ErrNotEmpty", err)
	}
}

func TestPostgresImportIsAllOrNothing(t *testing.T) {
	s := openTestPostgres(t)
	err := s.Import([]Entry{
		{Number: 1, Keys: []string{"s:1"}, Status: StatusNew, FirstSeen: day1, LastSeen: day1},
		{Number: 1, Keys: []string{"s:2"}, Status: StatusNew, FirstSeen: day1, LastSeen: day1},
	})
	if err == nil {
		t.Fatal("repeated number accepted")
	}
	if all, err := s.List(); err != nil || len(all) != 0 {
		t.Errorf("after a failed import: %+v, %v", all, err)
	}
}
