package tracker

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/victor-novakoski/garimpo/internal/job"
)

var (
	day1 = time.Date(2026, 11, 11, 9, 0, 0, 0, time.UTC)
	day2 = day1.AddDate(0, 0, 1)
)

func posting(source, id, title string) job.Job {
	return job.Job{Source: source, SourceID: id, Title: title, URL: "https://x/" + id}
}

// run builds a Run where every listed posting is present and its source answered.
func run(at time.Time, listed ...Seen) Run {
	r := Run{At: at, Listed: listed, Present: map[string]bool{}, Sources: map[string]bool{}}
	for _, s := range listed {
		r.Present[s.Job.Key()] = true
		r.Sources[s.Job.Source] = true
	}
	return r
}

// eachStore runs a test against every Store implementation: they must
// behave the same.
func eachStore(t *testing.T, test func(t *testing.T, s Store)) {
	t.Run("file", func(t *testing.T) {
		test(t, OpenFile(filepath.Join(t.TempDir(), "sub", "vagas.json")))
	})
}

func TestParseStatus(t *testing.T) {
	for text, want := range map[string]Status{
		"apliquei": StatusApplied, "APLIQUEI": StatusApplied, "applied": StatusApplied,
		"descartei": StatusDiscarded, "descartada": StatusDiscarded,
		"depois": StatusSaved, "Vou  Aplicar": StatusSaved,
		"entrevista": StatusInterview, "nova": StatusNew,
	} {
		if got, err := ParseStatus(text); err != nil || got != want {
			t.Errorf("ParseStatus(%q) = %q, %v; want %q", text, got, err, want)
		}
	}
	if _, err := ParseStatus("contratado"); err == nil {
		t.Error("unknown status accepted")
	}
	if !StatusNew.InDailyList() || !StatusSaved.InDailyList() || StatusApplied.InDailyList() || StatusDiscarded.InDailyList() {
		t.Error("daily list statuses are wrong")
	}
}

func TestSyncKeepsNumbersAndStatus(t *testing.T) { eachStore(t, testSyncKeepsNumbersAndStatus) }

func testSyncKeepsNumbersAndStatus(t *testing.T, s Store) {
	a, b := posting("gh:repo", "1", "Dev Go"), posting("gh:repo", "2", "Dev React")
	got, err := s.Sync(run(day1, Seen{Job: a, Score: 50, Reasons: []string{"+20 Go"}}, Seen{Job: b, Score: 40}))
	if err != nil {
		t.Fatal(err)
	}
	if got[0].Entry.Number != 1 || got[1].Entry.Number != 2 || !got[0].NewToday || got[0].Entry.Status != StatusNew {
		t.Fatalf("first run = %+v", got)
	}
	if _, err := s.Mark(1, StatusApplied, day1); err != nil {
		t.Fatal(err)
	}

	// Next day: a new posting first, the title of #2 edited, #1 still there.
	c := posting("gh:repo", "3", "Dev Node")
	b.Title = "Dev React Pleno"
	got, err = s.Sync(run(day2, Seen{Job: c}, Seen{Job: b, Score: 45}, Seen{Job: a, Score: -1}))
	if err != nil {
		t.Fatal(err)
	}
	if got[0].Entry.Number != 3 || !got[0].NewToday {
		t.Errorf("new posting = %+v", got[0])
	}
	if e := got[1].Entry; e.Number != 2 || got[1].NewToday || e.Title != "Dev React Pleno" || e.Score != 45 || !e.FirstSeen.Equal(day1) || !e.LastSeen.Equal(day2) {
		t.Errorf("seen again = %+v", got[1])
	}
	// A negative score keeps the score and the reasons on record.
	if e := got[2].Entry; e.Number != 1 || e.Status != StatusApplied || e.Score != 50 || len(e.Reasons) != 1 || e.Reasons[0] != "+20 Go" {
		t.Errorf("status or score lost on sync: %+v", e)
	}
}

func TestSyncCopiesAreOneEntry(t *testing.T) { eachStore(t, testSyncCopiesAreOneEntry) }

func testSyncCopiesAreOneEntry(t *testing.T, s Store) {
	first := posting("gh:backend", "10", "Dev Go")
	if _, err := s.Sync(run(day1, Seen{Job: first, Copies: []string{"gh:frontend:77"}})); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Mark(1, StatusSaved, day1); err != nil {
		t.Fatal(err)
	}
	// The original is gone and the copy is now the one listed: same entry.
	copy := posting("gh:frontend", "77", "Dev Go")
	got, err := s.Sync(run(day2, Seen{Job: copy}))
	if err != nil {
		t.Fatal(err)
	}
	if got[0].Entry.Number != 1 || got[0].NewToday || got[0].Entry.Status != StatusSaved {
		t.Fatalf("copy became a new entry: %+v", got[0])
	}
}

func TestSyncGone(t *testing.T) { eachStore(t, testSyncGone) }

