package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/victor-novakoski/garimpo/internal/db"
	"github.com/victor-novakoski/garimpo/internal/db/dbtest"
	"github.com/victor-novakoski/garimpo/internal/job"
	"github.com/victor-novakoski/garimpo/internal/tracker"
)

const secret = "0123456789abcdef0123456789abcdef"

var (
	day1 = time.Date(2026, 11, 11, 9, 0, 0, 0, time.UTC)
	day2 = day1.AddDate(0, 0, 1)
	day3 = day1.AddDate(0, 0, 2)
)

func seen(id, title string, score int) tracker.Seen {
	return tracker.Seen{
		Job:     job.Job{Source: "s", SourceID: id, Title: title, Company: "Acme", URL: "https://x/" + id, Description: "Texto da vaga " + id, WorkMode: job.WorkModeRemote},
		Score:   score,
		Reasons: []string{"+20 Go no título"},
	}
}

func syncRun(t *testing.T, store tracker.Store, at time.Time, listed ...tracker.Seen) {
	t.Helper()
	run := tracker.Run{At: at, Listed: listed, Present: map[string]bool{}, Sources: map[string]bool{"s": true}}
	for _, s := range listed {
		run.Present[s.Job.Key()] = true
	}
	if _, err := store.Sync(run); err != nil {
		t.Fatal(err)
	}
}

// client calls the API in memory, as the owner unless a token is given.
type client struct {
	t       *testing.T
	handler http.Handler
	token   string
}

func newClient(t *testing.T, store tracker.Store) *client {
	t.Helper()
	auth, err := NewToken(secret)
	if err != nil {
		t.Fatal(err)
	}
	s := &server{store: store, now: func() time.Time { return day3 }, logf: t.Logf}
	return &client{t: t, handler: newServer(s, auth), token: secret}
}

func (c *client) do(method, path, body string) (int, map[string]any) {
	c.t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	rec := httptest.NewRecorder()
	c.handler.ServeHTTP(rec, req)
	var out map[string]any
	if strings.HasPrefix(rec.Header().Get("Content-Type"), "application/json") {
		if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
			c.t.Fatalf("%s %s: body is not JSON: %q", method, path, rec.Body.String())
		}
	}
	return rec.Code, out
}

// numbers lists the "number new_today" of each job of a list answer.
func numbers(t *testing.T, out map[string]any) string {
	t.Helper()
	var parts []string
	for _, item := range out["jobs"].([]any) {
		j := item.(map[string]any)
		part := "#" + jsonNumber(j["number"])
		if j["new_today"] == true {
			part += " nova hoje"
		}
		parts = append(parts, part)
	}
	return strings.Join(parts, "|")
}

