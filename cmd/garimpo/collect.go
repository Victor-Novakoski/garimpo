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

	"github.com/victor-novakoski/garimpo/internal/ai"
	"github.com/victor-novakoski/garimpo/internal/collector"
	"github.com/victor-novakoski/garimpo/internal/collector/githubissues"
	"github.com/victor-novakoski/garimpo/internal/collector/greenhouse"
	"github.com/victor-novakoski/garimpo/internal/collector/gupy"
	"github.com/victor-novakoski/garimpo/internal/job"
	"github.com/victor-novakoski/garimpo/internal/normalize"
	"github.com/victor-novakoski/garimpo/internal/profile"
	"github.com/victor-novakoski/garimpo/internal/score"
)

// defaultProfile is read when -profile is not given and the file exists.
const defaultProfile = "profile.yaml"

type collectOptions struct {
	days      int
	gupy      bool
	gupyTerms string
	asJSON    bool
	all       bool
	profile   string
	minPoints int
	why       bool
	ai        string
	aiModel   string
	aiTop     int
	aiCache   string
}

func parseCollect(args []string, stderr io.Writer) (collectOptions, error) {
	var o collectOptions
	fs := flag.NewFlagSet("collect", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.IntVar(&o.days, "days", 30, "só vagas do GitHub atualizadas nos últimos N dias (o Greenhouse traz o board inteiro)")
	fs.BoolVar(&o.gupy, "gupy", false, "inclui a Gupy (só uso pessoal, veja docs/SOURCES.md)")
	fs.StringVar(&o.gupyTerms, "gupy-terms", "golang,react", "termos de busca na Gupy, separados por vírgula")
	fs.BoolVar(&o.asJSON, "json", false, "saída em JSON, uma vaga por linha")
	fs.BoolVar(&o.all, "all", false, "mantém vagas fora de tecnologia (comercial, financeiro...)")
	fs.StringVar(&o.profile, "profile", "", "perfil para filtrar e dar nota (padrão: profile.yaml, se existir)")
	fs.IntVar(&o.minPoints, "min", 0, "só mostra vagas com pelo menos esta nota")
	fs.StringVar(&o.ai, "ai", "", "soma a nota da IA (até 30): \"local\" usa o claude -p da máquina, \"api\" usa a API do Claude")
	fs.StringVar(&o.aiModel, "ai-model", "", "modelo da IA (padrão: haiku no local, claude-haiku-5-5 na API)")
	fs.IntVar(&o.aiTop, "ai-top", 20, "quantas vagas, das melhores pela nota das regras, a IA avalia")
	fs.StringVar(&o.aiCache, "ai-cache", "", "arquivo de cache das notas da IA (\"off\" desliga; padrão: pasta de cache do usuário)")
	fs.BoolVar(&o.why, "why", false, "mostra o motivo de cada ponto da nota")
	if err := fs.Parse(args); err != nil {
		return o, err
	}
	if fs.NArg() > 0 {
		return o, fmt.Errorf("argumento inesperado: %q", fs.Arg(0))
	}
	switch o.ai {
	case "", "local", "api":
	default:
		return o, fmt.Errorf("-ai: valor desconhecido %q (use local ou api)", o.ai)
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
// the classified, deduplicated jobs, only tech ones unless all is set. A
// failing source never stops the rest.
func gather(ctx context.Context, cs []collector.Collector, all bool, stderr io.Writer) []job.Job {
	results, _ := collector.RunAll(ctx, cs)
	var collected []job.Job
	for _, r := range results {
		if r.Err != nil {
			fmt.Fprintf(stderr, "%-28s erro: %v\n", r.Source, r.Err)
		} else {
			fmt.Fprintf(stderr, "%-28s %d vagas\n", r.Source, len(r.Jobs))
		}
		for _, j := range r.Jobs {
			collected = append(collected, normalize.Classify(j))
		}
	}
	kept := collected
	if !all {
		kept = kept[:0:0]
		for _, j := range collected {
			if normalize.IsTech(j) {
				kept = append(kept, j)
			}
		}
	}
	deduped := normalize.Dedupe(kept)
	fmt.Fprintf(stderr, "total: %d vagas (%d fora de tecnologia, %d repetidas removidas)\n",
		len(deduped), len(collected)-len(kept), len(kept)-len(deduped))
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
	p, hasProfile, err := loadProfile(o.profile)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	if o.ai != "" && !hasProfile {
		fmt.Fprintln(stderr, "-ai precisa de um perfil: a IA compara a vaga com ele")
		return 2
	}
	jobs := gather(ctx, collectors(o, time.Now()), o.all, stderr)
	if !hasProfile {
		fmt.Fprintf(stderr, "sem perfil: listando sem nota (copie profile.example.yaml para %s)\n", defaultProfile)
		err = printJobs(stdout, jobs, o.asJSON)
	} else {
		results, dropped := score.Rank(p, jobs)
		fmt.Fprintf(stderr, "perfil: %d vagas passaram, %d cortadas pelo filtro\n", len(results), dropped)
		ranked, maxPoints := withoutAI(results), score.Max
		if o.ai != "" {
			ranked, maxPoints = assess(ctx, o, p, results, stderr), ai.MaxTotal
		}
		err = printResults(stdout, atLeast(ranked, o.minPoints), maxPoints, o.asJSON, o.why)
	}
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}

func withoutAI(results []score.Result) []ai.Ranked {
	ranked := make([]ai.Ranked, len(results))
	for i, r := range results {
		ranked[i] = ai.Ranked{Result: r, Total: r.Points}
	}
	return ranked
}

// newScorer picks the AI implementation; tests replace it.
var newScorer = func(o collectOptions, p profile.Profile) ai.Scorer {
	if o.ai == "api" {
		return ai.NewAPI(p, o.aiModel)
	}
	return ai.NewLocal(p, o.aiModel)
}

// assess adds the AI part to the best results and reports what happened on
// stderr. Whatever goes wrong, every job keeps its deterministic points.
func assess(ctx context.Context, o collectOptions, p profile.Profile, results []score.Result, stderr io.Writer) []ai.Ranked {
	scorer := newScorer(o, p)
	var cache *ai.Cache
	if o.aiCache != "off" {
		path := o.aiCache
		if path == "" {
			path, _ = ai.DefaultCachePath()
		}
		if path != "" {
			cache = ai.OpenCache(path)
			scorer = cache.Wrap(scorer, ai.System(p))
		}
	}
	n := min(max(o.aiTop, 0), len(results))
	fmt.Fprintf(stderr, "IA (%s): avaliando as %d melhores ", scorer.Name(), n)
	ranked, fatal := ai.Rank(ctx, scorer, results, n, func(int, int) { fmt.Fprint(stderr, ".") })
	fmt.Fprintln(stderr)
	assessed, failed := 0, 0
	for _, r := range ranked {
		switch {
		case r.AI != nil:
			assessed++
		case r.AIError != "":
			failed++
		}
	}
	fmt.Fprintf(stderr, "IA: %d avaliadas, %d com erro\n", assessed, failed)
	if fatal != nil {
		fmt.Fprintf(stderr, "IA parou: %v. As vagas seguem só com a nota das regras.\n", fatal)
	}
	if cache != nil {
		if err := cache.Save(); err != nil {
			fmt.Fprintf(stderr, "cache da IA não foi salvo: %v\n", err)
		}
	}
	return ranked
}

// loadProfile reads the profile from path, or from defaultProfile when
// path is empty and that file exists. No profile is not an error.
func loadProfile(path string) (profile.Profile, bool, error) {
	if path == "" {
		if _, err := os.Stat(defaultProfile); err != nil {
			return profile.Profile{}, false, nil
		}
		path = defaultProfile
	}
	p, err := profile.Load(path)
	if err != nil {
		return p, false, err
	}
	return p, true, nil
}

func atLeast(results []ai.Ranked, minPoints int) []ai.Ranked {
	out := results[:0:0]
	for _, r := range results {
		if r.Total >= minPoints {
			out = append(out, r)
		}
	}
	return out
}

func printResults(w io.Writer, results []ai.Ranked, maxPoints int, asJSON, why bool) error {
	if asJSON {
		enc := json.NewEncoder(w)
		for _, r := range results {
			if err := enc.Encode(r); err != nil {
				return err
			}
		}
		return nil
	}
	if why {
		printWhy(w, results, maxPoints)
		return nil
	}
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintf(tw, "NOTA\tVAGA\tEMPRESA\tMODELO\tCONTRATO\tNÍVEL\tLINK\n")
	for _, r := range results {
		j := r.Job
		fmt.Fprintf(tw, "%d/%d\t%s\t%s\t%s\t%s\t%s\t%s\n",
			r.Total, maxPoints, cut(j.Title, 60), dash(j.Company), dash(string(j.WorkMode)),
			dash(string(j.Contract)), dash(string(j.Seniority)), j.URL)
	}
	return tw.Flush()
}

// printWhy prints one block per job with every reason. It is not a table:
// the AI lines are sentences and would stretch the columns.
func printWhy(w io.Writer, results []ai.Ranked, maxPoints int) {
	withAI := maxPoints == ai.MaxTotal
	for i, r := range results {
		if i > 0 {
			fmt.Fprintln(w)
		}
		j := r.Job
		fmt.Fprintf(w, "%d/%d  %s\n", r.Total, maxPoints, j.Title)
		fmt.Fprintf(w, "  %s | %s | %s | %s\n", dash(j.Company), dash(string(j.WorkMode)),
			dash(string(j.Contract)), dash(string(j.Seniority)))
		fmt.Fprintf(w, "  %s\n", j.URL)
		for _, reason := range r.Reasons {
			fmt.Fprintf(w, "  %s\n", reason)
		}
		switch {
		case r.AI != nil:
			fmt.Fprintf(w, "  %+d IA, encaixe %s: %s\n", r.AI.Points(), r.AI.Fit.Label(), r.AI.Summary)
			for _, s := range r.AI.Strengths {
				fmt.Fprintf(w, "      a favor: %s\n", s)
			}
			for _, s := range r.AI.Gaps {
				fmt.Fprintf(w, "      falta: %s\n", s)
			}
			for _, s := range r.AI.Highlights {
				fmt.Fprintf(w, "      destaque: %s\n", s)
			}
		case r.AIError != "":
			fmt.Fprintf(w, "  +0 IA não avaliou: %s\n", r.AIError)
		case withAI:
			fmt.Fprintln(w, "  +0 IA não avaliou (fora das melhores, veja -ai-top)")
		}
	}
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
