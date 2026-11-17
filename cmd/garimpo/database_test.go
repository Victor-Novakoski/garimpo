package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/victor-novakoski/garimpo/internal/db"
	"github.com/victor-novakoski/garimpo/internal/db/dbtest"
	"github.com/victor-novakoski/garimpo/internal/job"
	"github.com/victor-novakoski/garimpo/internal/score"
	"github.com/victor-novakoski/garimpo/internal/tracker"
)

func TestDatabaseCommandsNeedTheURL(t *testing.T) {
	t.Setenv(db.URLEnv, "")
	path := filepath.Join(t.TempDir(), "vagas.json")
	if _, err := tracker.OpenFile(path).Sync(tracker.Run{At: time.Now(), Listed: []tracker.Seen{{Job: job.Job{Source: "s", SourceID: "1"}}}}); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"migrate"}, {"import", "-from", path}, {"list", "-state", "db"}} {
		if _, errs, code := cli(t, args...); code != 1 || !strings.Contains(errs, db.URLEnv) {
			t.Errorf("%v: %q %d", args, errs, code)
		}
	}
	// A URL that does not parse is reported without being echoed.
	t.Setenv(db.URLEnv, "postgres://user:segredo@host:porta/banco")
	if _, errs, code := cli(t, "migrate"); code != 1 || strings.Contains(errs, "segredo") {
		t.Errorf("bad URL: %q %d", errs, code)
	}
}

// The move from the file to the database: mark in the file, migrate,
// import, and go on from the database with the same numbers.
func TestMoveFromFileToDatabase(t *testing.T) {
	t.Setenv(db.URLEnv, dbtest.URL(t))
	path := filepath.Join(t.TempDir(), "vagas.json")
	d1 := time.Date(2026, 11, 11, 9, 0, 0, 0, time.UTC)
	now = func() time.Time { return d1.Add(time.Hour) }
	t.Cleanup(func() { now = time.Now })
	results := []score.Result{
		{Job: job.Job{Source: "s", SourceID: "1", Title: "Dev Go", URL: "https://x/1"}, Points: 50},
		{Job: job.Job{Source: "s", SourceID: "2", Title: "Dev React", URL: "https://x/2"}, Points: 40},
	}
	day(t, tracker.OpenFile(path), d1, results)
	if _, errs, code := cli(t, "mark", "-state", path, "2", "apliquei"); code != 0 {
		t.Fatalf("mark in the file: %q", errs)
	}
	before, _ := os.ReadFile(path)

	if _, errs, code := cli(t, "list"); code != 1 || !strings.Contains(errs, "garimpo migrate") {
		t.Fatalf("list before migrate: %q %d", errs, code)
	}
	if out, errs, code := cli(t, "migrate"); code != 0 || !strings.Contains(out, "0001_init") {
		t.Fatalf("migrate: %q %q %d", out, errs, code)
	}
	if out, _, code := cli(t, "migrate"); code != 0 || !strings.Contains(out, "em dia") {
		t.Fatalf("second migrate: %q %d", out, code)
	}
	if out, errs, code := cli(t, "import", "-from", path); code != 0 || !strings.Contains(out, "2 vagas importadas") || !strings.Contains(out, "(1 com estado") {
		t.Fatalf("import: %q %q %d", out, errs, code)
	}
	if _, errs, code := cli(t, "import", "-from", path); code != 1 || !strings.Contains(errs, "banco vazio") {
		t.Fatalf("second import: %q %d", errs, code)
	}
	if after, _ := os.ReadFile(path); string(after) != string(before) {
		t.Error("import changed the file")
	}

	// With the URL set, the commands use the database without any flag.
	if out, errs, code := cli(t, "show", "2"); code != 0 || !strings.Contains(out, "nova -> apliquei") {
		t.Fatalf("show from the database: %q %q %d", out, errs, code)
	}
	if out, errs, code := cli(t, "mark", "1", "vou", "aplicar"); code != 0 || !strings.Contains(out, "#1 vou aplicar") {
		t.Fatalf("mark in the database: %q %q %d", out, errs, code)
	}
	// The file did not follow: -state file still reads it.
	if out, _, _ := cli(t, "list", "-state", path, "nova"); !strings.Contains(out, "#1") {
		t.Errorf("file after marking in the database: %q", out)
	}

	// Next day's run goes to the database and keeps numbers and statuses.
	store, done, err := tracker.Open(t.Context(), "")
	if err != nil {
		t.Fatal(err)
	}
	defer done()
	results = append(results, score.Result{Job: job.Job{Source: "s", SourceID: "3", Title: "Dev Node", URL: "https://x/3"}, Points: 45})
	rows, _ := day(t, store, d1.AddDate(0, 0, 1), results)
	var got []string
	for _, r := range rows {
		got = append(got, r.number()+" "+r.state())
	}
	if want := "#1 vou aplicar|#3 nova hoje"; strings.Join(got, "|") != want {
		t.Errorf("list from the database = %q, want %q", strings.Join(got, "|"), want)
	}
}
