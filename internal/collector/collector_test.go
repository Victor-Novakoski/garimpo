package collector

import (
	"context"
	"errors"
	"testing"

	"github.com/victor-novakoski/garimpo/internal/job"
)

type fake struct {
	name string
	jobs []job.Job
	err  error
}

func (f fake) Name() string { return f.name }

func (f fake) Collect(context.Context) ([]job.Job, error) { return f.jobs, f.err }

func TestRunAllKeepsGoingWhenOneFails(t *testing.T) {
	broken := errors.New("site down")
	results, err := RunAll(context.Background(), []Collector{
		fake{name: "a", err: broken},
		fake{name: "b", jobs: []job.Job{{Source: "b", SourceID: "1"}}},
	})
	if !errors.Is(err, broken) {
		t.Fatalf("err = %v, want it to wrap %v", err, broken)
	}
	if len(results) != 2 {
		t.Fatalf("len(results) = %d, want 2", len(results))
	}
	if got := len(results[1].Jobs); got != 1 {
		t.Fatalf("second collector jobs = %d, want 1", got)
	}
}

func TestRunAllNoError(t *testing.T) {
	if _, err := RunAll(context.Background(), []Collector{fake{name: "a"}}); err != nil {
		t.Fatalf("err = %v, want nil", err)
	}
}
