package gupy

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/victor-novakoski/garimpo/internal/job"
)

func TestParse(t *testing.T) {
	page, err := os.ReadFile("testdata/search.html")
	if err != nil {
		t.Fatal(err)
	}
	jobs, err := Parse(page)
	if err != nil {
		t.Fatal(err)
	}
	if len(jobs) != 2 {
		t.Fatalf("got %d jobs, want 2", len(jobs))
	}
	j := jobs[0]
	if j.Key() != "gupy:12717450" || j.Company != "Exemplo Tech" {
		t.Errorf("job = %+v", j)
	}
	if j.URL != "https://exemplotech.gupy.io/job/abc123" {
		t.Errorf("URL = %q, want tracking parameter removed", j.URL)
	}
	if j.WorkMode != job.WorkModeRemote || j.Contract != job.ContractPJ {
		t.Errorf("mode/contract = %q/%q", j.WorkMode, j.Contract)
	}
	if jobs[1].Location != "Curitiba, Paraná" || jobs[1].WorkMode != job.WorkModeHybrid || jobs[1].Contract != job.ContractCLT {
		t.Errorf("second job = %+v", jobs[1])
	}
}

func TestParseFormatChanged(t *testing.T) {
	pages := map[string]string{
		"no script":   "<html></html>",
		"bad json":    `<script id="__NEXT_DATA__" type="application/json">{nope</script>`,
		"no list":     `<script id="__NEXT_DATA__" type="application/json">{"props":{"pageProps":{}}}</script>`,
		"missing url": `<script id="__NEXT_DATA__" type="application/json">{"props":{"pageProps":{"initialJobList":{"data":[{"id":1}]}}}}</script>`,
	}
	for name, p := range pages {
		if _, err := Parse([]byte(p)); !errors.Is(err, ErrFormat) {
			t.Errorf("%s: err = %v, want ErrFormat", name, err)
		}
	}
}

type fakeGetter struct {
	page []byte
	urls []string
}

func (f *fakeGetter) Get(_ context.Context, u string) ([]byte, error) {
	f.urls = append(f.urls, u)
	return f.page, nil
}

func TestCollectStopsAtFirstError(t *testing.T) {
	g := &fakeGetter{page: []byte("<html></html>")}
	_, err := Collector{Terms: []string{"golang", "react"}, Client: g}.Collect(context.Background())
	if !errors.Is(err, ErrFormat) {
		t.Fatalf("err = %v", err)
	}
	if len(g.urls) != 1 || g.urls[0] != "https://portal.gupy.io/job-search/term=golang" {
		t.Fatalf("urls = %v", g.urls)
	}
}
