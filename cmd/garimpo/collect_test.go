package main

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/victor-novakoski/garimpo/internal/collector"
	"github.com/victor-novakoski/garimpo/internal/job"
	"github.com/victor-novakoski/garimpo/internal/score"
)

type fakeCollector struct {
	name string
	jobs []job.Job
	err  error
}

func (f fakeCollector) Name() string { return f.name }
func (f fakeCollector) Collect(context.Context) ([]job.Job, error) {
	return f.jobs, f.err
}

func TestGatherClassifiesDedupesAndReports(t *testing.T) {
	cs := []collector.Collector{
		fakeCollector{name: "a", jobs: []job.Job{
			{Source: "a", SourceID: "1", Company: "Acme", Title: "[Remoto] Dev Go Pleno"},
		}},
		fakeCollector{name: "b", jobs: []job.Job{
			{Source: "b", SourceID: "7", Company: "acme", Title: "[remoto] dev go pleno"},
		}},
		fakeCollector{name: "c", jobs: []job.Job{
			{Source: "c", SourceID: "3", Company: "Acme", Title: "Analista Comercial"},
		}},
		fakeCollector{name: "broken", err: errors.New("fora do ar")},
	}
	var log bytes.Buffer
	jobs := gather(context.Background(), cs, false, &log)
	if len(jobs) != 1 {
		t.Fatalf("got %d jobs, want 1", len(jobs))
	}
	if jobs[0].WorkMode != job.WorkModeRemote || jobs[0].Seniority != job.SeniorityMid {
		t.Errorf("not classified: %+v", jobs[0])
	}
	for _, want := range []string{"broken", "erro: fora do ar", "1 fora de tecnologia", "1 repetidas removidas"} {
		if !strings.Contains(log.String(), want) {
			t.Errorf("log %q missing %q", log.String(), want)
		}
	}
}

func TestCollectorsGupyOffByDefault(t *testing.T) {
	o, err := parseCollect(nil, &bytes.Buffer{})
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range collectors(o, time.Now()) {
		if c.Name() == "gupy" {
			t.Fatal("gupy collector present without -gupy")
		}
	}
	o, _ = parseCollect([]string{"-gupy"}, &bytes.Buffer{})
	found := false
	for _, c := range collectors(o, time.Now()) {
		found = found || c.Name() == "gupy"
	}
	if !found {
		t.Fatal("gupy collector missing with -gupy")
	}
}

func TestPrintJobs(t *testing.T) {
	jobs := []job.Job{{Title: "Dev Go", URL: "https://x/1", WorkMode: job.WorkModeRemote}}
	var out bytes.Buffer
	if err := printJobs(&out, jobs, false); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "Dev Go") || !strings.Contains(out.String(), "remote") {
		t.Errorf("table = %q", out.String())
	}
	out.Reset()
	if err := printJobs(&out, jobs, true); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), `"Title":"Dev Go"`) {
		t.Errorf("json = %q", out.String())
	}
}

func TestParseCollectRejectsExtraArgs(t *testing.T) {
	if _, err := parseCollect([]string{"nope"}, &bytes.Buffer{}); err == nil {
		t.Fatal("want error")
	}
}

func TestPrintResultsWhy(t *testing.T) {
	results := []score.Result{{
		Job:     job.Job{Title: "Dev Go", URL: "https://x/1"},
		Points:  25,
		Reasons: []score.Reason{{Points: 15, Text: "pede Go"}},
	}}
	var out bytes.Buffer
	if err := printResults(&out, results, false, true); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"25/70", "Dev Go", "+15 pede Go"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("output %q missing %q", out.String(), want)
		}
	}
}

func TestLoadProfile(t *testing.T) {
	if _, ok, err := loadProfile(""); ok || err != nil {
		t.Fatalf("no profile.yaml here: ok=%v err=%v", ok, err)
	}
	p, ok, err := loadProfile("../../profile.example.yaml")
	if !ok || err != nil || p.Seniority != job.SeniorityMid {
		t.Fatalf("example: ok=%v err=%v", ok, err)
	}
	if _, _, err := loadProfile("nope.yaml"); err == nil {
		t.Fatal("missing explicit profile should fail")
	}
}

func TestAtLeast(t *testing.T) {
	rs := []score.Result{{Points: 60}, {Points: 30}}
	if got := atLeast(rs, 50); len(got) != 1 || got[0].Points != 60 {
		t.Fatalf("atLeast = %v", got)
	}
}
