package tracker

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

// File is a Store kept in one JSON file on the person's machine. It is
// read and written whole: a job search has a few hundred entries.
type File struct {
	path string
}

type fileData struct {
	Next    int     `json:"next"` // the next number to hand out
	Entries []Entry `json:"entries"`
}

// DefaultPath is the tracker file in the user's config directory, outside
// any repository.
func DefaultPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "garimpo", "vagas.json"), nil
}

// OpenFile returns the store at path. The file is created on first write.
func OpenFile(path string) *File { return &File{path: path} }

func (f *File) load() (fileData, error) {
	d := fileData{Next: 1}
	data, err := os.ReadFile(f.path)
	if errors.Is(err, fs.ErrNotExist) {
		return d, nil
	}
	if err != nil {
		return d, err
	}
	// A broken file is an error, never an empty start: it holds what the
	// person marked and must not be overwritten.
	if err := json.Unmarshal(data, &d); err != nil {
		return d, fmt.Errorf("%s está corrompido: %w", f.path, err)
	}
	return d, nil
}

func (f *File) save(d fileData) error {
	data, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(f.path), 0o700); err != nil {
		return err
	}
	tmp := f.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, f.path)
}

func (f *File) Sync(run Run) ([]Synced, error) {
	d, err := f.load()
	if err != nil {
		return nil, err
	}
	byKey := make(map[string]int) // key -> index in d.Entries
	for i, e := range d.Entries {
		for _, k := range e.Keys {
			byKey[k] = i
		}
	}
	out := make([]Synced, len(run.Listed))
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
			d.Entries = append(d.Entries, Entry{Number: d.Next, Status: StatusNew, FirstSeen: run.At})
			d.Next++
			i = len(d.Entries) - 1
		}
		e := &d.Entries[i]
		for _, k := range keys {
			if !slices.Contains(e.Keys, k) {
				e.Keys = append(e.Keys, k)
			}
			byKey[k] = i
		}
		e.Title, e.Company, e.URL = s.Job.Title, s.Job.Company, s.Job.URL
		if s.Score >= 0 {
			e.Score = s.Score
		}
		e.LastSeen, e.Gone = run.At, false
		listed[i] = true
		out[n] = Synced{Entry: *e, NewToday: !known}
	}
	for i := range d.Entries {
		e := &d.Entries[i]
		if listed[i] || !sourceAnswered(e.Keys, run.Sources) {
			continue
		}
		present := false
		for _, k := range e.Keys {
			present = present || run.Present[k]
		}
		e.Gone = !present
	}
	return out, f.save(d)
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

func (f *File) update(number int, change func(*Entry)) (Entry, error) {
	d, err := f.load()
	if err != nil {
		return Entry{}, err
	}
	for i := range d.Entries {
		if d.Entries[i].Number == number {
			change(&d.Entries[i])
			return d.Entries[i], f.save(d)
		}
	}
	return Entry{}, ErrNotFound(number)
}

func (f *File) Get(number int) (Entry, error) {
	d, err := f.load()
	if err != nil {
		return Entry{}, err
	}
	for _, e := range d.Entries {
		if e.Number == number {
			return e, nil
		}
	}
	return Entry{}, ErrNotFound(number)
}

func (f *File) Mark(number int, to Status, at time.Time) (Entry, error) {
	return f.update(number, func(e *Entry) {
		if e.Status == to {
			return
		}
		e.History = append(e.History, Event{At: at, From: e.Status, To: to})
		e.Status = to
	})
}

func (f *File) SetNote(number int, note string) (Entry, error) {
	return f.update(number, func(e *Entry) { e.Note = strings.TrimSpace(note) })
}

func (f *File) List(statuses ...Status) ([]Entry, error) {
	d, err := f.load()
	if err != nil {
		return nil, err
	}
	var out []Entry
	for _, e := range d.Entries {
		if len(statuses) == 0 || slices.Contains(statuses, e.Status) {
			out = append(out, e)
		}
	}
	slices.SortFunc(out, func(a, b Entry) int { return a.Number - b.Number })
	return out, nil
}
