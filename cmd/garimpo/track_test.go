package main

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/victor-novakoski/garimpo/internal/job"
	"github.com/victor-novakoski/garimpo/internal/score"
	"github.com/victor-novakoski/garimpo/internal/tracker"
)

// day runs the tracking part of a collect: split, (no AI), track.
func day(t *testing.T, store tracker.Store, at time.Time, results []score.Result) (rows []row, log string) {
	t.Helper()
	g := gathered{present: map[string]bool{}, sources: map[string]bool{}}
	for _, r := range results {
		g.present[r.Job.Key()] = true
		g.sources[r.Job.Source] = true
	}
	visible, hidden, err := splitByStatus(store, results)
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	rows, err = track(store, g, at, withoutAI(visible), hidden, false, &buf)
	if err != nil {
		t.Fatal(err)
	}
	return rows, buf.String()
}

func cli(t *testing.T, args ...string) (stdout, stderr string, code int) {
	t.Helper()
	var out, errb bytes.Buffer
	code = run(args, &out, &errb)
	return out.String(), errb.String(), code
}

func TestDailyListFollowsTheFlow(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vagas.json")
	store := tracker.OpenFile(path)
	d1 := time.Date(2026, 11, 11, 9, 0, 0, 0, time.UTC)
	results := []score.Result{
		{Job: job.Job{Source: "s", SourceID: "1", Title: "Dev Go", URL: "https://x/1"}, Points: 50, Reasons: []score.Reason{{Points: 20, Text: "Go no título"}}},
		{Job: job.Job{Source: "s", SourceID: "2", Title: "Dev React", URL: "https://x/2"}, Points: 40},
		{Job: job.Job{Source: "s", SourceID: "3", Title: "Dev Java", URL: "https://x/3"}, Points: 30},
	}
	rows, log := day(t, store, d1, results)
	if len(rows) != 3 || !rows[0].NewToday || rows[0].Number != 1 || !strings.Contains(log, "3 novas hoje") {
		t.Fatalf("day 1: rows = %+v, log = %q", rows, log)
	}

	// The person marks from the terminal.
	now = func() time.Time { return d1.Add(time.Hour) }
	t.Cleanup(func() { now = time.Now })
	if out, errs, code := cli(t, "mark", "-state", path, "1", "apliquei"); code != 0 || !strings.Contains(out, "#1 apliquei: Dev Go") {
		t.Fatalf("mark: %q %q %d", out, errs, code)
	}
	if _, errs, code := cli(t, "mark", "-state", path, "#3", "vou", "aplicar"); code != 0 {
		t.Fatalf("mark two words: %q", errs)
	}
	if out, _, code := cli(t, "note", "-state", path, "3", "pede", "inglês"); code != 0 || !strings.Contains(out, "pede inglês") {
		t.Fatalf("note: %q", out)
	}

	// Day 2: a new posting arrives. Applied is out, saved comes first, the
	// one never marked is still there and no longer "nova hoje".
	results = append(results, score.Result{Job: job.Job{Source: "s", SourceID: "4", Title: "Dev Node", URL: "https://x/4"}, Points: 45})
	rows, log = day(t, store, d1.AddDate(0, 0, 1), results)
	var got []string
	for _, r := range rows {
		got = append(got, r.number()+" "+r.state())
	}
	if want := "#3 vou aplicar|#2 nova|#4 nova hoje"; strings.Join(got, "|") != want {
		t.Errorf("day 2 list = %q, want %q", strings.Join(got, "|"), want)
	}
	if !strings.Contains(log, "1 novas hoje, 1 marcadas como vou aplicar); 1 já aplicadas ou descartadas") {
		t.Errorf("day 2 log = %q", log)
	}
	if rows[0].Note != "pede inglês" {
		t.Errorf("note missing from the list: %+v", rows[0])
	}

	// Follow-up list and detail, without collecting.
	out, _, code := cli(t, "list", "-state", path)
	if code != 0 || !strings.Contains(out, "#1") || !strings.Contains(out, "apliquei") || !strings.Contains(out, "#3") || strings.Contains(out, "#2") {
		t.Errorf("list = %q", out)
	}
	if out, _, _ = cli(t, "list", "-state", path, "nova"); !strings.Contains(out, "#2") || strings.Contains(out, "#1 ") {
		t.Errorf("list nova = %q", out)
	}
	out, _, code = cli(t, "show", "-state", path, "1")
	if code != 0 || !strings.Contains(out, "nova -> apliquei") || !strings.Contains(out, "+20 Go no título") || !strings.Contains(out, "https://x/1") {
		t.Errorf("show = %q", out)
	}
}

func TestTrackCommandsRejectBadInput(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vagas.json")
	for _, args := range [][]string{
		{"mark", "-state", path, "1"},
		{"mark", "-state", path, "x", "apliquei"},
		{"mark", "-state", path, "1", "contratado"},
		{"mark", "-state", "off", "1", "apliquei"},
		{"show", "-state", path},
		{"list", "-state", path, "contratado"},
	} {
		if _, _, code := cli(t, args...); code != 2 {
			t.Errorf("%v: code = %d, want 2", args, code)
		}
	}
	if _, errs, code := cli(t, "mark", "-state", path, "7", "apliquei"); code != 1 || !strings.Contains(errs, "não existe vaga #7") {
		t.Errorf("unknown number: %q %d", errs, code)
	}
}

func TestTrackingOff(t *testing.T) {
	results := []score.Result{{Job: job.Job{Source: "s", SourceID: "1", Title: "Dev Go"}, Points: 50}}
	visible, hidden, err := splitByStatus(nil, results)
	if err != nil || len(visible) != 1 || len(hidden) != 0 {
		t.Fatal("split without a store")
	}
	rows, err := track(nil, gathered{}, time.Now(), withoutAI(visible), nil, false, &bytes.Buffer{})
	if err != nil || len(rows) != 1 || rows[0].number() != "-" || rows[0].state() != "" {
		t.Fatalf("rows = %+v", rows)
	}
}
