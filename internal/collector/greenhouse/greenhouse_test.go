package greenhouse

import (
	"context"
	"os"
	"testing"
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
	body, err := os.ReadFile("testdata/board.json")
	if err != nil {
		t.Fatal(err)
	}
	g := &fakeGetter{body: body}
	c := Collector{Board: Board{Token: "stone", Company: "Stone"}, Client: g}
	jobs, err := c.Collect(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if g.url != "https://boards-api.greenhouse.io/v1/boards/stone/jobs?content=true" {
		t.Errorf("url = %s", g.url)
	}
	if len(jobs) != 2 {
		t.Fatalf("got %d jobs, want 2", len(jobs))
	}
	j := jobs[0]
	if j.Key() != "greenhouse:stone:4012345" || j.Company != "Stone" || j.Location != "Remote - Brazil" {
		t.Errorf("job = %+v", j)
	}
	want := "Time de pagamentos.\n\n- Go\n- Kafka & Postgres"
	if j.Description != want {
		t.Errorf("Description = %q, want %q", j.Description, want)
	}
	if len(j.Tags) != 1 || j.Tags[0] != "Engineering" {
		t.Errorf("Tags = %v", j.Tags)
	}
}
