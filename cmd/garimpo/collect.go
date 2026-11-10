package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/victor-novakoski/garimpo/internal/collector"
	"github.com/victor-novakoski/garimpo/internal/collector/githubissues"
	"github.com/victor-novakoski/garimpo/internal/collector/greenhouse"
	"github.com/victor-novakoski/garimpo/internal/collector/gupy"
	"github.com/victor-novakoski/garimpo/internal/job"
	"github.com/victor-novakoski/garimpo/internal/normalize"
)

type collectOptions struct {
	days      int
	gupy      bool
	gupyTerms string
	asJSON    bool
}

func parseCollect(args []string, stderr io.Writer) (collectOptions, error) {
	var o collectOptions
	fs := flag.NewFlagSet("collect", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.IntVar(&o.days, "days", 30, "só vagas do GitHub atualizadas nos últimos N dias")
	fs.BoolVar(&o.gupy, "gupy", false, "inclui a Gupy (só uso pessoal, veja docs/SOURCES.md)")
	fs.StringVar(&o.gupyTerms, "gupy-terms", "golang,react", "termos de busca na Gupy, separados por vírgula")
	fs.BoolVar(&o.asJSON, "json", false, "saída em JSON, uma vaga por linha")
	if err := fs.Parse(args); err != nil {
		return o, err
	}
	if fs.NArg() > 0 {
		return o, fmt.Errorf("argumento inesperado: %q", fs.Arg(0))
	}
	return o, nil
}

// collectors builds the source list from the options.
func collectors(o collectOptions, now time.Time) []collector.Collector {
	var cs []collector.Collector
	gh := githubissues.NewClient(os.Getenv("GITHUB_TOKEN"))
	for _, repo := range githubissues.Repos {
		cs = append(cs, githubissues.Collector{
			Repo:   repo,
			Since:  now.AddDate(0, 0, -o.days),
			Client: gh,
		})
	}
	green := greenhouse.NewClient()
	for _, b := range greenhouse.Boards {
		cs = append(cs, greenhouse.Collector{Board: b, Client: green})
	}
	if o.gupy {
		var terms []string
		for _, t := range strings.Split(o.gupyTerms, ",") {
			if t = strings.TrimSpace(t); t != "" {
				terms = append(terms, t)
			}
		}
		cs = append(cs, gupy.Collector{Terms: terms, Client: gupy.NewClient()})
	}
	return cs
}

// gather runs the collectors, reports each source on stderr and returns
// the classified, deduplicated jobs. A failing source never stops the rest.
func gather(ctx context.Context, cs []collector.Collector, stderr io.Writer) []job.Job {
	results, _ := collector.RunAll(ctx, cs)
	var all []job.Job
	for _, r := range results {
		if r.Err != nil {
			fmt.Fprintf(stderr, "%-28s erro: %v\n", r.Source, r.Err)
		} else {
			fmt.Fprintf(stderr, "%-28s %d vagas\n", r.Source, len(r.Jobs))
		}
		for _, j := range r.Jobs {
			all = append(all, normalize.Classify(j))
		}
	}
	deduped := normalize.Dedupe(all)
	fmt.Fprintf(stderr, "total: %d vagas (%d repetidas removidas)\n", len(deduped), len(all)-len(deduped))
	return deduped
}

func printJobs(w io.Writer, jobs []job.Job, asJSON bool) error {
	if asJSON {
		enc := json.NewEncoder(w)
		for _, j := range jobs {
			if err := enc.Encode(j); err != nil {
				return err
			}
		}
		return nil
	}
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "VAGA\tEMPRESA\tMODELO\tCONTRATO\tNÍVEL\tLINK")
	for _, j := range jobs {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\n",
			cut(j.Title, 60), dash(j.Company), dash(string(j.WorkMode)),
			dash(string(j.Contract)), dash(string(j.Seniority)), j.URL)
	}
	return tw.Flush()
}

func runCollect(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	o, err := parseCollect(args, stderr)
	if err != nil {
		if err != flag.ErrHelp {
			fmt.Fprintln(stderr, err)
		}
		return 2
	}
	jobs := gather(ctx, collectors(o, time.Now()), stderr)
	if err := printJobs(stdout, jobs, o.asJSON); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}

func cut(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}

func dash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}
