package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/victor-novakoski/garimpo/internal/ai"
	"github.com/victor-novakoski/garimpo/internal/collector"
	"github.com/victor-novakoski/garimpo/internal/job"
	"github.com/victor-novakoski/garimpo/internal/profile"
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
	g := gather(context.Background(), cs, false, &log)
	jobs := g.jobs
	if len(jobs) != 1 {
		t.Fatalf("got %d jobs, want 1", len(jobs))
	}
	if jobs[0].WorkMode != job.WorkModeRemote || jobs[0].Seniority != job.SeniorityMid {
		t.Errorf("not classified: %+v", jobs[0])
	}
	if copies := g.copies["a:1"]; len(copies) != 1 || copies[0] != "b:7" {
		t.Errorf("copies = %v", g.copies)
	}
	if !g.present["c:3"] || !g.sources["a"] || g.sources["broken"] {
		t.Errorf("present = %v, sources = %v", g.present, g.sources)
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
	results := rowsOf(withoutAI([]score.Result{{
		Job:     job.Job{Title: "Dev Go", URL: "https://x/1"},
		Points:  25,
		Reasons: []score.Reason{{Points: 15, Text: "pede Go"}},
	}}))
	var out bytes.Buffer
	if err := printResults(&out, results, score.Max, false, true); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"25/70", "Dev Go", "+15 pede Go"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("output %q missing %q", out.String(), want)
		}
	}
	if strings.Contains(out.String(), "IA") {
		t.Errorf("AI line without -ai: %q", out.String())
	}
}

func rowsOf(ranked []ai.Ranked) []row {
	rows := make([]row, len(ranked))
	for i, r := range ranked {
		rows[i] = row{Ranked: r}
	}
	return rows
}

// fakeScorer answers by title, without any model.
type fakeScorer struct{ err error }

func (fakeScorer) Name() string { return "fake" }
func (f fakeScorer) Assess(_ context.Context, j job.Job) (ai.Assessment, error) {
	if f.err != nil {
		return ai.Assessment{}, f.err
	}
	if strings.Contains(j.Title, "React") {
		return ai.Assessment{Fit: ai.FitStrong, Summary: "bate com o perfil", Gaps: []string{"inglês fluente"}}, nil
	}
	return ai.Assessment{Fit: ai.FitWeak, Summary: "stack diferente"}, nil
}

func useScorer(t *testing.T, s ai.Scorer) {
	t.Helper()
	old := newScorer
	newScorer = func(collectOptions, profile.Profile) ai.Scorer { return s }
	t.Cleanup(func() { newScorer = old })
}

func TestAssessReordersAndExplains(t *testing.T) {
	useScorer(t, fakeScorer{})
	results := []score.Result{
		{Job: job.Job{Source: "a", SourceID: "1", Title: "Dev Go"}, Points: 40},
		{Job: job.Job{Source: "a", SourceID: "2", Title: "Dev React"}, Points: 35},
		{Job: job.Job{Source: "a", SourceID: "3", Title: "Dev Java"}, Points: 10},
	}
	o := collectOptions{ai: "local", aiTop: 2, aiCache: "off"}
	var log bytes.Buffer
	ranked := assess(context.Background(), o, profile.Profile{}, results, &log)
	// React: 35 + 30 = 65; Go: 40 + 6 = 46; Java was not assessed.
	if ranked[0].Job.Title != "Dev React" || ranked[0].Total != 65 || ranked[1].Total != 46 {
		t.Fatalf("ranked = %+v", ranked)
	}
	if ranked[2].AI != nil || ranked[2].Total != 10 {
		t.Errorf("job beyond -ai-top was assessed: %+v", ranked[2])
	}
	if !strings.Contains(log.String(), "2 avaliadas, 0 com erro") {
		t.Errorf("log = %q", log.String())
	}
	var out bytes.Buffer
	if err := printResults(&out, rowsOf(ranked), ai.MaxTotal, false, true); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"65/100", "+30 IA, encaixe forte: bate com o perfil", "falta: inglês fluente", "fora das melhores"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("output missing %q:\n%s", want, out.String())
		}
	}
}

func TestAssessUnavailableKeepsRuleScores(t *testing.T) {
	useScorer(t, fakeScorer{err: fmt.Errorf("%w: sem login", ai.ErrUnavailable)})
	results := []score.Result{{Job: job.Job{Title: "Dev Go"}, Points: 40}}
	var log bytes.Buffer
	ranked := assess(context.Background(), collectOptions{ai: "local", aiTop: 5, aiCache: "off"}, profile.Profile{}, results, &log)
	if len(ranked) != 1 || ranked[0].Total != 40 || ranked[0].AI != nil {
		t.Fatalf("ranked = %+v", ranked)
	}
	if !strings.Contains(log.String(), "IA parou") || !strings.Contains(log.String(), "sem login") {
		t.Errorf("log = %q", log.String())
	}
}

func TestParseCollectAI(t *testing.T) {
	if _, err := parseCollect([]string{"-ai", "gpt"}, &bytes.Buffer{}); err == nil {
		t.Fatal("unknown -ai value accepted")
	}
	o, err := parseCollect([]string{"-ai", "local"}, &bytes.Buffer{})
	if err != nil || o.ai != "local" || o.aiTop != 20 {
		t.Fatalf("o = %+v, err = %v", o, err)
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
	rs := rowsOf([]ai.Ranked{{Total: 60}, {Total: 30}})
	if got := atLeast(rs, 50); len(got) != 1 || got[0].Total != 60 {
		t.Fatalf("atLeast = %v", got)
	}
}
