package githubissues

import (
	"context"
	"errors"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"
)

// fakeGetter serves saved responses by page number.
type fakeGetter struct {
	pages map[string][]byte
	urls  []string
	err   error
}

func (f *fakeGetter) Get(_ context.Context, raw string) ([]byte, error) {
	f.urls = append(f.urls, raw)
	if f.err != nil {
		return nil, f.err
	}
	u, _ := url.Parse(raw)
	if b, ok := f.pages[u.Query().Get("page")]; ok {
		return b, nil
	}
	return []byte("[]"), nil
}

func TestCollect(t *testing.T) {
	page1, err := os.ReadFile("testdata/page1.json")
	if err != nil {
		t.Fatal(err)
	}
	g := &fakeGetter{pages: map[string][]byte{"1": page1}}
	since := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	c := Collector{Repo: "backend-br/vagas", Since: since, Client: g}

	jobs, err := c.Collect(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(jobs) != 2 {
		t.Fatalf("got %d jobs, want 2 (pull request skipped)", len(jobs))
	}
	j := jobs[0]
	if j.Source != "github:backend-br/vagas" || j.SourceID != "101" {
		t.Errorf("key = %s", j.Key())
	}
	if j.Company != "Acme Pagamentos" {
		t.Errorf("Company = %q", j.Company)
	}
	if j.URL != "https://github.com/backend-br/vagas/issues/101" {
		t.Errorf("URL = %q", j.URL)
	}
	if strings.Join(j.Tags, ",") != "Remoto,PJ,Pleno,Go" {
		t.Errorf("Tags = %v", j.Tags)
	}
	if !j.PostedAt.Equal(time.Date(2026, 11, 5, 12, 0, 0, 0, time.UTC)) {
		t.Errorf("PostedAt = %v", j.PostedAt)
	}

	if len(g.urls) != 1 {
		t.Fatalf("requests = %d, want 1 (short page ends paging)", len(g.urls))
	}
	q, _ := url.Parse(g.urls[0])
	if q.Host != Host || q.Path != "/repos/backend-br/vagas/issues" {
		t.Errorf("url = %s", g.urls[0])
	}
	if q.Query().Get("since") != "2026-11-01T00:00:00Z" || q.Query().Get("state") != "open" {
		t.Errorf("query = %s", q.RawQuery)
	}
}

func TestCollectStopsAtMaxPages(t *testing.T) {
	full := "[" + strings.Repeat(`{"number":1,"title":"x"},`, perPage-1) + `{"number":1,"title":"x"}]`
	g := &fakeGetter{pages: map[string][]byte{"1": []byte(full), "2": []byte(full), "3": []byte(full)}}
	c := Collector{Repo: "a/b", MaxPages: 2, Client: g}
	if _, err := c.Collect(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(g.urls) != 2 {
		t.Fatalf("requests = %d, want 2", len(g.urls))
	}
}

func TestCollectError(t *testing.T) {
	boom := errors.New("boom")
	c := Collector{Repo: "a/b", Client: &fakeGetter{err: boom}}
	if _, err := c.Collect(context.Background()); !errors.Is(err, boom) {
		t.Fatalf("err = %v", err)
	}
}

func TestCompanyFromTitle(t *testing.T) {
	tests := map[string]string{
		"[Remoto] Dev Go na Acme":            "Acme",
		"Backend Sênior @ Banco X - Híbrido": "Banco X",
		"[PJ] Dev React | Loja Y (Remoto)":   "Loja Y",
		"Desenvolvedor Fullstack Pleno":      "",
		"[Remoto][PJ] Engenheiro no Grupo Z": "Grupo Z",
	}
	for title, want := range tests {
		if got := companyFromTitle(title); got != want {
			t.Errorf("companyFromTitle(%q) = %q, want %q", title, got, want)
		}
	}
}
