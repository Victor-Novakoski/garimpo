package tracker

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// File is a Store kept in one JSON file on the person's machine. It is
// read and written whole: a job search has a few hundred entries.
type File struct {
	path string
	// mu keeps one read-change-write at a time inside a process: the panel
	// marks postings while a collection it started is still running.
	mu sync.Mutex
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

func (f *File) load() (book, error) {
	d := book{Next: 1}
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

func (f *File) save(d book) error {
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
	f.mu.Lock()
	defer f.mu.Unlock()
	d, err := f.load()
	if err != nil {
		return nil, err
	}
	out, _ := d.sync(run)
	return out, f.save(d)
}

func (f *File) update(number int, change func(*Entry)) (Entry, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	d, err := f.load()
	if err != nil {
		return Entry{}, err
	}
	e := d.find(number)
	if e == nil {
		return Entry{}, ErrNotFound(number)
	}
	change(e)
	return *e, f.save(d)
}

func (f *File) Get(number int) (Entry, error) {
	d, err := f.load()
	if err != nil {
		return Entry{}, err
	}
	if e := d.find(number); e != nil {
		return *e, nil
	}
	return Entry{}, ErrNotFound(number)
}

func (f *File) Mark(number int, to Status, at time.Time) (Entry, error) {
	return f.update(number, func(e *Entry) { e.mark(to, at) })
}

func (f *File) SetNote(number int, note string) (Entry, error) {
	return f.update(number, func(e *Entry) { e.Note = cleanNote(note) })
}

func (f *File) List(statuses ...Status) ([]Entry, error) {
	d, err := f.load()
	if err != nil {
		return nil, err
	}
	return d.list(statuses), nil
}
