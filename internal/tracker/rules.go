package tracker

import (
	"slices"
	"strings"
	"time"
)

// book is every entry in memory. The rules of docs/FLOW.md live here, so
// the file and the database stores cannot drift apart: each one loads a
// book, applies the same change and writes back what changed.
type book struct {
	Next    int     `json:"next"` // the next number to hand out
	Entries []Entry `json:"entries"`
}

// sync applies a run. It returns the entry of each listed posting, in the
// same order, and the indexes of the entries the run changed.
func (b *book) sync(run Run) (out []Synced, changed []int) {
	byKey := make(map[string]int) // key -> index in b.Entries
	for i, e := range b.Entries {
		for _, k := range e.Keys {
			byKey[k] = i
		}
	}
	out = make([]Synced, len(run.Listed))
	listed := make(map[int]bool)
	for n, s := range run.Listed {
		keys := append([]string{s.Job.Key()}, s.Copies...)
		i, known := -1, false
		for _, k := range keys {
			if at, ok := byKey[k]; ok {
				i, known = at, true
				break
			}
		}
		if !known {
			b.Entries = append(b.Entries, Entry{Number: b.Next, Status: StatusNew, FirstSeen: run.At})
			b.Next++
			i = len(b.Entries) - 1
		}
		e := &b.Entries[i]
		for _, k := range keys {
			if !slices.Contains(e.Keys, k) {
				e.Keys = append(e.Keys, k)
			}
			byKey[k] = i
		}
		e.Title, e.Company, e.URL = s.Job.Title, s.Job.Company, s.Job.URL
		if s.Score >= 0 {
			e.Score, e.Reasons = s.Score, s.Reasons
		}
		e.LastSeen, e.Gone = run.At, false
		if !listed[i] {
			listed[i] = true
			changed = append(changed, i)
		}
		out[n] = Synced{Entry: *e, NewToday: !known}
	}
	for i := range b.Entries {
		e := &b.Entries[i]
		if listed[i] || !sourceAnswered(e.Keys, run.Sources) {
			continue
		}
		present := false
		for _, k := range e.Keys {
			present = present || run.Present[k]
		}
		if e.Gone == present {
			e.Gone = !present
			changed = append(changed, i)
		}
	}
	return out, changed
}

// sourceAnswered reports whether every source of the keys answered in the
// run. Keys are "source:id" (job.Key), and the source may hold a colon.
func sourceAnswered(keys []string, sources map[string]bool) bool {
	for _, k := range keys {
		i := strings.LastIndex(k, ":")
		if i < 0 || !sources[k[:i]] {
			return false
		}
	}
	return len(keys) > 0
}

// find returns the entry with the number, or nil.
func (b *book) find(number int) *Entry {
	for i := range b.Entries {
		if b.Entries[i].Number == number {
			return &b.Entries[i]
		}
	}
	return nil
}

// list returns the entries in any of the statuses (all when none is given),
// ordered by number.
func (b *book) list(statuses []Status) []Entry {
	var out []Entry
	for _, e := range b.Entries {
		if len(statuses) == 0 || slices.Contains(statuses, e.Status) {
			out = append(out, e)
		}
	}
	slices.SortFunc(out, func(a, c Entry) int { return a.Number - c.Number })
	return out
}

// mark changes the status and records the change. Marking the status the
// entry already has records nothing. It reports whether anything changed.
func (e *Entry) mark(to Status, at time.Time) bool {
	if e.Status == to {
		return false
	}
	e.History = append(e.History, Event{At: at, From: e.Status, To: to})
	e.Status = to
	return true
}

// cleanNote is the note as it is stored.
func cleanNote(note string) string { return strings.TrimSpace(note) }

// NewToday reports which entries were first seen in the latest run, from
// the entries alone: the latest run is the newest LastSeen of all of them.
// It answers "nova hoje" for readers that did not make the run themselves.
func NewToday(all []Entry) map[int]bool {
	var latest time.Time
	for _, e := range all {
		if e.LastSeen.After(latest) {
			latest = e.LastSeen
		}
	}
	fresh := make(map[int]bool)
	for _, e := range all {
		if !latest.IsZero() && e.FirstSeen.Equal(latest) {
			fresh[e.Number] = true
		}
	}
	return fresh
}
