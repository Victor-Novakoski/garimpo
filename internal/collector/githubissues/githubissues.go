// Package githubissues collects postings published as issues in community
// job repositories such as backend-br/vagas, through the official GitHub
// REST API.
package githubissues

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/victor-novakoski/garimpo/internal/httpx"
	"github.com/victor-novakoski/garimpo/internal/job"
)

// Host is the only host this collector calls.
const Host = "api.github.com"

const perPage = 100

// Repos are the community job repositories collected by default.
var Repos = []string{
	"backend-br/vagas",
	"frontendbr/vagas",
	"react-brasil/vagas",
}

// Getter is the part of httpx.Client the collector needs.
type Getter interface {
	Get(ctx context.Context, rawURL string) ([]byte, error)
}

// NewClient builds the HTTP client for the GitHub API. token may be empty;
// without it GitHub allows 60 requests per hour.
func NewClient(token string) *httpx.Client {
	h := http.Header{
		"Accept":               {"application/vnd.github+json"},
		"X-Github-Api-Version": {"2022-11-28"},
	}
	if token != "" {
		h.Set("Authorization", "Bearer "+token)
	}
	return httpx.New(httpx.Options{Hosts: []string{Host}, Header: h})
}

// Collector reads open issues from one repository.
type Collector struct {
	Repo     string    // "owner/name"
	Since    time.Time // only issues updated after this; zero means all
	MaxPages int       // safety cap; zero means 5
	Client   Getter
}

// Name is the repository, e.g. "github:backend-br/vagas".
func (c Collector) Name() string { return "github:" + c.Repo }

type issue struct {
	Number      int       `json:"number"`
	Title       string    `json:"title"`
	HTMLURL     string    `json:"html_url"`
	Body        string    `json:"body"`
	CreatedAt   time.Time `json:"created_at"`
	PullRequest *struct{} `json:"pull_request"`
	Labels      []struct {
		Name string `json:"name"`
	} `json:"labels"`
}

// Collect pages through the open issues.
func (c Collector) Collect(ctx context.Context) ([]job.Job, error) {
	maxPages := c.MaxPages
	if maxPages <= 0 {
		maxPages = 5
	}
	var jobs []job.Job
	for page := 1; page <= maxPages; page++ {
		body, err := c.Client.Get(ctx, c.pageURL(page))
		if err != nil {
			return jobs, err
		}
		var issues []issue
		if err := json.Unmarshal(body, &issues); err != nil {
			return jobs, fmt.Errorf("decode page %d: %w", page, err)
		}
		for _, is := range issues {
			if is.PullRequest != nil {
				continue
			}
			jobs = append(jobs, c.toJob(is))
		}
		if len(issues) < perPage {
			break
		}
	}
	return jobs, nil
}

func (c Collector) pageURL(page int) string {
	q := url.Values{
		"state":    {"open"},
		"per_page": {strconv.Itoa(perPage)},
		"page":     {strconv.Itoa(page)},
	}
	if !c.Since.IsZero() {
		q.Set("since", c.Since.UTC().Format(time.RFC3339))
	}
	return fmt.Sprintf("https://%s/repos/%s/issues?%s", Host, c.Repo, q.Encode())
}

func (c Collector) toJob(is issue) job.Job {
	tags := make([]string, 0, len(is.Labels))
	for _, l := range is.Labels {
		tags = append(tags, l.Name)
	}
	return job.Job{
		Source:      c.Name(),
		SourceID:    strconv.Itoa(is.Number),
		URL:         is.HTMLURL,
		Title:       strings.TrimSpace(is.Title),
		Company:     companyFromTitle(is.Title),
		Description: is.Body,
		Tags:        tags,
		PostedAt:    is.CreatedAt,
	}
}

// companyFromTitle reads the company from the common title shapes
// "Cargo na Empresa", "Cargo @ Empresa" and "Cargo | Empresa". It returns
// "" when unsure, which only turns off cross-source dedupe for that job.
func companyFromTitle(title string) string {
	t := title
	// Drop leading tags like "[Remoto] [PJ]".
	for strings.HasPrefix(strings.TrimSpace(t), "[") {
		t = strings.TrimSpace(t)
		end := strings.Index(t, "]")
		if end < 0 {
			break
		}
		t = t[end+1:]
	}
	for _, sep := range []string{" @ ", " na ", " no ", " | "} {
		if i := strings.LastIndex(t, sep); i >= 0 {
			company := strings.TrimSpace(t[i+len(sep):])
			// Cut trailing details: "Acme - Remoto", "Acme (PJ)".
			if j := strings.IndexAny(company, "-–(["); j > 0 {
				company = strings.TrimSpace(company[:j])
			}
			return company
		}
	}
	return ""
}
