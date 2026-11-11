package api

import (
	"context"
	"io"
	"net/http"
	"sync"
	"time"
)

// RunFunc is one collection. What it writes to log is shown to the person
// while it runs, so it must not write secrets or database errors there.
type RunFunc func(ctx context.Context, log io.Writer) error

// maxRunLog is how much of the log of a collection is kept: the end of it.
const maxRunLog = 16 << 10

// Runner runs collections for the panel, one at a time. The API never
// chooses what a collection does: the command line builds the RunFunc from
// its own flags, and a request can only start it.
type Runner struct {
	ctx context.Context
	run RunFunc
	now func() time.Time

	mu       sync.Mutex
	running  bool
	started  time.Time
	finished time.Time
	failure  string
	log      []byte
	done     chan struct{}
}

// NewRunner returns a runner for run. Cancelling ctx stops a collection in
// progress.
func NewRunner(ctx context.Context, run RunFunc) *Runner {
	return &Runner{ctx: ctx, run: run, now: time.Now}
}

// Start begins a collection in the background. It reports false, and
// starts nothing, when one is already running.
func (r *Runner) Start() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.running {
		return false
	}
	r.running, r.started, r.finished, r.failure, r.log = true, r.now(), time.Time{}, "", nil
	done := make(chan struct{})
	r.done = done
	go func() {
		defer close(done)
		err := r.run(r.ctx, runLog{r})
		r.mu.Lock()
		defer r.mu.Unlock()
		r.running, r.finished = false, r.now()
		if err != nil {
			r.failure = err.Error()
		}
	}()
	return true
}

// Wait blocks until the collection in progress, if any, ends.
func (r *Runner) Wait() {
	r.mu.Lock()
	done := r.done
	r.mu.Unlock()
	if done != nil {
		<-done
	}
}

// runLog appends to the log of the runner, keeping only the end.
type runLog struct{ r *Runner }

func (l runLog) Write(p []byte) (int, error) {
	l.r.mu.Lock()
	defer l.r.mu.Unlock()
	l.r.log = append(l.r.log, p...)
	if extra := len(l.r.log) - maxRunLog; extra > 0 {
		l.r.log = l.r.log[extra:]
	}
	return len(p), nil
}

// collectView is the state of collecting, for the panel.
type collectView struct {
	// Available is false when this server cannot collect; Reason says why.
	Available bool   `json:"available"`
	Reason    string `json:"reason,omitempty"`
	Running   bool   `json:"running"`
	// StartedAt and FinishedAt are of the latest collection asked here.
	StartedAt  *time.Time `json:"started_at"`
	FinishedAt *time.Time `json:"finished_at"`
	// Error is why the latest collection failed, when it did.
	Error string `json:"error,omitempty"`
	Log   string `json:"log"`
}

func (r *Runner) view() collectView {
	r.mu.Lock()
	defer r.mu.Unlock()
	v := collectView{Available: true, Running: r.running, Error: r.failure, Log: string(r.log)}
	if !r.started.IsZero() {
		at := r.started.UTC()
		v.StartedAt = &at
	}
	if !r.finished.IsZero() {
		at := r.finished.UTC()
		v.FinishedAt = &at
	}
	return v
}

func (s *server) collectView() collectView {
	if s.collect == nil {
		reason := s.collectOff
		if reason == "" {
			reason = "a coleta é feita pelo terminal: garimpo collect"
		}
		return collectView{Reason: reason}
	}
	return s.collect.view()
}

// collectStatus answers GET /collect.
func (s *server) collectStatus(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.collectView())
}

// collectStart answers POST /collect: it starts a collection and returns
// at once. The request carries nothing; the panel follows GET /collect.
func (s *server) collectStart(w http.ResponseWriter, _ *http.Request) {
	if s.collect == nil {
		writeJSON(w, http.StatusConflict, s.collectView())
		return
	}
	if !s.collect.Start() {
		writeJSON(w, http.StatusConflict, s.collectView())
		return
	}
	writeJSON(w, http.StatusAccepted, s.collectView())
}
