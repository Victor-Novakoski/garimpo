package main

import (
	"bytes"
	"context"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/victor-novakoski/garimpo/internal/tracker"
)

var demoDay = time.Date(2026, 11, 16, 9, 0, 0, 0, time.UTC)

func openTestDemo(t *testing.T) tracker.Store {
	t.Helper()
	// The demo must not need the owner's database: with the variable
	// pointing nowhere, opening it would fail.
	t.Setenv("GARIMPO_DATABASE_URL", "postgres://ninguem@127.0.0.1:1/nada")
	var terminal bytes.Buffer
	store, err := openDemo(context.Background(), "../../"+demoJobs, "../../"+demoProfile, demoDay, &terminal)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(terminal.String(), "modo demo") {
		t.Errorf("terminal = %q", terminal.String())
	}
	return store
}

// The file in the repository is what the README's screenshots show: every
// state of the funnel, something new today, the AI reading, and nothing
// that points to a real posting.
func TestDemoShowsEveryStateWithMadeUpPostings(t *testing.T) {
	store := openTestDemo(t)
	all, err := store.List()
	if err != nil {
		t.Fatal(err)
	}
	count := map[tracker.Status]int{}
	withAI := 0
	for _, e := range all {
		count[e.Status]++
		if !strings.HasPrefix(e.URL, "https://example.com/") {
			t.Errorf("#%d points outside example.com: %s", e.Number, e.URL)
		}
		for _, k := range e.Keys {
			if !strings.HasPrefix(k, demoSource+":") {
				t.Errorf("#%d has a key from a real source: %s", e.Number, k)
			}
		}
		if strings.Contains(strings.Join(e.Reasons, "\n"), "IA, encaixe") {
			withAI++
		}
		if e.LastSeen.After(demoDay) || e.Since().After(demoDay) {
			t.Errorf("#%d has a date in the future: %+v", e.Number, e)
		}
	}
	for _, s := range tracker.Statuses() {
		if count[s] == 0 {
			t.Errorf("no posting in state %q", s)
		}
	}
	if fresh := tracker.NewToday(all); len(fresh) == 0 || len(fresh) == len(all) {
		t.Errorf("%d of %d postings are new today", len(fresh), len(all))
	}
	if withAI == 0 {
		t.Error("no posting has the AI reading")
	}
	// The duplicate was folded into the first posting, and the ones the
	// example profile refuses are not listed.
	if first, _ := store.Get(1); len(first.Keys) != 2 || first.Note == "" || len(first.History) == 0 {
		t.Errorf("#1 = %+v", first)
	}
	for _, e := range all {
		if strings.Contains(e.Title, "Presencial") || strings.Contains(e.Company, "Logística") {
			t.Errorf("#%d should have been dropped by the filter: %s, %s", e.Number, e.Title, e.Company)
		}
	}
}

func TestDemoServesThePanelWithoutCollecting(t *testing.T) {
	srv := httptest.NewServer(serveHandler(openTestDemo(t), demoOptions))
	defer srv.Close()

	code, out := call(t, srv, "GET", "/api/jobs")
	if code != 200 || len(out["jobs"].([]any)) == 0 {
		t.Fatalf("daily list = %d %v", code, out)
	}
	if first := out["jobs"].([]any)[0].(map[string]any); first["status"] != "saved" {
		t.Errorf("first of the daily list = %v", first)
	}
	// The text of the posting is there, as with the database.
	_, out = call(t, srv, "GET", "/api/jobs/1")
	if p, _ := out["posting"].(map[string]any); p == nil || p["description"] == "" || p["source"] != demoSource {
		t.Errorf("posting = %v", out["posting"])
	}
	code, out = call(t, srv, "GET", "/api/search")
	if code != 200 || out["available"] != false || out["reason"] != demoOptions.CollectOff {
		t.Errorf("collect = %d %v", code, out)
	}
	if code, _ := call(t, srv, "POST", "/api/search"); code < 400 {
		t.Errorf("the demo started a collection: %d", code)
	}
}

func TestDemoRefusesABrokenFile(t *testing.T) {
	for name, content := range map[string]string{
		"unknown field":                      `{"postings":[{"id":"1","salary":10}]}`,
		"repeated id":                        `{"postings":[{"id":"1","title":"Dev Go"},{"id":"1","title":"Dev React"}]}`,
		"bad AI answer":                      `{"postings":[{"id":"1","title":"[Remoto] Dev Go Pleno","ai":{"fit":"perfeito"}}]}`,
		"bad status":                         `{"postings":[{"id":"1","title":"[Remoto] Dev Go Pleno","marks":[{"to":"contratado"}]}]}`,
		"mark on a posting the filter drops": `{"postings":[{"id":"1","title":"[Presencial] Dev Go Pleno","marks":[{"to":"apliquei"}]}]}`,
	} {
		path := filepath.Join(t.TempDir(), "vagas.json")
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := openDemo(context.Background(), path, "../../"+demoProfile, demoDay, &bytes.Buffer{}); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
	if _, err := openDemo(context.Background(), "nao-existe.json", "../../"+demoProfile, demoDay, &bytes.Buffer{}); err == nil || !strings.Contains(err.Error(), "pasta do projeto") {
		t.Errorf("missing file: %v", err)
	}
}
