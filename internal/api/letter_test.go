package api

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/victor-novakoski/garimpo/internal/ai"
	"github.com/victor-novakoski/garimpo/internal/job"
	"github.com/victor-novakoski/garimpo/internal/tracker"
)

func TestLetterDraft(t *testing.T) { stores(t, testLetterDraft) }

func testLetterDraft(t *testing.T, store tracker.Store) {
	const host = "127.0.0.1:8080"
	page := map[string]string{"Origin": "http://" + host, "Sec-Fetch-Site": "same-origin"}
	syncRun(t, store, day1, seen("1", "Dev Go", 50))

	// Without the AI the button is off, the detail says why, and the route
	// answers the same reason.
	off := NewLocal(store, Options{LetterOff: "sem IA"})
	if code, out := localDo(t, off, "GET", "/jobs/1", host, nil); code != 200 || out["letter_off"] != "sem IA" {
		t.Errorf("detail without a writer = %d %v", code, out["letter_off"])
	}
	if code, out := localDo(t, off, "POST", "/jobs/1/letter", host, page); code != 409 || out["error"] != "sem IA" {
		t.Errorf("letter without a writer = %d %v", code, out)
	}
	// The API behind the token writes none either.
	if code, out := newClient(t, store).do("POST", "/jobs/1/letter", ""); code != 409 || out["letter"] != nil {
		t.Errorf("letter on the token API = %d %v", code, out)
	}

	var (
		got     job.Job
		reasons []string
		fail    error
		calls   int
	)
	h := NewLocal(store, Options{Letter: func(_ context.Context, j job.Job, r []string) (string, error) {
		calls++
		got, reasons = j, r
		return "Olá, time da Acme,\n\n<b>Trabalho</b> com Go.", fail
	}})
	if code, out := localDo(t, h, "GET", "/jobs/1", host, nil); code != 200 || out["letter_off"] != "" {
		t.Errorf("detail with a writer = %d %v", code, out["letter_off"])
	}
	code, out := localDo(t, h, "POST", "/jobs/1/letter", host, page)
	if code != 200 || out["letter"] != "Olá, time da Acme,\n\n<b>Trabalho</b> com Go." {
		t.Fatalf("letter = %d %v", code, out)
	}
	if got.Title != "Dev Go" || got.Company != "Acme" || len(reasons) != 1 || reasons[0] != "+20 Go no título" {
		t.Errorf("writer got %+v, reasons %q", got, reasons)
	}
	// The text of the posting goes along when the store keeps it.
	if _, keeps := store.(tracker.PostingReader); keeps && got.Description != "Texto da vaga 1" {
		t.Errorf("description = %q", got.Description)
	}
	// Asking for a letter changes nothing about the posting.
	if _, after := localDo(t, h, "GET", "/jobs/1", host, nil); after["status"] != "new" || len(after["history"].([]any)) != 0 || after["note"] != "" {
		t.Errorf("the letter changed the posting: %v", after)
	}

	calls = 0
	for path, want := range map[string]int{"/jobs/99/letter": 404, "/jobs/x/letter": 400} {
		if code, _ := localDo(t, h, "POST", path, host, page); code != want {
			t.Errorf("POST %s = %d, want %d", path, code, want)
		}
	}
	// Another site cannot spend a call to the model.
	if code, _ := localDo(t, h, "POST", "/jobs/1/letter", host, map[string]string{"Origin": "https://evil.example", "Sec-Fetch-Site": "cross-site"}); code != 403 {
		t.Errorf("cross-site letter = %d, want 403", code)
	}
	if calls != 0 {
		t.Errorf("the writer ran %d times for requests that should not reach it", calls)
	}

	for name, c := range map[string]struct {
		err  error
		code int
		say  string
	}{
		"no login":  {fmt.Errorf("%w: faça login", ai.ErrUnavailable), 503, "faça login"},
		"too slow":  {context.DeadlineExceeded, 504, "demorou demais"},
		"bad reply": {errors.New("a IA devolveu uma carta vazia"), 502, "carta vazia"},
	} {
		fail = c.err
		code, out := localDo(t, h, "POST", "/jobs/1/letter", host, page)
		if code != c.code || out["letter"] != nil || !strings.Contains(fmt.Sprint(out["error"]), c.say) {
			t.Errorf("%s = %d %v, want %d with %q", name, code, out, c.code, c.say)
		}
	}
}

func TestOneLetterAtATime(t *testing.T) {
	store := tracker.OpenFile(filepath.Join(t.TempDir(), "vagas.json"))
	syncRun(t, store, day1, seen("1", "Dev Go", 50))
	const host = "127.0.0.1:8080"
	started, release := make(chan struct{}), make(chan struct{})
	h := NewLocal(store, Options{Letter: func(context.Context, job.Job, []string) (string, error) {
		close(started)
		<-release
		return "Olá,", nil
	}})
	first := make(chan int)
	go func() {
		code, _ := localDo(t, h, "POST", "/jobs/1/letter", host, nil)
		first <- code
	}()
	<-started
	if code, out := localDo(t, h, "POST", "/jobs/1/letter", host, nil); code != 409 || out["letter"] != nil {
		t.Errorf("second letter = %d %v, want 409", code, out)
	}
	close(release)
	if code := <-first; code != 200 {
		t.Errorf("first letter = %d", code)
	}
}
