// Package api serves the job flow over HTTP: the same states and rules of
// docs/FLOW.md that the command line follows, for the panel to call. It
// only reads and writes the tracker; collecting stays with the CLI.
package api

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/victor-novakoski/garimpo/internal/tracker"
)

// TokenEnv is the environment variable with the owner's token.
const TokenEnv = "GARIMPO_API_TOKEN"

// MinTokenLength refuses tokens short enough to guess. `openssl rand -hex
// 32` gives 64 characters.
const MinTokenLength = 32

const (
	maxBody = 64 << 10 // bytes of a request body
	maxNote = 4000     // characters of a note
)

// Authenticator decides whether a request comes from the owner. The token
// is the only implementation today; the panel's login (Supabase Auth) can
// take its place without touching the routes.
type Authenticator interface {
	Authenticate(r *http.Request) bool
}

// Token authenticates by a fixed secret sent as "Authorization: Bearer".
type Token struct {
	sum [sha256.Size]byte
}

// NewToken returns the authenticator for a secret, or an error when the
// secret is too short to be one.
func NewToken(secret string) (Token, error) {
	if len(secret) < MinTokenLength {
		return Token{}, fmt.Errorf("%s precisa ter pelo menos %d caracteres (gere um com: openssl rand -hex 32)", TokenEnv, MinTokenLength)
	}
	return Token{sum: sha256.Sum256([]byte(secret))}, nil
}

func (t Token) Authenticate(r *http.Request) bool {
	scheme, got, ok := strings.Cut(r.Header.Get("Authorization"), " ")
	if !ok || !strings.EqualFold(scheme, "Bearer") {
		return false
	}
	// Hashing first makes both sides the same length, so the comparison
	// takes the same time whatever was sent.
	sum := sha256.Sum256([]byte(strings.TrimSpace(got)))
	return subtle.ConstantTimeCompare(sum[:], t.sum[:]) == 1
}

type server struct {
	store tracker.Store
	now   func() time.Time
	logf  func(format string, args ...any)
}

// New returns the API over a store. Every route but /health needs the
// owner.
func New(store tracker.Store, auth Authenticator) http.Handler {
	return newServer(&server{store: store, now: time.Now, logf: log.Printf}, auth)
}

func newServer(s *server, auth Authenticator) http.Handler {
	private := http.NewServeMux()
	private.HandleFunc("GET /jobs", s.list)
	private.HandleFunc("GET /jobs/{number}", s.show)
	private.HandleFunc("PATCH /jobs/{number}", s.update)

	root := http.NewServeMux()
	root.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	// Authentication comes before routing: without the token, every path
	// answers the same and nothing about the API leaks.
	root.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if !auth.Authenticate(r) {
			w.Header().Set("WWW-Authenticate", `Bearer realm="garimpo"`)
			writeError(w, http.StatusUnauthorized, "token ausente ou inválido")
			return
		}
		private.ServeHTTP(w, r)
	})
	return root
}

// jobView is one posting in a list.
type jobView struct {
	Number      int            `json:"number"`
	Title       string         `json:"title"`
	Company     string         `json:"company"`
	URL         string         `json:"url"`
	Status      tracker.Status `json:"status"`
	StatusLabel string         `json:"status_label"`
	Note        string         `json:"note"`
	// Score is the one of the last run that listed the posting.
	Score     int       `json:"score"`
	NewToday  bool      `json:"new_today"`
	Gone      bool      `json:"gone"`
	Since     time.Time `json:"since"` // when it got the current status
	FirstSeen time.Time `json:"first_seen"`
	LastSeen  time.Time `json:"last_seen"`
}

type eventView struct {
	At   time.Time      `json:"at"`
	From tracker.Status `json:"from"`
	To   tracker.Status `json:"to"`
}

// postingView is the text of the posting, when the store keeps it.
type postingView struct {
	Source      string     `json:"source"`
	Location    string     `json:"location"`
	Description string     `json:"description"`
	Tags        []string   `json:"tags"`
	WorkMode    string     `json:"work_mode"`
	Contract    string     `json:"contract"`
	Seniority   string     `json:"seniority"`
	PostedAt    *time.Time `json:"posted_at"`
}

// detailView is one posting with everything known about it.
type detailView struct {
	jobView
	Reasons []string     `json:"reasons"`
	History []eventView  `json:"history"`
	Posting *postingView `json:"posting"`
}

func view(e tracker.Entry, fresh map[int]bool) jobView {
	return jobView{
		Number: e.Number, Title: e.Title, Company: e.Company, URL: e.URL,
		Status: e.Status, StatusLabel: e.Status.Label(), Note: e.Note, Score: e.Score,
		NewToday: fresh[e.Number], Gone: e.Gone,
		// Always UTC, whatever zone the store or the machine uses.
		Since: e.Since().UTC(), FirstSeen: e.FirstSeen.UTC(), LastSeen: e.LastSeen.UTC(),
	}
}

// parseStatuses reads the status parameter: names separated by commas, in
// the code form ("applied") or as a person writes them ("apliquei").
func parseStatuses(param string) ([]tracker.Status, error) {
	var out []tracker.Status
	for _, word := range strings.Split(param, ",") {
		if word = strings.TrimSpace(word); word == "" {
			continue
		}
		if word == "all" || word == "tudo" {
			return tracker.Statuses(), nil
		}
		status, err := tracker.ParseStatus(word)
		if err != nil {
			return nil, err
		}
		if !slices.Contains(out, status) {
			out = append(out, status)
		}
	}
	if len(out) == 0 {
		return nil, errors.New("status vazio: informe um ou mais estados separados por vírgula, ou tire o parâmetro para ver a lista do dia")
	}
	return out, nil
}

