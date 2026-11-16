package tracker

import (
	"sync"
	"time"

	"github.com/victor-novakoski/garimpo/internal/job"
)

// Memory is a Store that lives only while the program runs. The demo uses
// it: what is marked there is never written anywhere. It follows the same
// rules (rules.go) and passes the same tests as the other stores.
type Memory struct {
	mu       sync.Mutex
	book     book
	postings map[int]job.Job // number -> the posting as last listed
}

// OpenMemory returns an empty store.
func OpenMemory() *Memory {
	return &Memory{book: book{Next: 1}, postings: map[int]job.Job{}}
}

func (m *Memory) Sync(run Run) ([]Synced, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out, _ := m.book.sync(run)
	for i, s := range run.Listed {
		m.postings[out[i].Entry.Number] = s.Job
	}
	return clone(out, func(s Synced) Synced { s.Entry = s.Entry.clone(); return s }), nil
}

func (m *Memory) update(number int, change func(*Entry)) (Entry, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	e := m.book.find(number)
	if e == nil {
		return Entry{}, ErrNotFound(number)
	}
	change(e)
	return e.clone(), nil
}

func (m *Memory) Get(number int) (Entry, error) {
	return m.update(number, func(*Entry) {})
}

func (m *Memory) Mark(number int, to Status, at time.Time) (Entry, error) {
	return m.update(number, func(e *Entry) { e.mark(to, at) })
}

func (m *Memory) SetNote(number int, note string) (Entry, error) {
	return m.update(number, func(e *Entry) { e.Note = cleanNote(note) })
}

func (m *Memory) List(statuses ...Status) ([]Entry, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return clone(m.book.list(statuses), Entry.clone), nil
}

// Posting returns the text of a posting as of the last run that listed it.
func (m *Memory) Posting(number int) (job.Job, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	j, found := m.postings[number]
	return j, found, nil
}

// clone copies the slices of an entry, so what a caller holds does not
// change under it when the store is written again.
func (e Entry) clone() Entry {
	e.Keys = append([]string(nil), e.Keys...)
	e.Reasons = append([]string(nil), e.Reasons...)
	e.History = append([]Event(nil), e.History...)
	return e
}

func clone[T any](items []T, copyOf func(T) T) []T {
	out := make([]T, len(items))
	for i, item := range items {
		out[i] = copyOf(item)
	}
	return out
}