func jsonNumber(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

// stores runs a test over the file and, when configured, the database.
func stores(t *testing.T, test func(t *testing.T, store tracker.Store)) {
	t.Run("file", func(t *testing.T) {
		test(t, tracker.OpenFile(filepath.Join(t.TempDir(), "vagas.json")))
	})
	t.Run("postgres", func(t *testing.T) {
		ctx := context.Background()
		pool, err := db.Open(ctx, dbtest.URL(t))
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(pool.Close)
		if _, err := db.Migrate(ctx, pool); err != nil {
			t.Fatal(err)
		}
		test(t, tracker.OpenPostgres(pool))
	})
}

func TestTheFlowOverHTTP(t *testing.T) { stores(t, testTheFlowOverHTTP) }

func testTheFlowOverHTTP(t *testing.T, store tracker.Store) {
	c := newClient(t, store)
	// Day 1: four postings. Day 2: #4 left its source and #5 arrived.
	syncRun(t, store, day1, seen("1", "Dev Go", 50), seen("2", "Dev React", 60), seen("3", "Dev Java", 30), seen("4", "Dev PHP", 20))
	syncRun(t, store, day2, seen("1", "Dev Go", 50), seen("2", "Dev React", 60), seen("3", "Dev Java", 30), seen("5", "Dev Node", 55))

	// An empty list is a list, not null.
	if code, out := c.do("GET", "/jobs?status=interview", ""); code != 200 || out["jobs"] == nil || len(out["jobs"].([]any)) != 0 {
		t.Fatalf("empty list = %d %v", code, out)
	}

	// Mark and note, as the panel will.
	code, out := c.do("PATCH", "/jobs/3", `{"status":"saved","note":"  pede inglês  "}`)
	if code != 200 || out["status"] != "saved" || out["status_label"] != "vou aplicar" || out["note"] != "pede inglês" {
		t.Fatalf("patch both = %d %v", code, out)
	}
	if history := out["history"].([]any); len(history) != 1 || history[0].(map[string]any)["to"] != "saved" {
		t.Errorf("history = %v", out["history"])
	}
	// The words of the command line are accepted too.
	if code, out = c.do("PATCH", "/jobs/1", `{"status":"apliquei"}`); code != 200 || out["status"] != "applied" {
		t.Fatalf("patch status = %d %v", code, out)
	}
	if since, _ := time.Parse(time.RFC3339, out["since"].(string)); !since.Equal(day3) {
		t.Errorf("since = %v, want the time of the change", out["since"])
	}

	// Daily list: "vou aplicar" first, then by score; applied (#1) and
	// gone (#4) are out; #5 arrived in the latest run.
	code, out = c.do("GET", "/jobs", "")
	if got, want := numbers(t, out), "#3|#2|#5 nova hoje"; code != 200 || got != want {
		t.Errorf("daily list = %d %q, want %q", code, got, want)
	}
	// Follow-up, in the order asked.
	code, out = c.do("GET", "/jobs?status=saved,applied,interview", "")
	if got, want := numbers(t, out), "#3|#1"; code != 200 || got != want {
		t.Errorf("follow-up = %d %q, want %q", code, got, want)
	}
	// One status lists what left the source too, flagged.
	code, out = c.do("GET", "/jobs?status=new", "")
	if got, want := numbers(t, out), "#2|#4|#5 nova hoje"; code != 200 || got != want {
		t.Errorf("new = %d %q, want %q", code, got, want)
	}
	if gone := out["jobs"].([]any)[1].(map[string]any); gone["gone"] != true {
		t.Errorf("#4 is not flagged as gone: %v", gone)
	}
	if _, out = c.do("GET", "/jobs?status=all", ""); len(out["jobs"].([]any)) != 5 {
		t.Errorf("all = %v", numbers(t, out))
	}

	// Detail: the reasons of the score, the note and the history.
	code, out = c.do("GET", "/jobs/3", "")
	if code != 200 || out["title"] != "Dev Java" || out["company"] != "Acme" || out["url"] != "https://x/3" ||
		jsonNumber(out["score"]) != "30" || out["note"] != "pede inglês" || len(out["history"].([]any)) != 1 {
		t.Fatalf("detail = %d %v", code, out)
	}
	if reasons := out["reasons"].([]any); len(reasons) != 1 || reasons[0] != "+20 Go no título" {
		t.Errorf("reasons = %v", out["reasons"])
	}
	// Only the database keeps the text of the posting.
	if _, keeps := store.(tracker.PostingReader); keeps {
		posting, _ := out["posting"].(map[string]any)
		if posting["description"] != "Texto da vaga 3" || posting["work_mode"] != "remote" || posting["source"] != "s" {
			t.Errorf("posting = %v", out["posting"])
		}
	} else if out["posting"] != nil {
		t.Errorf("posting from a store that has none: %v", out["posting"])
	}

	// An empty note clears it and leaves the status alone.
	if code, out = c.do("PATCH", "/jobs/3", `{"note":""}`); code != 200 || out["note"] != "" || out["status"] != "saved" {
		t.Errorf("clear note = %d %v", code, out)
	}
}

func TestBadRequests(t *testing.T) {
	store := tracker.OpenFile(filepath.Join(t.TempDir(), "vagas.json"))
	syncRun(t, store, day1, seen("1", "Dev Go", 50))
	c := newClient(t, store)
	for _, tc := range []struct {
		method, path, body string
		want               int
	}{
		{"GET", "/jobs/9", "", 404},
		{"PATCH", "/jobs/9", `{"status":"applied"}`, 404},
		{"GET", "/jobs/abc", "", 400},
		{"GET", "/jobs/0", "", 400},
		{"GET", "/jobs?status=contratado", "", 400},
		{"GET", "/jobs?status=", "", 400},
		{"PATCH", "/jobs/1", `{}`, 400},
		{"PATCH", "/jobs/1", `not json`, 400},
		{"PATCH", "/jobs/1", `{"status":"contratado"}`, 400},
		{"PATCH", "/jobs/1", `{"status":"applied","score":100}`, 400},
		{"PATCH", "/jobs/1", `{"status":"applied"}{"status":"new"}`, 400},
		{"PATCH", "/jobs/1", `{"status":"applied","note":"` + strings.Repeat("a", maxNote+1) + `"}`, 400},
		{"PATCH", "/jobs/1", `{"note":"` + strings.Repeat("a", maxBody) + `"}`, 400},
	} {
		code, out := c.do(tc.method, tc.path, tc.body)
		if code != tc.want || out["error"] == nil || out["error"] == "" {
			t.Errorf("%s %s %.40s: %d %v, want %d with an error message", tc.method, tc.path, tc.body, code, out, tc.want)
		}
	}
	// Nothing above changed the posting: a request is checked whole
	// before anything is written.
	if e, err := store.Get(1); err != nil || e.Status != tracker.StatusNew || e.Note != "" || len(e.History) != 0 {
		t.Errorf("a refused request changed the posting: %+v, %v", e, err)
	}
	// Other methods and paths do not exist.
	for _, tc := range []struct{ method, path string }{{"DELETE", "/jobs/1"}, {"POST", "/jobs"}, {"PUT", "/jobs/1"}} {
		if code, _ := c.do(tc.method, tc.path, ""); code != 405 {
			t.Errorf("%s %s = %d, want 405", tc.method, tc.path, code)
		}
	}
	if code, _ := c.do("GET", "/nada", ""); code != 404 {
		t.Errorf("unknown path = %d, want 404", code)
	}
}

func TestOnlyTheOwnerGetsIn(t *testing.T) {
	store := tracker.OpenFile(filepath.Join(t.TempDir(), "vagas.json"))
	syncRun(t, store, day1, seen("1", "Dev Go", 50))
	c := newClient(t, store)
	for _, token := range []string{"", "errado", secret + "x", secret[:len(secret)-1]} {
		c.token = token
		for _, tc := range []struct{ method, path, body string }{
			{"GET", "/jobs", ""},
			{"GET", "/jobs/1", ""},
			{"PATCH", "/jobs/1", `{"status":"applied"}`},
			{"GET", "/nada", ""}, // without the token, every path answers the same
		} {
			code, out := c.do(tc.method, tc.path, tc.body)
			if code != 401 || out["jobs"] != nil || out["title"] != nil {
				t.Errorf("token %q: %s %s = %d %v, want 401", token, tc.method, tc.path, code, out)
			}
		}
	}
	if e, _ := store.Get(1); e.Status != tracker.StatusNew {
		t.Error("a request without the token changed the posting")
	}
	// The scheme matters and its case does not.
	for header, want := range map[string]int{"bearer " + secret: 200, "Basic " + secret: 401, secret: 401} {
		req := httptest.NewRequest("GET", "/jobs", nil)
		req.Header.Set("Authorization", header)
		rec := httptest.NewRecorder()
		c.handler.ServeHTTP(rec, req)
		if rec.Code != want {
			t.Errorf("Authorization %q = %d, want %d", strings.Fields(header)[0], rec.Code, want)
		}
	}
	// Health says nothing personal and needs no token.
	c.token = ""
	if code, out := c.do("GET", "/health", ""); code != 200 || out["status"] != "ok" {
		t.Errorf("health = %d %v", code, out)
	}
}

func TestShortTokenIsRefused(t *testing.T) {
	for _, short := range []string{"", "curto", secret[:MinTokenLength-1]} {
		if _, err := NewToken(short); err == nil {
			t.Errorf("token of %d characters accepted", len(short))
		}
	}
}

// brokenStore fails like a database that went away.
type brokenStore struct{ tracker.Store }

var errBroken = errors.New(`dial tcp db.segredo.supabase.co:5432: connection refused`)

func (brokenStore) List(...tracker.Status) ([]tracker.Entry, error) { return nil, errBroken }
func (brokenStore) Get(int) (tracker.Entry, error)                  { return tracker.Entry{}, errBroken }

func TestStoreErrorsStayInTheLog(t *testing.T) {
	auth, _ := NewToken(secret)
	var logged string
	s := &server{store: brokenStore{}, now: time.Now, logf: func(format string, args ...any) { logged += format + jsonNumber(args[2].(error).Error()) }}
	c := &client{t: t, handler: newServer(s, auth), token: secret}
	for _, path := range []string{"/jobs", "/jobs/1"} {
		code, out := c.do("GET", path, "")
		if code != 500 || strings.Contains(out["error"].(string), "supabase") {
			t.Errorf("%s = %d %v, want 500 without the cause", path, code, out)
		}
	}
	if !strings.Contains(logged, "supabase") {
		t.Errorf("the cause was not logged: %q", logged)
	}
}

func TestListShowsTheFactsOfThePosting(t *testing.T) { stores(t, testListShowsTheFactsOfThePosting) }

func testListShowsTheFactsOfThePosting(t *testing.T, store tracker.Store) {
	s := seen("1", "Dev Go", 50)
	s.Job.Contract, s.Job.Seniority, s.Job.Location = job.ContractPJ, job.SeniorityMid, "Brasil"
	syncRun(t, store, day1, s)
	_, out := newClient(t, store).do("GET", "/jobs", "")
	got := out["jobs"].([]any)[0].(map[string]any)
	if got["work_mode"] != "remote" || got["contract"] != "pj" || got["seniority"] != "mid" || got["location"] != "Brasil" {
		t.Errorf("facts of the posting = %v", got)
	}
}

func TestFunnel(t *testing.T) { stores(t, testFunnel) }

func testFunnel(t *testing.T, store tracker.Store) {
	c := newClient(t, store)
	if code, out := c.do("GET", "/funnel", ""); code != 200 || out["last_run"] != nil || len(out["history"].([]any)) != 0 || len(out["stages"].([]any)) != 5 {
		t.Fatalf("empty funnel = %d %v", code, out)
	}
	// Day 1: four postings. Day 2: #4 left its source.
	syncRun(t, store, day1, seen("1", "Dev Go", 50), seen("2", "Dev React", 60), seen("3", "Dev Java", 30), seen("4", "Dev PHP", 20))
	syncRun(t, store, day2, seen("1", "Dev Go", 50), seen("2", "Dev React", 60), seen("3", "Dev Java", 30))
	for _, step := range []struct {
		number int
		to     tracker.Status
		at     time.Time
	}{{1, tracker.StatusApplied, day2}, {2, tracker.StatusSaved, day2.Add(time.Hour)}, {1, tracker.StatusInterview, day3}} {
		if _, err := store.Mark(step.number, step.to, step.at); err != nil {
			t.Fatal(err)
		}
	}
	code, out := c.do("GET", "/funnel", "")
	if code != 200 || jsonNumber(out["total"]) != "4" {
		t.Fatalf("funnel = %d %v", code, out)
	}
	var got []string
	for _, v := range out["stages"].([]any) {
		stage := v.(map[string]any)
		got = append(got, stage["label"].(string)+" "+jsonNumber(stage["count"])+"/"+jsonNumber(stage["gone"]))
	}
	if want := "nova 2/1|vou aplicar 1/0|apliquei 0/0|entrevista 1/0|descartada 0/0"; strings.Join(got, "|") != want {
		t.Errorf("stages = %q, want %q", strings.Join(got, "|"), want)
	}
	// The history is of every posting, newest first.
	history := out["history"].([]any)
	if len(history) != 3 {
		t.Fatalf("history = %v", history)
	}
	if first := history[0].(map[string]any); first["to"] != "interview" || first["from"] != "applied" || jsonNumber(first["number"]) != "1" || first["title"] != "Dev Go" {
		t.Errorf("newest change = %v", first)
	}
	if last := history[2].(map[string]any); last["to"] != "applied" {
		t.Errorf("oldest change = %v", last)
	}
	if at, _ := time.Parse(time.RFC3339, out["last_run"].(string)); !at.Equal(day2) {
		t.Errorf("last_run = %v, want the latest run", out["last_run"])
	}
}

// localDo calls the API of the local panel with the headers given.
func localDo(t *testing.T, h http.Handler, method, path, host string, headers map[string]string) (int, map[string]any) {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	req.Host = host
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	var out map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("%s %s: not JSON: %q", method, path, rec.Body.String())
	}
	return rec.Code, out
}

