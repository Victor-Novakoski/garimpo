package greenhouse

import (
	"context"
	"os"
	"testing"

	"github.com/victor-novakoski/garimpo/internal/job"
	"github.com/victor-novakoski/garimpo/internal/normalize"
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

// TestWorkMode runs cuts of real postings from the six boards of issue #18
// (read on 16/11/2026) through the collector and the normalizer, the same
// path `collect` takes. Each company states its work mode in a different
// place, and two of them only in some postings.
func TestWorkMode(t *testing.T) {
	body, err := os.ReadFile("testdata/work_mode.json")
	if err != nil {
		t.Fatal(err)
	}
	c := Collector{Board: Board{Token: "mixed", Company: "Mixed"}, Client: &fakeGetter{body: body}}
	jobs, err := c.Collect(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]struct {
		collected  job.WorkMode // from the board's own field
		classified job.WorkMode // after the title, location and description
	}{
		// Wildlife: the custom field "Work Model".
		"8912839002": {job.WorkModeHybrid, job.WorkModeHybrid},
		"8635029002": {job.WorkModeHybrid, job.WorkModeHybrid}, // "Híbrido/Hybrid"
		"8375546002": {job.WorkModeOnsite, job.WorkModeOnsite}, // "In-Office"
		"8797800002": {job.WorkModeOnsite, job.WorkModeOnsite}, // "Presencial/In-Office"
		"8656067002": {"", ""},                                 // field left empty
		// QuintoAndar: the "remote-first" paragraph, in English and Portuguese.
		"4023099009": {"", job.WorkModeRemote},
		"4362525009": {"", job.WorkModeRemote},
		// XP: the "Presencial Flexível" paragraph, unless the posting asks for 100%.
		"8696050002": {"", job.WorkModeHybrid},
		"8198058002": {"", job.WorkModeOnsite},
		// Inter and C6: a line in the requirements, when there is one.
		"4664774005": {"", job.WorkModeOnsite},
		"4724343005": {"", job.WorkModeOnsite},
		"4725043005": {"", ""},
		// BTG: only the location, and only when it says "Teletrabalho".
		"6177447004": {"", job.WorkModeRemote},
		"6026814004": {"", ""},
	}
	if len(jobs) != len(want) {
		t.Fatalf("got %d jobs, want %d", len(jobs), len(want))
	}
	for _, j := range jobs {
		w, ok := want[j.SourceID]
		if !ok {
			t.Errorf("unexpected job %s", j.SourceID)
			continue
		}
		if j.WorkMode != w.collected {
			t.Errorf("%s %q: collected %q, want %q", j.SourceID, j.Title, j.WorkMode, w.collected)
		}
		if got := normalize.Classify(j).WorkMode; got != w.classified {
			t.Errorf("%s %q: classified %q, want %q", j.SourceID, j.Title, got, w.classified)
		}
	}
}
