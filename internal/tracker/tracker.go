// Package tracker remembers what the person did with each posting. The
// flow is in docs/FLOW.md: a posting only leaves the daily list when the
// person marks it, and the same states serve the CLI, the API and the
// panel. Store is the interface; File keeps it in a local JSON file until
// the database arrives.
package tracker

import (
	"fmt"
	"strings"
	"time"

	"github.com/victor-novakoski/garimpo/internal/job"
	"github.com/victor-novakoski/garimpo/internal/normalize"
)

// Status is where a posting is in the funnel.
type Status string

const (
	StatusNew       Status = "new"       // found, not decided yet
	StatusSaved     Status = "saved"     // wants to apply, later
	StatusApplied   Status = "applied"   // application sent
	StatusInterview Status = "interview" // the company answered
	StatusDiscarded Status = "discarded" // not interested, or it did not work out
)

// statuses lists every status in funnel order, with its Portuguese label
// and the words accepted on the command line.
var statuses = []struct {
	status Status
	label  string
	words  []string
}{
	{StatusNew, "nova", []string{"nova", "novo", "new"}},
	{StatusSaved, "vou aplicar", []string{"vou aplicar", "depois", "salvar", "salva", "saved"}},
	{StatusApplied, "apliquei", []string{"apliquei", "aplicada", "aplicado", "applied"}},
	{StatusInterview, "entrevista", []string{"entrevista", "interview"}},
	{StatusDiscarded, "descartada", []string{"descartada", "descartei", "descartar", "discarded"}},
}

// Statuses returns every status in funnel order.
func Statuses() []Status {
	out := make([]Status, len(statuses))
	for i, s := range statuses {
		out[i] = s.status
	}
	return out
}

// Label is the status in Portuguese, for people.
func (s Status) Label() string {
	for _, c := range statuses {
		if c.status == s {
			return c.label
		}
	}
	return string(s)
}

// InDailyList reports whether postings in this status still show up in the
// daily list. Applied and interview move to the follow-up list; discarded
// ones are gone.
func (s Status) InDailyList() bool { return s == StatusNew || s == StatusSaved }

// ParseStatus reads a status typed by a person: the Portuguese label, a
// close variant ("descartei") or the code name, in any case, with or
// without accents.
func ParseStatus(text string) (Status, error) {
	word := normalize.Fold(text)
	for _, c := range statuses {
		for _, w := range c.words {
			if word == w {
				return c.status, nil
			}
		}
	}
	labels := make([]string, len(statuses))
	for i, c := range statuses {
		labels[i] = c.label
	}
	return "", fmt.Errorf("estado desconhecido: %q (use %s)", text, strings.Join(labels, ", "))
}

// Event is one status change.
type Event struct {
	At   time.Time `json:"at"`
	From Status    `json:"from"`
	To   Status    `json:"to"`
}

// Entry is what is remembered about one posting.
type Entry struct {
	// Number is the short id the person types (#12). It is given once and
	// never changes.
	Number int `json:"number"`
	// Keys are the job.Key of the posting and of its copies in other
	// sources: the same opening posted twice is one entry.
	Keys    []string `json:"keys"`
	Title   string   `json:"title"`
	Company string   `json:"company,omitempty"`
	URL     string   `json:"url"`
	Status  Status   `json:"status"`
	Note    string   `json:"note,omitempty"`
	// Score is the total from the last run that listed it. It is only
	// informative: the score is recomputed on every run.
	Score     int       `json:"score"`
	FirstSeen time.Time `json:"first_seen"`
	LastSeen  time.Time `json:"last_seen"`
	// Gone is set when the posting no longer shows up in its source.
	Gone    bool    `json:"gone,omitempty"`
	History []Event `json:"history,omitempty"`
}

// Since is when the entry got its current status.
func (e Entry) Since() time.Time {
	if n := len(e.History); n > 0 {
		return e.History[n-1].At
	}
	return e.FirstSeen
}

// Seen is one posting of a run, with the copies dedupe folded into it.
type Seen struct {
	Job    job.Job
	Copies []string // keys of the duplicates
	// Score is the total of this run. A negative score keeps the one on
	// record (for postings the run did not score again).
	Score int
}

// Run describes one collection to Sync.
type Run struct {
	At time.Time
	// Listed are the postings that passed the filter in this run.
	Listed []Seen
	// Present has the key of every posting the sources returned, listed
	// or not. A known posting missing from it is gone.
	Present map[string]bool
	// Sources has the collectors that answered. A posting is only
	// declared gone when its own source answered without it.
	Sources map[string]bool
}

// Synced is the result of Sync for one listed posting.
type Synced struct {
	Entry Entry
	// NewToday is true when this run saw the posting for the first time.
	NewToday bool
}

// Store keeps the entries. The file implementation is in file.go; the
// database one comes with the API and must behave the same.
type Store interface {
	// Sync records a run and returns the entry of each listed posting, in
	// the same order.
	Sync(run Run) ([]Synced, error)
	// Get returns one entry by number.
	Get(number int) (Entry, error)
	// Mark changes the status of an entry and records the change.
	Mark(number int, to Status, at time.Time) (Entry, error)
	// SetNote replaces the note of an entry. An empty note clears it.
	SetNote(number int, note string) (Entry, error)
	// List returns the entries in any of the statuses (all when none is
	// given), ordered by number.
	List(statuses ...Status) ([]Entry, error)
}

// ErrNotFound is returned when no entry has the number.
type ErrNotFound int

func (e ErrNotFound) Error() string {
	return fmt.Sprintf("não existe vaga #%d (rode o collect para ver os números)", int(e))
}
