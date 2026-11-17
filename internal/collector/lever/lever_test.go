package lever

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/victor-novakoski/garimpo/internal/job"
)

type fakeGetter struct {
	body []byte
	url  string
}

func (f *fakeGetter) Get(_ context.Context, u string) ([]byte, error) {
	f.url = u
	return f.body, nil
}

func TestCollect(t *testing.T) {
	body, err := os.ReadFile("testdata/site.json")
	if err != nil {
		t.Fatal(err)
	}
	g := &fakeGetter{body: body}
	c := Collector{Site: Site{Slug: "neon", Company: "Neon"}, Client: g}
	jobs, err := c.Collect(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if g.url != "https://api.lever.co/v0/postings/neon?mode=json" {
		t.Errorf("url = %s", g.url)
	}
	if len(jobs) != 2 {
		t.Fatalf("got %d jobs, want 2 (the one in France is left out)", len(jobs))
	}
	j := jobs[0]
	if j.Key() != "lever:neon:5cae0695-01bc-4b65-81a9-11fcba5c53ba" || j.Company != "Neon" || j.Location != "Remoto" {
		t.Errorf("job = %+v", j)
	}
	if j.Title != "Pessoa Engenheira de Software Pleno" || j.WorkMode != job.WorkModeRemote {
		t.Errorf("Title = %q, WorkMode = %q", j.Title, j.WorkMode)
	}
	want := "Sobre a Neon\n\nSomos uma fintech.\n\nRequisitos\n- Go ou Java\n- Kafka & Postgres\n\nBenefícios: vale refeição."
	if j.Description != want {
		t.Errorf("Description = %q, want %q", j.Description, want)
	}
	if len(j.Tags) != 3 || j.Tags[0] != "Tecnologia" || j.Tags[2] != "CLT" {
		t.Errorf("Tags = %v", j.Tags)
	}
	if !j.PostedAt.Equal(time.UnixMilli(1792708355883)) {
		t.Errorf("PostedAt = %v", j.PostedAt)
	}
	other := jobs[1]
	if other.WorkMode != job.WorkModeUnknown || !other.PostedAt.IsZero() {
		t.Errorf("job without mode or date = %+v", other)
	}
}

func TestCollectBadJSON(t *testing.T) {
	c := Collector{Site: Site{Slug: "neon"}, Client: &fakeGetter{body: []byte(`{"ok":false}`)}}
	if _, err := c.Collect(context.Background()); err == nil {
		t.Error("want an error for a body that is not a list")
	}
}