func TestLocalPanelNeedsNoTokenButOnlyServesItsOwnPage(t *testing.T) {
	store := tracker.OpenFile(filepath.Join(t.TempDir(), "vagas.json"))
	syncRun(t, store, day1, seen("1", "Dev Go", 50))
	h := NewLocal(store, Options{})
	const host = "127.0.0.1:8080"
	for name, c := range map[string]struct {
		method, host string
		headers      map[string]string
		want         int
	}{
		"curl on the machine":           {"GET", host, nil, 200},
		"the panel reading":             {"GET", host, map[string]string{"Sec-Fetch-Site": "same-origin"}, 200},
		"the panel by the name":         {"GET", "localhost:8080", map[string]string{"Origin": "http://localhost:8080", "Sec-Fetch-Site": "same-origin"}, 200},
		"ipv6 loopback":                 {"GET", "[::1]:8080", nil, 200},
		"the address typed":             {"GET", host, map[string]string{"Sec-Fetch-Site": "none"}, 200},
		"the panel collecting":          {"POST", host, map[string]string{"Origin": "http://" + host, "Sec-Fetch-Site": "same-origin"}, 409},
		"a site reading":                {"GET", host, map[string]string{"Sec-Fetch-Site": "cross-site"}, 403},
		"a site posting":                {"POST", host, map[string]string{"Origin": "https://evil.example", "Sec-Fetch-Site": "cross-site"}, 403},
		"a site, old browser":           {"POST", host, map[string]string{"Origin": "https://evil.example"}, 403},
		"another port of the machine":   {"POST", host, map[string]string{"Origin": "http://127.0.0.1:3000"}, 403},
		"same-site is not same-origin":  {"GET", host, map[string]string{"Sec-Fetch-Site": "same-site"}, 403},
		"a post nobody typed":           {"POST", host, map[string]string{"Sec-Fetch-Site": "none"}, 403},
		"dns rebinding":                 {"GET", "evil.example:8080", nil, 403},
		"rebinding with its own origin": {"POST", "evil.example:8080", map[string]string{"Origin": "http://evil.example:8080", "Sec-Fetch-Site": "same-origin"}, 403},
		"another machine by address":    {"GET", "192.168.0.10:8080", nil, 403},
		"null origin":                   {"POST", host, map[string]string{"Origin": "null"}, 403},
	} {
		path := "/jobs"
		if c.method == "POST" {
			path = "/collect"
		}
		code, out := localDo(t, h, c.method, path, c.host, c.headers)
		if code != c.want {
			t.Errorf("%s: %s %s = %d %v, want %d", name, c.method, path, code, out, c.want)
		}
		if c.want == 403 && (out["jobs"] != nil || out["error"] == nil) {
			t.Errorf("%s: refused request leaked: %v", name, out)
		}
	}
}