// list answers GET /jobs. Without parameters it is the daily list: what is
// still to decide, "vou aplicar" first and then by score, without what
// left its source. With ?status= it is every posting in those states, in
// the order they were asked and then by number.
func (s *server) list(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	daily := !query.Has("status")
	var want []tracker.Status
	if !daily {
		var err error
		if want, err = parseStatuses(query.Get("status")); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
	}
	// Everything is read even when only some is shown: "nova hoje" depends
	// on the latest run, and a job search has a few hundred entries.
	all, err := s.store.List()
	if err != nil {
		s.fail(w, r, err)
		return
	}
	fresh := tracker.NewToday(all)
	jobs := []jobView{}
	for _, e := range all {
		switch {
		case daily && (!e.Status.InDailyList() || e.Gone):
		case !daily && !slices.Contains(want, e.Status):
		default:
			jobs = append(jobs, view(e, fresh))
		}
	}
	if daily {
		slices.SortStableFunc(jobs, func(a, b jobView) int {
			if a.Status != b.Status {
				if a.Status == tracker.StatusSaved {
					return -1
				}
				return 1
			}
			return b.Score - a.Score
		})
	} else {
		slices.SortStableFunc(jobs, func(a, b jobView) int {
			return slices.Index(want, a.Status) - slices.Index(want, b.Status)
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"jobs": jobs})
}

func (s *server) number(w http.ResponseWriter, r *http.Request) (int, bool) {
	n, err := strconv.Atoi(r.PathValue("number"))
	if err != nil || n <= 0 {
		writeError(w, http.StatusBadRequest, "número de vaga inválido: use o número da lista, como /jobs/12")
		return 0, false
	}
	return n, true
}

// show answers GET /jobs/{number}.
func (s *server) show(w http.ResponseWriter, r *http.Request) {
	n, ok := s.number(w, r)
	if !ok {
		return
	}
	e, err := s.store.Get(n)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	s.detail(w, r, e)
}

func (s *server) detail(w http.ResponseWriter, r *http.Request, e tracker.Entry) {
	all, err := s.store.List()
	if err != nil {
		s.fail(w, r, err)
		return
	}
	d := detailView{jobView: view(e, tracker.NewToday(all)), Reasons: e.Reasons, History: []eventView{}}
	if d.Reasons == nil {
		d.Reasons = []string{}
	}
	for _, ev := range e.History {
		d.History = append(d.History, eventView{At: ev.At.UTC(), From: ev.From, To: ev.To})
	}
	if reader, ok := s.store.(tracker.PostingReader); ok {
		j, found, err := reader.Posting(e.Number)
		if err != nil {
			s.fail(w, r, err)
			return
		}
		if found {
			p := &postingView{
				Source: j.Source, Location: j.Location, Description: j.Description, Tags: j.Tags,
				WorkMode: string(j.WorkMode), Contract: string(j.Contract), Seniority: string(j.Seniority),
			}
			if p.Tags == nil {
				p.Tags = []string{}
			}
			if !j.PostedAt.IsZero() {
				posted := j.PostedAt.UTC()
				p.PostedAt = &posted
			}
			d.Posting = p
		}
	}
	writeJSON(w, http.StatusOK, d)
}

// update answers PATCH /jobs/{number}: {"status": "applied"} marks,
// {"note": "..."} writes the note (an empty one clears it), and both can
// come together.
func (s *server) update(w http.ResponseWriter, r *http.Request) {
	n, ok := s.number(w, r)
	if !ok {
		return
	}
	var body struct {
		Status *string `json:"status"`
		Note   *string `json:"note"`
	}
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBody))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, `corpo inválido: envie um JSON com "status", "note" ou os dois`)
		return
	}
	if dec.More() {
		writeError(w, http.StatusBadRequest, "corpo inválido: envie um único objeto JSON")
		return
	}
	if body.Status == nil && body.Note == nil {
		writeError(w, http.StatusBadRequest, `nada para mudar: envie "status", "note" ou os dois`)
		return
	}
	// Everything is checked before anything is written, so a bad note
	// never leaves the status changed alone.
	var status tracker.Status
	if body.Status != nil {
		var err error
		if status, err = tracker.ParseStatus(*body.Status); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
	}
	if body.Note != nil && utf8.RuneCountInString(*body.Note) > maxNote {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("anotação longa demais: o limite é de %d caracteres", maxNote))
		return
	}
	var (
		e   tracker.Entry
		err error
	)
	if body.Status != nil {
		if e, err = s.store.Mark(n, status, s.now()); err != nil {
			s.fail(w, r, err)
			return
		}
	}
	if body.Note != nil {
		if e, err = s.store.SetNote(n, *body.Note); err != nil {
			s.fail(w, r, err)
			return
		}
	}
	s.detail(w, r, e)
}

// fail answers an error from the store. Only "not found" is told to the
// caller; the rest goes to the log, because a database error can name
// tables, hosts and values.
func (s *server) fail(w http.ResponseWriter, r *http.Request, err error) {
	var notFound tracker.ErrNotFound
	if errors.As(err, &notFound) {
		writeError(w, http.StatusNotFound, fmt.Sprintf("não existe vaga #%d", int(notFound)))
		return
	}
	s.logf("%s %s: %v", r.Method, r.URL.Path, err)
	writeError(w, http.StatusInternalServerError, "erro interno: veja o log do servidor")
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	h := w.Header()
	h.Set("Content-Type", "application/json; charset=utf-8")
	h.Set("Cache-Control", "no-store") // notes and statuses are personal
	h.Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, code int, message string) {
	writeJSON(w, code, map[string]string{"error": message})
}
