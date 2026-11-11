package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/victor-novakoski/garimpo/internal/ai"
	"github.com/victor-novakoski/garimpo/internal/score"
	"github.com/victor-novakoski/garimpo/internal/tracker"
)

// The commands here follow docs/FLOW.md: a posting only leaves the daily
// list when the person marks it.

func stateFlag(fs *flag.FlagSet, target *string) {
	fs.StringVar(target, "state", "", "arquivo com o que você marcou em cada vaga (\"off\" desliga; padrão: pasta de configuração do usuário)")
}

// openStore returns the tracker store, or nil when tracking is off.
func openStore(path string) (tracker.Store, error) {
	if path == "off" {
		return nil, nil
	}
	if path == "" {
		var err error
		if path, err = tracker.DefaultPath(); err != nil {
			return nil, fmt.Errorf("não achei a pasta de configuração (use -state): %w", err)
		}
	}
	return tracker.OpenFile(path), nil
}

// row is one line of the daily list: the scored job and what the tracker
// knows about it.
type row struct {
	ai.Ranked
	Number   int            `json:"number,omitempty"`
	Status   tracker.Status `json:"status,omitempty"`
	NewToday bool           `json:"new_today,omitempty"`
	Note     string         `json:"note,omitempty"`
}

func (r row) number() string {
	if r.Number == 0 {
		return "-"
	}
	return "#" + strconv.Itoa(r.Number)
}

func (r row) state() string {
	switch {
	case r.Number == 0:
		return ""
	case r.NewToday:
		return "nova hoje"
	}
	return r.Status.Label()
}

// splitByStatus separates the results still in the daily list from the
// ones the person already applied to or discarded.
func splitByStatus(store tracker.Store, results []score.Result) (visible, hidden []score.Result, err error) {
	if store == nil {
		return results, nil, nil
	}
	entries, err := store.List()
	if err != nil {
		return nil, nil, err
	}
	status := make(map[string]tracker.Status)
	for _, e := range entries {
		for _, k := range e.Keys {
			status[k] = e.Status
		}
	}
	for _, r := range results {
		if s, known := status[r.Job.Key()]; known && !s.InDailyList() {
			hidden = append(hidden, r)
		} else {
			visible = append(visible, r)
		}
	}
	return visible, hidden, nil
}

// track records the run and builds the daily list: postings marked "vou
// aplicar" first, then the rest by score.
func track(store tracker.Store, g gathered, now time.Time, ranked []ai.Ranked, hidden []score.Result, stderr io.Writer) ([]row, error) {
	rows := make([]row, len(ranked))
	for i, r := range ranked {
		rows[i] = row{Ranked: r}
	}
	if store == nil {
		return rows, nil
	}
	run := tracker.Run{At: now, Present: g.present, Sources: g.sources}
	for _, r := range ranked {
		run.Listed = append(run.Listed, tracker.Seen{Job: r.Job, Copies: g.copies[r.Job.Key()], Score: r.Total})
	}
	for _, r := range hidden {
		run.Listed = append(run.Listed, tracker.Seen{Job: r.Job, Copies: g.copies[r.Job.Key()], Score: -1})
	}
	synced, err := store.Sync(run)
	if err != nil {
		return nil, err
	}
	fresh, saved := 0, 0
	for i := range rows {
		e := synced[i].Entry
		rows[i].Number, rows[i].Status, rows[i].NewToday, rows[i].Note = e.Number, e.Status, synced[i].NewToday, e.Note
		if synced[i].NewToday {
			fresh++
		}
		if e.Status == tracker.StatusSaved {
			saved++
		}
	}
	slices.SortStableFunc(rows, func(a, b row) int {
		return boolInt(b.Status == tracker.StatusSaved) - boolInt(a.Status == tracker.StatusSaved)
	})
	fmt.Fprintf(stderr, "lista: %d vagas (%d novas hoje, %d marcadas como vou aplicar); %d já aplicadas ou descartadas ficaram de fora\n",
		len(rows), fresh, saved, len(hidden))
	return rows, nil
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// now is replaced in tests.
var now = time.Now

// trackArgs parses the flags shared by mark, note, list and show and
// returns the store and the remaining arguments.
func trackArgs(name string, args []string, stderr io.Writer) (tracker.Store, []string, int) {
	var state string
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(stderr)
	stateFlag(fs, &state)
	if err := fs.Parse(args); err != nil {
		return nil, nil, 2
	}
	store, err := openStore(state)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return nil, nil, 1
	}
	if store == nil {
		fmt.Fprintln(stderr, "-state off não faz sentido aqui: este comando lê o que você marcou")
		return nil, nil, 2
	}
	return store, fs.Args(), 0
}

func parseNumber(s string) (int, error) {
	n, err := strconv.Atoi(strings.TrimPrefix(s, "#"))
	if err != nil || n <= 0 {
		return 0, fmt.Errorf("número de vaga inválido: %q (use o número da lista, como 12)", s)
	}
	return n, nil
}

