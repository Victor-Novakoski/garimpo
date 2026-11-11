package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/victor-novakoski/garimpo/internal/api"
	"github.com/victor-novakoski/garimpo/internal/collector"
	"github.com/victor-novakoski/garimpo/internal/job"
	"github.com/victor-novakoski/garimpo/internal/tracker"
)

func TestParseServe(t *testing.T) {
	o, err := parseServe([]string{"-port", "9000", "-ai", "local", "-gupy"}, &bytes.Buffer{})
	if err != nil || o.port != 9000 || o.ai != "local" || !o.gupy {
		t.Fatalf("o = %+v, err = %v", o, err)
	}
	for _, args := range [][]string{
		{"-port", "0"}, {"-port", "70000"}, {"-state", "off"}, {"-ai", "gpt"}, {"sobra"},
		// The panel never leaves the machine: there is no flag for the address.
		{"-addr", "0.0.0.0:8080"}, {"-host", "0.0.0.0"},
	} {
		if _, err := parseServe(args, &bytes.Buffer{}); err == nil {
			t.Errorf("parseServe(%v) accepted", args)
		}
	}
}

// call makes a request the way the panel's own page does.
func call(t *testing.T, srv *httptest.Server, method, path string) (int, map[string]any) {
	t.Helper()
	req, err := http.NewRequest(method, srv.URL+path, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Origin", srv.URL)
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	res, err := srv.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	var out map[string]any
	if strings.HasPrefix(res.Header.Get("Content-Type"), "application/json") {
		if err := json.Unmarshal(body, &out); err != nil {
			t.Fatalf("%s %s: not JSON: %q", method, path, body)
		}
	}
	return res.StatusCode, out
}

func TestServeCollectsFromThePanel(t *testing.T) {
	old := sources
	t.Cleanup(func() { sources = old })
	sources = func(collectOptions, time.Time) []collector.Collector {
		return []collector.Collector{fakeCollector{name: "a", jobs: []job.Job{
			{Source: "a", SourceID: "1", Company: "Acme", Title: "[Remoto] Dev Go Pleno", URL: "https://x/1", Location: "Brasil"},
			{Source: "a", SourceID: "2", Company: "Beta", Title: "[Presencial] Dev React Pleno", URL: "https://x/2"},
		}}}
	}
	store := tracker.OpenFile(filepath.Join(t.TempDir(), "vagas.json"))
	var terminal bytes.Buffer
	o := collectOptions{profile: "../../profile.example.yaml", state: "file"}
	runner := api.NewRunner(context.Background(), collectRun(o, store, &terminal))
	srv := httptest.NewServer(serveHandler(store, api.Options{Collect: runner}))
	defer srv.Close()

	// The panel and the API come from the same address, without a token.
	res, err := srv.Client().Get(srv.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != 200 && res.StatusCode != 503 { // 503: binary built without the panel
		t.Errorf("panel = %d", res.StatusCode)
	}
	if code, out := call(t, srv, "GET", "/api/jobs"); code != 200 || len(out["jobs"].([]any)) != 0 {
		t.Fatalf("empty list = %d %v", code, out)
	}
	// The routes are only under /api/.
	if code, out := call(t, srv, "GET", "/jobs"); out["jobs"] != nil {
		t.Errorf("/jobs outside /api = %d %v", code, out)
	}

	if code, out := call(t, srv, "POST", "/api/collect"); code != 202 {
		t.Fatalf("start = %d %v", code, out)
	}
	runner.Wait()
	code, out := call(t, srv, "GET", "/api/collect")
	if code != 200 || out["running"] != false || out["error"] != nil {
		t.Fatalf("after the collection = %d %v", code, out)
	}
	for _, want := range []string{"a ", "2 vagas", "1 cortadas pelo filtro", "1 novas hoje"} {
		if !strings.Contains(out["log"].(string), want) {
			t.Errorf("log %q missing %q", out["log"], want)
		}
	}
	_, out = call(t, srv, "GET", "/api/jobs")
	jobs := out["jobs"].([]any)
	if len(jobs) != 1 {
		t.Fatalf("list after the collection = %v", out)
	}
	if got := jobs[0].(map[string]any); got["title"] != "[Remoto] Dev Go Pleno" || got["work_mode"] != "remote" || got["new_today"] != true || got["location"] != "Brasil" {
		t.Errorf("listed = %v", got)
	}
	if code, out := call(t, srv, "PATCH", "/api/jobs/1"); code != 400 {
		t.Errorf("patch without a body = %d %v", code, out)
	}
}

func TestServeCollectFailuresAreReadable(t *testing.T) {
	old := sources
	t.Cleanup(func() { sources = old })
	sources = func(collectOptions, time.Time) []collector.Collector {
		return []collector.Collector{fakeCollector{name: "a", err: io.ErrUnexpectedEOF}}
	}
	store := tracker.OpenFile(filepath.Join(t.TempDir(), "vagas.json"))
	var log bytes.Buffer
	// Every source down: nothing is recorded, so nothing is marked as gone.
	err := collectRun(collectOptions{profile: "../../profile.example.yaml"}, store, io.Discard)(context.Background(), &log)
	if err == nil || !strings.Contains(err.Error(), "nenhuma fonte respondeu") {
		t.Errorf("all sources down = %v", err)
	}
	// No profile: the sentence tells what to do.
	err = collectRun(collectOptions{}, store, io.Discard)(context.Background(), &log)
	if err == nil || !strings.Contains(err.Error(), "profile.example.yaml") {
		t.Errorf("no profile = %v", err)
	}
	// A store that fails goes to the terminal, not to the panel.
	sources = func(collectOptions, time.Time) []collector.Collector {
		return []collector.Collector{fakeCollector{name: "a", jobs: []job.Job{{Source: "a", SourceID: "1", Title: "Dev Go Pleno", URL: "https://x/1"}}}}
	}
	path := filepath.Join(t.TempDir(), "vagas.json")
	if err := os.WriteFile(path, []byte("{broken"), 0o600); err != nil {
		t.Fatal(err)
	}
	var terminal bytes.Buffer
	err = collectRun(collectOptions{profile: "../../profile.example.yaml"}, tracker.OpenFile(path), &terminal)(context.Background(), &log)
	if err == nil || strings.Contains(err.Error(), path) || !strings.Contains(terminal.String(), "corrompido") {
		t.Errorf("broken store: err = %v, terminal = %q", err, terminal.String())
	}
}
