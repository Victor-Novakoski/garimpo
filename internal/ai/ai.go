// Package ai adds the part of the score a rule cannot compute: it reads the
// whole posting and says how well what the job really asks fits the
// person. The model only picks one of five fixed classes; the points for
// each class are set here, capped at 30, and every answer is validated. It
// runs after the hard filter and cannot drop a job or bring one back.
package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/victor-novakoski/garimpo/internal/job"
)

// Max is the most points the AI step can add.
const Max = 30

// Fit is the class the model picks.
type Fit string

const (
	FitStrong  Fit = "strong"
	FitGood    Fit = "good"
	FitPartial Fit = "partial"
	FitWeak    Fit = "weak"
	FitNone    Fit = "none"
)

// fits lists the classes from best to worst, with the points and the
// Portuguese label of each.
var fits = []struct {
	fit    Fit
	points int
	label  string
}{
	{FitStrong, Max, "forte"},
	{FitGood, 22, "bom"},
	{FitPartial, 14, "parcial"},
	{FitWeak, 6, "fraco"},
	{FitNone, 0, "nenhum"},
}

// Points is what the class is worth. An unknown class is worth nothing.
func (f Fit) Points() int {
	for _, c := range fits {
		if c.fit == f {
			return c.points
		}
	}
	return 0
}

// Label is the class in Portuguese, for people.
func (f Fit) Label() string {
	for _, c := range fits {
		if c.fit == f {
			return c.label
		}
	}
	return string(f)
}

func (f Fit) valid() bool {
	for _, c := range fits {
		if c.fit == f {
			return true
		}
	}
	return false
}

// Assessment is the validated answer for one job.
type Assessment struct {
	Fit        Fit      `json:"fit"`
	Summary    string   `json:"summary"`
	Strengths  []string `json:"strengths"`
	Gaps       []string `json:"gaps"`
	Highlights []string `json:"highlights"`
}

// Points is what the assessment adds to the score.
func (a Assessment) Points() int { return a.Fit.Points() }

// Scorer assesses one job against the profile it was built with.
type Scorer interface {
	// Name identifies the implementation and model ("api:claude-haiku-5-5").
	// It is part of the cache key: answers from different models do not mix.
	Name() string
	Assess(ctx context.Context, j job.Job) (Assessment, error)
}

// ErrUnavailable marks an error that will repeat for every job (no
// credentials, CLI not installed). The caller stops instead of failing
// once per job.
var ErrUnavailable = errors.New("IA indisponível")

// Limits for the lists and texts of an answer. Longer answers are cut, not
// rejected: the class is what scores.
const (
	maxItems     = 3
	maxItemRunes = 200
	maxSummary   = 300
)

// Schema is the JSON Schema both implementations ask the model to follow.
func Schema() map[string]any {
	classes := make([]string, len(fits))
	for i, c := range fits {
		classes[i] = string(c.fit)
	}
	list := map[string]any{"type": "array", "items": map[string]any{"type": "string"}}
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"fit":        map[string]any{"type": "string", "enum": classes},
			"summary":    map[string]any{"type": "string"},
			"strengths":  list,
			"gaps":       list,
			"highlights": list,
		},
		"required":             []string{"fit", "summary", "strengths", "gaps", "highlights"},
		"additionalProperties": false,
	}
}

// Parse validates a model answer. The class must be one of the five and the
// summary must exist; anything else is an error, so a malformed or
// manipulated answer never becomes points.
func Parse(data []byte) (Assessment, error) {
	var a Assessment
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&a); err != nil {
		return Assessment{}, fmt.Errorf("resposta da IA fora do formato: %w", err)
	}
	if dec.More() {
		return Assessment{}, errors.New("resposta da IA fora do formato: texto depois do JSON")
	}
	if !a.Fit.valid() {
		return Assessment{}, fmt.Errorf("resposta da IA com classe desconhecida: %q", a.Fit)
	}
	a.Summary = cut(clean(a.Summary), maxSummary)
	if a.Summary == "" {
		return Assessment{}, errors.New("resposta da IA sem resumo")
	}
	a.Strengths = tidy(a.Strengths)
	a.Gaps = tidy(a.Gaps)
	a.Highlights = tidy(a.Highlights)
	return a, nil
}

func tidy(items []string) []string {
	out := make([]string, 0, min(len(items), maxItems))
	for _, s := range items {
		if s = cut(clean(s), maxItemRunes); s != "" && len(out) < maxItems {
			out = append(out, s)
		}
	}
	return out
}

// clean collapses whitespace, so an answer never breaks the terminal table.
func clean(s string) string { return strings.Join(strings.Fields(s), " ") }

func cut(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}