func runMark(_ context.Context, args []string, stdout, stderr io.Writer) int {
	store, rest, code := trackArgs("mark", args, stderr)
	if store == nil {
		return code
	}
	if len(rest) < 2 {
		fmt.Fprintln(stderr, "uso: garimpo mark <número> <estado>   (nova, vou aplicar, apliquei, entrevista, descartada)")
		return 2
	}
	n, err := parseNumber(rest[0])
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	status, err := tracker.ParseStatus(strings.Join(rest[1:], " "))
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	e, err := store.Mark(n, status, now())
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintf(stdout, "#%d %s: %s\n", e.Number, e.Status.Label(), e.Title)
	return 0
}

func runNote(_ context.Context, args []string, stdout, stderr io.Writer) int {
	store, rest, code := trackArgs("note", args, stderr)
	if store == nil {
		return code
	}
	if len(rest) < 1 {
		fmt.Fprintln(stderr, `uso: garimpo note <número> <texto>   (sem texto, apaga a anotação)`)
		return 2
	}
	n, err := parseNumber(rest[0])
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	e, err := store.SetNote(n, strings.Join(rest[1:], " "))
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if e.Note == "" {
		fmt.Fprintf(stdout, "#%d sem anotação: %s\n", e.Number, e.Title)
	} else {
		fmt.Fprintf(stdout, "#%d anotado: %s\n", e.Number, e.Note)
	}
	return 0
}

func entryState(e tracker.Entry) string {
	if e.Gone && e.Status.InDailyList() {
		return e.Status.Label() + " (saiu da fonte)"
	}
	return e.Status.Label()
}

// runList prints what was marked, without collecting. With no status it
// shows the follow-up: vou aplicar, apliquei and entrevista.
func runList(_ context.Context, args []string, stdout, stderr io.Writer) int {
	store, rest, code := trackArgs("list", args, stderr)
	if store == nil {
		return code
	}
	want := []tracker.Status{tracker.StatusInterview, tracker.StatusApplied, tracker.StatusSaved}
	if len(rest) > 0 {
		if word := strings.Join(rest, " "); word == "tudo" {
			want = tracker.Statuses()
		} else {
			status, err := tracker.ParseStatus(word)
			if err != nil {
				fmt.Fprintf(stderr, "%v, ou \"tudo\"\n", err)
				return 2
			}
			want = []tracker.Status{status}
		}
	}
	entries, err := store.List(want...)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if len(entries) == 0 {
		fmt.Fprintln(stderr, "nenhuma vaga nesse estado. Marque com: garimpo mark <número> <estado>")
		return 0
	}
	// Funnel order, furthest first, then by number.
	slices.SortStableFunc(entries, func(a, b tracker.Entry) int {
		return slices.Index(want, a.Status) - slices.Index(want, b.Status)
	})
	tw := tabwriter.NewWriter(stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "#\tESTADO\tDESDE\tNOTA\tVAGA\tEMPRESA\tLINK")
	for _, e := range entries {
		fmt.Fprintf(tw, "#%d\t%s\t%s\t%d\t%s\t%s\t%s\n", e.Number, entryState(e),
			e.Since().Local().Format("02/01"), e.Score, cut(e.Title, 60), dash(e.Company), e.URL)
	}
	return boolInt(tw.Flush() != nil)
}

func runShow(_ context.Context, args []string, stdout, stderr io.Writer) int {
	store, rest, code := trackArgs("show", args, stderr)
	if store == nil {
		return code
	}
	if len(rest) != 1 {
		fmt.Fprintln(stderr, "uso: garimpo show <número>")
		return 2
	}
	n, err := parseNumber(rest[0])
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	e, err := store.Get(n)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	const day = "02/01/2006"
	fmt.Fprintf(stdout, "#%d  %s\n", e.Number, e.Title)
	fmt.Fprintf(stdout, "  empresa: %s\n", dash(e.Company))
	fmt.Fprintf(stdout, "  link: %s\n", e.URL)
	fmt.Fprintf(stdout, "  estado: %s, desde %s\n", entryState(e), e.Since().Local().Format(day))
	fmt.Fprintf(stdout, "  nota na última coleta: %d\n", e.Score)
	fmt.Fprintf(stdout, "  vista pela primeira vez em %s, pela última em %s\n",
		e.FirstSeen.Local().Format(day), e.LastSeen.Local().Format(day))
	if e.Note != "" {
		fmt.Fprintf(stdout, "  anotação: %s\n", e.Note)
	}
	for _, ev := range e.History {
		fmt.Fprintf(stdout, "  %s: %s -> %s\n", ev.At.Local().Format(day), ev.From.Label(), ev.To.Label())
	}
	return 0
}