func TestCollectFromThePanel(t *testing.T) {
	store := tracker.OpenFile(filepath.Join(t.TempDir(), "vagas.json"))
	const host = "127.0.0.1:8080"

	// Without a runner the button is off and says why.
	off := NewLocal(store, Options{CollectOff: "sem perfil"})
	if code, out := localDo(t, off, "GET", "/collect", host, nil); code != 200 || out["available"] != false || out["reason"] != "sem perfil" {
		t.Errorf("status without a runner = %d %v", code, out)
	}
	if code, _ := localDo(t, off, "POST", "/collect", host, nil); code != 409 {
		t.Errorf("start without a runner = %d, want 409", code)
	}
	// The API behind the token does not collect either.
	if code, out := newClient(t, store).do("POST", "/collect", ""); code != 409 || out["available"] != false {
		t.Errorf("start on the token API = %d %v", code, out)
	}

	release := make(chan struct{})
	fail := false
	runner := NewRunner(context.Background(), func(_ context.Context, log io.Writer) error {
		io.WriteString(log, "fonte-a 3 vagas\n")
		<-release
		if fail {
			return errors.New("a coleta falhou")
		}
		syncRun(t, store, day1, seen("1", "Dev Go", 50))
		return nil
	})
	h := NewLocal(store, Options{Collect: runner})
	if code, out := localDo(t, h, "GET", "/collect", host, nil); code != 200 || out["available"] != true || out["running"] != false || out["started_at"] != nil {
		t.Fatalf("status before = %d %v", code, out)
	}
	if code, out := localDo(t, h, "POST", "/collect", host, nil); code != 202 || out["running"] != true {
		t.Fatalf("start = %d %v", code, out)
	}
	// One at a time.
	if code, out := localDo(t, h, "POST", "/collect", host, nil); code != 409 || out["running"] != true {
		t.Errorf("second start = %d %v, want 409", code, out)
	}
	close(release)
	runner.Wait()
	code, out := localDo(t, h, "GET", "/collect", host, nil)
	if code != 200 || out["running"] != false || out["finished_at"] == nil || out["error"] != nil || out["log"] != "fonte-a 3 vagas\n" {
		t.Errorf("status after = %d %v", code, out)
	}
	if _, out := localDo(t, h, "GET", "/jobs", host, nil); len(out["jobs"].([]any)) != 1 {
		t.Errorf("the collection did not reach the list: %v", out)
	}

	// A failed collection says so, and the next one starts clean.
	release, fail = make(chan struct{}), true
	close(release)
	localDo(t, h, "POST", "/collect", host, nil)
	runner.Wait()
	if _, out := localDo(t, h, "GET", "/collect", host, nil); out["error"] != "a coleta falhou" || out["running"] != false {
		t.Errorf("status after a failure = %v", out)
	}
}

func TestRunLogKeepsTheEnd(t *testing.T) {
	runner := NewRunner(context.Background(), func(_ context.Context, log io.Writer) error {
		for range 3 {
			log.Write([]byte(strings.Repeat("a", maxRunLog/2)))
		}
		io.WriteString(log, "fim")
		return nil
	})
	runner.Start()
	runner.Wait()
	if got := runner.view().Log; len(got) != maxRunLog || !strings.HasSuffix(got, "afim") {
		t.Errorf("log has %d bytes, ending %q", len(got), got[len(got)-4:])
	}
}