func testSyncGone(t *testing.T, s Store) {
	a, b, c := posting("gh:repo", "1", "A"), posting("green:x", "2", "B"), posting("gh:repo", "3", "C")
	if _, err := s.Sync(run(day1, Seen{Job: a}, Seen{Job: b}, Seen{Job: c})); err != nil {
		t.Fatal(err)
	}
	// Day 2: gh:repo answered with only C listed; A is still in the source
	// but filtered out; green:x failed, so B's fate is unknown.
	r := run(day2, Seen{Job: c})
	r.Present[a.Key()] = true
	if _, err := s.Sync(r); err != nil {
		t.Fatal(err)
	}
	all, _ := s.List()
	if all[0].Gone || all[1].Gone || all[2].Gone {
		t.Fatalf("gone too early: %+v", all)
	}
	// Day 3: gh:repo answers without A.
	if _, err := s.Sync(run(day2.AddDate(0, 0, 1), Seen{Job: c})); err != nil {
		t.Fatal(err)
	}
	all, _ = s.List()
	if !all[0].Gone || all[1].Gone || all[2].Gone {
		t.Fatalf("gone = %v %v %v, want true false false", all[0].Gone, all[1].Gone, all[2].Gone)
	}
	// It comes back: no longer gone.
	if _, err := s.Sync(run(day2.AddDate(0, 0, 2), Seen{Job: a}, Seen{Job: c})); err != nil {
		t.Fatal(err)
	}
	if e, _ := s.Get(1); e.Gone {
		t.Error("posting that came back is still gone")
	}
}

func TestMarkNoteListHistory(t *testing.T) { eachStore(t, testMarkNoteListHistory) }

func testMarkNoteListHistory(t *testing.T, s Store) {
	if _, err := s.Sync(run(day1, Seen{Job: posting("s", "1", "A")}, Seen{Job: posting("s", "2", "B")})); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Mark(1, StatusApplied, day1); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Mark(1, StatusApplied, day2); err != nil { // same status: no new event
		t.Fatal(err)
	}
	e, err := s.Mark(1, StatusInterview, day2)
	if err != nil || len(e.History) != 2 || e.History[1].From != StatusApplied || !e.Since().Equal(day2) {
		t.Fatalf("history = %+v, err = %v", e.History, err)
	}
	if e, err := s.SetNote(1, "  falei com a Ana  "); err != nil || e.Note != "falei com a Ana" {
		t.Fatalf("note = %q, err = %v", e.Note, err)
	}
	if e, err := s.Get(1); err != nil || e.Note != "falei com a Ana" || e.Status != StatusInterview || len(e.History) != 2 || len(e.Keys) != 1 || e.Keys[0] != "s:1" {
		t.Fatalf("get = %+v, err = %v", e, err)
	}
	if e, err := s.SetNote(1, ""); err != nil || e.Note != "" {
		t.Fatalf("cleared note = %q, err = %v", e.Note, err)
	}
	var notFound ErrNotFound
	if _, err := s.Mark(9, StatusApplied, day1); !errors.As(err, &notFound) {
		t.Errorf("mark missing number: %v", err)
	}
	if _, err := s.SetNote(9, "x"); !errors.As(err, &notFound) {
		t.Errorf("note missing number: %v", err)
	}
	if _, err := s.Get(9); !errors.As(err, &notFound) {
		t.Errorf("get missing number: %v", err)
	}
	list, err := s.List(StatusInterview, StatusApplied)
	if err != nil || len(list) != 1 || list[0].Number != 1 {
		t.Fatalf("list = %+v, err = %v", list, err)
	}
	if all, err := s.List(); err != nil || len(all) != 2 || all[0].Number != 1 || all[1].Number != 2 {
		t.Fatalf("list all = %+v, err = %v", all, err)
	}
}

func TestNewToday(t *testing.T) {
	entries := []Entry{
		{Number: 1, FirstSeen: day1, LastSeen: day2},
		{Number: 2, FirstSeen: day2, LastSeen: day2},
		{Number: 3, FirstSeen: day1, LastSeen: day1}, // gone before the latest run
	}
	if fresh := NewToday(entries); len(fresh) != 1 || !fresh[2] {
		t.Errorf("NewToday = %v, want only #2", fresh)
	}
	if fresh := NewToday(nil); len(fresh) != 0 {
		t.Errorf("NewToday(nil) = %v", fresh)
	}
}

func TestFileIsPrivateAndReopens(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vagas.json")
	s := OpenFile(path)
	if _, err := s.Sync(run(day1, Seen{Job: posting("s", "1", "A")})); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SetNote(1, "falei com a Ana"); err != nil {
		t.Fatal(err)
	}
	// A new handle reads the same file.
	list, err := OpenFile(path).List()
	if err != nil || len(list) != 1 || list[0].Note == "" {
		t.Fatalf("list = %+v, err = %v", list, err)
	}
	if info, _ := os.Stat(path); info.Mode().Perm() != 0o600 {
		t.Errorf("file mode = %v", info.Mode().Perm())
	}
}

func TestCorruptFileIsNotOverwritten(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vagas.json")
	if err := os.WriteFile(path, []byte("{broken"), 0o600); err != nil {
		t.Fatal(err)
	}
	s := OpenFile(path)
	if _, err := s.Sync(run(day1, Seen{Job: posting("s", "1", "A")})); err == nil {
		t.Fatal("sync over a corrupt file succeeded")
	}
	if data, _ := os.ReadFile(path); string(data) != "{broken" {
		t.Error("corrupt file was overwritten")
	}
}
