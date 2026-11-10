// Package githubissues collects postings published as issues in community
// job repositories such as backend-br/vagas, through the official GitHub
// REST API.
package githubissues

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/victor-novakoski/garimpo/internal/httpx"
	"github.com/victor-novakoski/garimpo/internal/job"
	"github.com/victor-novakoski/garimpo/internal/normalize"
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
	title := strings.TrimSpace(unescape(is.Title))
	tags := make([]string, 0, len(is.Labels))
	for _, l := range is.Labels {
		tags = append(tags, l.Name)
	}
	return job.Job{
		Source:      c.Name(),
		SourceID:    strconv.Itoa(is.Number),
		URL:         is.HTMLURL,
		Title:       title,
		Company:     companyFromTitle(title),
		Description: is.Body,
		Tags:        tags,
		PostedAt:    is.CreatedAt,
	}
}

// companyFromTitle reads the company from the common title shapes:
// "Cargo na Empresa", "Cargo @ Empresa", "Cargo | Empresa | detalhe",
// "Cargo - Empresa" and "Cargo (Empresa)". It returns "" when unsure, which
// only turns off cross-source dedupe for that job.
func companyFromTitle(title string) string {
	t := strings.TrimSpace(stripLeadingTags(title))
	// Strongest markers first; for these the company is what follows the
	// last one.
	for _, sep := range []string{"@", " na ", " no "} {
		if i := strings.LastIndex(t, sep); i >= 0 {
			if c := cleanCompany(t[i+len(sep):]); c != "" {
				return c
			}
		}
	}
	// "Cargo | Empresa | 4 dias": the company is the first valid segment
	// after the job title.
	if parts := strings.Split(t, " | "); len(parts) > 1 {
		for _, p := range parts[1:] {
			if c := cleanCompany(p); c != "" {
				return c
			}
		}
	}
	// "Cargo - Empresa": the company is the last segment.
	for _, sep := range []string{" - ", " — ", " – "} {
		if i := strings.LastIndex(t, sep); i >= 0 {
			if c := cleanCompany(t[i+len(sep):]); c != "" {
				return c
			}
		}
	}
	// "Cargo (Empresa)" or "Cargo- (Empresa)".
	if strings.HasSuffix(t, ")") {
		if i := strings.LastIndex(t, "("); i >= 0 {
			return cleanCompany(t[i+1 : len(t)-1])
		}
	}
	return ""
}

// stripLeadingTags drops "[Remoto] [PJ]" style prefixes.
func stripLeadingTags(t string) string {
	for {
		t = strings.TrimSpace(t)
		if !strings.HasPrefix(t, "[") {
			return t
		}
		end := strings.Index(t, "]")
		if end < 0 {
			return t
		}
		t = t[end+1:]
	}
}

// notCompany are words that follow a separator but describe the job.
var notCompany = map[string]bool{
	"remoto": true, "remote": true, "hibrido": true, "presencial": true,
	"pj": true, "clt": true, "freela": true, "junior": true, "jr": true,
	"pleno": true, "senior": true, "sr": true, "brasil": true, "brazil": true,
	"sao paulo": true, "home office": true, "vaga": true, "urgente": true,
	"usd": true, "brl": true, "eur": true, "us$": true, "r$": true,
}

// brazilianStates are UF codes that show up after a dash ("- RJ").
var brazilianStates = map[string]bool{
	"ac": true, "al": true, "ap": true, "am": true, "ba": true, "ce": true, "df": true,
	"es": true, "go": true, "ma": true, "mt": true, "ms": true, "mg": true, "pa": true,
	"pb": true, "pr": true, "pe": true, "pi": true, "rj": true, "rn": true, "rs": true,
	"ro": true, "rr": true, "sc": true, "sp": true, "se": true, "to": true,
}

// stackWords mark a segment that lists technologies, not a company
// ("React.js, Nest.js, Kafka", "Net+React", "Node.js + React").
var stackWords = map[string]bool{
	"react": true, "react.js": true, "node": true, "node.js": true, "nest.js": true,
	"net": true, ".net": true, "java": true, "python": true, "go": true, "golang": true,
	"kafka": true, "typescript": true, "javascript": true, "angular": true, "vue": true,
	"php": true, "ruby": true, "c#": true, "aws": true,
}

func looksLikeStack(s string) bool {
	if strings.ContainsAny(s, "+,") {
		return true
	}
	for _, w := range normalize.Words(s) {
		if stackWords[w] {
			return true
		}
	}
	return false
}

// cleanCompany cuts details after the name ("Aurum | US$ 5k",
// "Acme (PJ)", "Evertec 🧡") and rejects words that are not a company.
func cleanCompany(s string) string {
	if i := strings.IndexAny(s, "|—–([/"); i >= 0 {
		s = s[:i]
	}
	if i := strings.Index(s, " - "); i >= 0 {
		s = s[:i]
	}
	s = strings.TrimFunc(s, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	folded := normalize.Fold(s)
	if s == "" || notCompany[folded] || brazilianStates[folded] || looksLikeStack(s) {
		return ""
	}
	return s
}

// unescape decodes HTML entities until none are left: some titles arrive
// escaped twice ("AI &amp;amp; Agents").
func unescape(s string) string {
	for range 3 {
		u := html.UnescapeString(s)
		if u == s {
			break
		}
		s = u
	}
	return s
}
