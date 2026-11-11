package ai

import (
	"context"
	"errors"
	"slices"
	"sync"

	"github.com/victor-novakoski/garimpo/internal/score"
)

// MaxTotal is the full score: the deterministic part plus the AI part.
const MaxTotal = score.Max + Max // 100

// Ranked is a job that passed the filter, with the AI part when it ran.
type Ranked struct {
	score.Result
	// AI is nil when the job was not assessed: outside the top N, or the
	// assessment failed (AIError says why).
	AI      *Assessment `json:"ai,omitempty"`
	AIError string      `json:"ai_error,omitempty"`
	Total   int         `json:"total"`
}

// workers is how many jobs are assessed at once: enough to finish a daily
// run in a couple of minutes without hammering the API or the plan.
const workers = 3

// Rank assesses the top jobs of results (already sorted, best first) and
// returns everything sorted by total. Jobs beyond top keep their
// deterministic points. A failed assessment never removes a job: it stays
// with what the rules gave it and the error is recorded.
//
// An ErrUnavailable from the scorer stops the remaining assessments and is
// returned along with the ranking, so the caller can say it once.
func Rank(ctx context.Context, s Scorer, results []score.Result, top int, progress func(done, total int)) ([]Ranked, error) {
	ranked := make([]Ranked, len(results))
	for i, r := range results {
		ranked[i] = Ranked{Result: r, Total: r.Points}
	}
	n := min(max(top, 0), len(ranked))

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	var (
		mu    sync.Mutex
		fatal error
		done  int
		wg    sync.WaitGroup
		next  = make(chan int)
	)
	for range min(workers, n) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range next {
				a, err := s.Assess(ctx, ranked[i].Job)
				mu.Lock()
				switch {
				case err == nil:
					ranked[i].AI = &a
					ranked[i].Total += a.Points()
				case errors.Is(err, ErrUnavailable):
					if fatal == nil {
						fatal = err
						cancel()
					}
				case ctx.Err() == nil:
					ranked[i].AIError = err.Error()
				}
				done++
				if progress != nil {
					progress(done, n)
				}
				mu.Unlock()
			}
		}()
	}
feed:
	for i := range n {
		select {
		case next <- i:
		case <-ctx.Done():
			break feed
		}
	}
	close(next)
	wg.Wait()

	slices.SortStableFunc(ranked, func(a, b Ranked) int { return b.Total - a.Total })
	return ranked, fatal
}
