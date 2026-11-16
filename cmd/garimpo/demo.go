package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"slices"
	"time"

	"github.com/victor-novakoski/garimpo/internal/ai"
	"github.com/victor-novakoski/garimpo/internal/collector"
	"github.com/victor-novakoski/garimpo/internal/job"
	"github.com/victor-novakoski/garimpo/internal/profile"
	"github.com/victor-novakoski/garimpo/internal/score"
	"github.com/victor-novakoski/garimpo/internal/tracker"
)

// The demo shows the panel with made-up postings and the example profile:
// no database, no network, nothing of the owner. The postings pass through
// the same steps of a real collection (classify, dedupe, hard filter,
// score), so the demo cannot show a score the code would not give. Only
// the AI part is canned: the file brings the answer a model would give.
const (
	demoJobs    = "demo/vagas.json"
	demoProfile = "profile.example.yaml"
	demoSource  = "exemplo"
)

const demoCollectOff = "modo demo: as vagas são de exemplo e a coleta fica desligada"

// demoPosting is one made-up posting of the demo file and what the
// made-up person did with it.
type demoPosting struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Company     string   `json:"company"`
	Location    string   `json:"location"`
	URL         string   `json:"url"`
	Tags        []string `json:"tags"`
	Description string   `json:"description"`
	// PostedDaysAgo and SeenDaysAgo are counted back from the moment the
	// demo starts, so its dates are always recent.
	PostedDaysAgo int `json:"posted_days_ago"`
	// SeenDaysAgo is the run that found the posting; 0 is "nova hoje".
	SeenDaysAgo int `json:"seen_days_ago"`
	// AI is the canned answer of the model, in the format of ai.Schema.
	AI    json.RawMessage `json:"ai"`
	Marks []demoMark      `json:"marks"`
	Note  string          `json:"note"`
}

type demoMark struct {
	HoursAgo int    `json:"hours_ago"`
	To       string `json:"to"`
}

// demoFile is a collector that reads the demo file instead of the network.
type demoFile struct {
	postings []demoPosting
	at       time.Time
}

func (demoFile) Name() string { return demoSource }

func (d demoFile) Collect(context.Context) ([]job.Job, error) {
	jobs := make([]job.Job, len(d.postings))
	for i, p := range d.postings {
		jobs[i] = job.Job{
			Source: demoSource, SourceID: p.ID, URL: p.URL, Title: p.Title, Company: p.Company,
			Location: p.Location, Description: p.Description, Tags: p.Tags,
			PostedAt: d.at.AddDate(0, 0, -p.PostedDaysAgo),
		}
	}
	return jobs, nil
}

func loadDemoPostings(path string) ([]demoPosting, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("o modo demo lê %s e %s: rode na pasta do projeto (%w)", demoJobs, demoProfile, err)
	}
	var file struct {
		Postings []demoPosting `json:"postings"`
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&file); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	seen := map[string]bool{}
	for _, p := range file.Postings {
		if p.ID == "" || seen[p.ID] {
			return nil, fmt.Errorf("%s: id vazio ou repetido: %q", path, p.ID)
		}
		seen[p.ID] = true
	}
	return file.Postings, nil
}

// demoRanked gives each result that passed the filter its canned AI answer
// and sorts by total, as ai.Rank does with a real model.
func demoRanked(results []score.Result, byID map[string]demoPosting) ([]ai.Ranked, error) {
	ranked := withoutAI(results)
	for i := range ranked {
		raw := byID[ranked[i].Job.SourceID].AI
		if len(raw) == 0 {
			continue
		}
		a, err := ai.Parse(raw)
		if err != nil {
			return nil, fmt.Errorf("leitura da IA da vaga %q: %w", ranked[i].Job.SourceID, err)
		}
		ranked[i].AI = &a
		ranked[i].Total += a.Points()
	}
	slices.SortStableFunc(ranked, func(a, b ai.Ranked) int { return b.Total - a.Total })
	return ranked, nil
}

// openDemo builds the store of the demo: an in-memory record filled by
// running the postings of the file through a collection, one run per day
// the file names, and then marking what the file says was marked.
func openDemo(ctx context.Context, jobsPath, profilePath string, at time.Time, stderr io.Writer) (tracker.Store, error) {
	postings, err := loadDemoPostings(jobsPath)
	if err != nil {
		return nil, err
	}
	p, err := profile.Load(profilePath)
	if err != nil {
		return nil, err
	}
	byID := make(map[string]demoPosting, len(postings))
	var days []int
	for _, d := range postings {
		byID[d.ID] = d
		if !slices.Contains(days, d.SeenDaysAgo) {
			days = append(days, d.SeenDaysAgo)
		}
	}
	slices.Sort(days)
	slices.Reverse(days) // the oldest run first

	g := gather(ctx, []collector.Collector{demoFile{postings, at}}, false, io.Discard)
	results, dropped := score.Rank(p, g.jobs)
	ranked, err := demoRanked(results, byID)
	if err != nil {
		return nil, err
	}

	store := tracker.OpenMemory()
	numbers := map[string]int{} // id in the file -> number in the list
	for _, day := range days {
		var listed []ai.Ranked
		for _, r := range ranked {
			if byID[r.Job.SourceID].SeenDaysAgo >= day {
				listed = append(listed, r)
			}
		}
		rows, err := track(store, g, at.AddDate(0, 0, -day), listed, nil, true, io.Discard)
		if err != nil {
			return nil, err
		}
		for _, r := range rows {
			numbers[r.Job.SourceID] = r.Number
		}
	}
	for _, d := range postings {
		n, listed := numbers[d.ID]
		if !listed {
			if len(d.Marks) > 0 || d.Note != "" {
				return nil, fmt.Errorf("%s: a vaga %q foi cortada ou é repetida e não pode ter marca nem anotação", jobsPath, d.ID)
			}
			continue
		}
		for _, m := range d.Marks {
			to, err := tracker.ParseStatus(m.To)
			if err != nil {
				return nil, fmt.Errorf("%s: vaga %q: %w", jobsPath, d.ID, err)
			}
			if _, err := store.Mark(n, to, at.Add(-time.Duration(m.HoursAgo)*time.Hour)); err != nil {
				return nil, err
			}
		}
		if d.Note != "" {
			if _, err := store.SetNote(n, d.Note); err != nil {
				return nil, err
			}
		}
	}
	fmt.Fprintf(stderr, "modo demo: %d vagas de exemplo na lista, %d cortadas pelo filtro e %d repetidas. Nada é gravado: o que você marcar some ao fechar.\n",
		len(ranked), dropped, len(postings)-len(g.jobs))
	return store, nil
}
