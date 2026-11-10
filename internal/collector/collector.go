// Package collector defines how job sources are read. Each source gets its
// own Collector; adding a source means writing one new implementation.
package collector

import (
	"context"
	"errors"
	"fmt"

	"github.com/victor-novakoski/garimpo/internal/job"
)

// Collector fetches open postings from one source.
type Collector interface {
	// Name is the stable source name stored with every job.
	Name() string
	// Collect returns the postings currently open in the source.
	Collect(ctx context.Context) ([]job.Job, error)
}

// Result is what one collector produced in a run.
type Result struct {
	Source string
	Jobs   []job.Job
	Err    error
}

// RunAll runs every collector and keeps going when one fails, so a broken
// source never stops the daily run. The returned error joins all failures.
func RunAll(ctx context.Context, collectors []Collector) ([]Result, error) {
	results := make([]Result, 0, len(collectors))
	var errs []error
	for _, c := range collectors {
		jobs, err := c.Collect(ctx)
		if err != nil {
			err = fmt.Errorf("%s: %w", c.Name(), err)
			errs = append(errs, err)
		}
		results = append(results, Result{Source: c.Name(), Jobs: jobs, Err: err})
	}
	return results, errors.Join(errs...)
}
