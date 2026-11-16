package api

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/victor-novakoski/garimpo/internal/ai"
	"github.com/victor-novakoski/garimpo/internal/job"
	"github.com/victor-novakoski/garimpo/internal/tracker"
)

// LetterFunc drafts the cover letter of one posting, from the posting and
// the reason lines kept with it. The command line builds it from its own
// flags (profile, AI mode, model); a request can only ask for a posting by
// its number. Its error is shown to the person, so it must not carry
// secrets.
type LetterFunc func(ctx context.Context, j job.Job, reasons []string) (string, error)

// letterTimeout is how long one letter may take. It is longer than the
// write timeout of the server, which the route extends for itself.
const letterTimeout = 2 * time.Minute

const letterOffDefault = "a carta é escrita pelo painel: garimpo serve -ai local"

// letterOffReason is why this server writes no letter, or "" when it does.
func (s *server) letterOffReason() string {
	switch {
	case s.letter != nil:
		return ""
	case s.letterOff != "":
		return s.letterOff
	}
	return letterOffDefault
}

// postingOf gathers what is known of a posting for the letter: the text,
// when the store keeps it, and otherwise the facts of the entry.
func (s *server) postingOf(e tracker.Entry) (job.Job, error) {
	if reader, ok := s.store.(tracker.PostingReader); ok {
		j, found, err := reader.Posting(e.Number)
		if err != nil {
			return job.Job{}, err
		}
		if found {
			return j, nil
		}
	}
	return job.Job{
		Title: e.Title, Company: e.Company, URL: e.URL, Location: e.Location,
		WorkMode: e.WorkMode, Contract: e.Contract, Seniority: e.Seniority,
	}, nil
}

// letterDraft answers POST /jobs/{number}/letter with a draft of the cover
// letter, as plain text in JSON. The request carries nothing, nothing is
// saved and nothing is sent anywhere: the person copies the text, edits it
// and applies by themselves. One letter at a time, because each one costs
// a call to the model.
func (s *server) letterDraft(w http.ResponseWriter, r *http.Request) {
	n, ok := s.number(w, r)
	if !ok {
		return
	}
	if s.letter == nil {
		writeError(w, http.StatusConflict, s.letterOffReason())
		return
	}
	e, err := s.store.Get(n)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	j, err := s.postingOf(e)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	if !s.letterBusy.TryLock() {
		writeError(w, http.StatusConflict, "já há uma carta sendo escrita: espere ela terminar")
		return
	}
	defer s.letterBusy.Unlock()

	ctx, cancel := context.WithTimeout(r.Context(), letterTimeout)
	defer cancel()
	// A recorder in a test has no deadline to move; that error is ignored.
	_ = http.NewResponseController(w).SetWriteDeadline(time.Now().Add(letterTimeout + 10*time.Second))

	text, err := s.letter(ctx, j, e.Reasons)
	switch {
	case err == nil:
		writeJSON(w, http.StatusOK, map[string]string{"letter": text})
	case errors.Is(err, context.DeadlineExceeded):
		s.logf("%s %s: %v", r.Method, r.URL.Path, err)
		writeError(w, http.StatusGatewayTimeout, "a IA demorou demais para escrever a carta: tente de novo")
	case errors.Is(err, ai.ErrUnavailable):
		s.logf("%s %s: %v", r.Method, r.URL.Path, err)
		writeError(w, http.StatusServiceUnavailable, err.Error())
	default:
		s.logf("%s %s: %v", r.Method, r.URL.Path, err)
		writeError(w, http.StatusBadGateway, "não consegui escrever a carta: "+err.Error())
	}
}
