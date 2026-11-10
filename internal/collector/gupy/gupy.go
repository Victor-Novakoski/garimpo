// Package gupy reads Gupy's public job search page for personal alerts.
//
// Gupy's terms forbid aggregating its content (see docs/SOURCES.md), so
// this collector is off by default, keeps only what scoring needs, always
// points to the original posting and must never feed anything published.
// The page has no documented format: the collector stops at the first
// surprise instead of guessing.
package gupy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/victor-novakoski/garimpo/internal/httpx"
	"github.com/victor-novakoski/garimpo/internal/job"
)

// Host is the only host this collector calls.
const Host = "portal.gupy.io"

// ErrFormat means the page no longer looks like the one this collector
// was written for.
var ErrFormat = errors.New("gupy page format changed")

// Getter is the part of httpx.Client the collector needs.
type Getter interface {
	Get(ctx context.Context, rawURL string) ([]byte, error)
}

// NewClient builds the HTTP client for Gupy, with a longer pause between
// requests than the default.
func NewClient() *httpx.Client {
	return httpx.New(httpx.Options{Hosts: []string{Host}, Interval: 3 * time.Second})
}

// Collector searches Gupy for a few terms, one page each.
type Collector struct {
	Terms  []string // e.g. "golang", "react"; keep it short
	Client Getter
}

// Name is "gupy".
func (Collector) Name() string { return "gupy" }

var nextDataRe = regexp.MustCompile(`(?s)<script id="__NEXT_DATA__" type="application/json"[^>]*>(.*?)</script>`)

type nextData struct {
	Props struct {
		PageProps struct {
			InitialJobList *struct {
				Data []gupyJob `json:"data"`
			} `json:"initialJobList"`
		} `json:"pageProps"`
	} `json:"props"`
}

type gupyJob struct {
	ID             int64     `json:"id"`
	Name           string    `json:"name"`
	CareerPageName string    `json:"careerPageName"`
	Type           string    `json:"type"`
	WorkplaceType  string    `json:"workplaceType"`
	City           string    `json:"city"`
	State          string    `json:"state"`
	PublishedDate  time.Time `json:"publishedDate"`
	Description    string    `json:"description"`
	JobURL         string    `json:"jobUrl"`
}

// Collect runs each term and stops at the first error.
func (c Collector) Collect(ctx context.Context) ([]job.Job, error) {
	var jobs []job.Job
	for _, term := range c.Terms {
		page, err := c.Client.Get(ctx, "https://"+Host+"/job-search/term="+url.PathEscape(term))
		if err != nil {
			return jobs, fmt.Errorf("term %q: %w", term, err)
		}
		found, err := Parse(page)
		if err != nil {
			return jobs, fmt.Errorf("term %q: %w", term, err)
		}
		jobs = append(jobs, found...)
	}
	return jobs, nil
}

// Parse reads the jobs embedded in a search page.
func Parse(page []byte) ([]job.Job, error) {
	m := nextDataRe.FindSubmatch(page)
	if m == nil {
		return nil, fmt.Errorf("%w: no __NEXT_DATA__", ErrFormat)
	}
	var nd nextData
	if err := json.Unmarshal(m[1], &nd); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrFormat, err)
	}
	list := nd.Props.PageProps.InitialJobList
	if list == nil {
		return nil, fmt.Errorf("%w: no initialJobList", ErrFormat)
	}
	jobs := make([]job.Job, 0, len(list.Data))
	for _, g := range list.Data {
		if g.ID == 0 || g.JobURL == "" {
			return nil, fmt.Errorf("%w: job without id or url", ErrFormat)
		}
		jobs = append(jobs, job.Job{
			Source:      "gupy",
			SourceID:    strconv.FormatInt(g.ID, 10),
			URL:         originalURL(g.JobURL),
			Title:       strings.TrimSpace(g.Name),
			Company:     g.CareerPageName,
			Location:    location(g.City, g.State),
			Description: g.Description,
			WorkMode:    workMode(g.WorkplaceType),
			Contract:    contract(g.Type),
			PostedAt:    g.PublishedDate,
		})
	}
	return jobs, nil
}

// originalURL drops Gupy's tracking parameter so the link goes straight to
// the company's own career page.
func originalURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	q := u.Query()
	q.Del("jobBoardSource")
	u.RawQuery = q.Encode()
	return u.String()
}

func location(city, state string) string {
	switch {
	case city != "" && state != "":
		return city + ", " + state
	default:
		return city + state
	}
}

func workMode(s string) job.WorkMode {
	switch s {
	case "remote":
		return job.WorkModeRemote
	case "hybrid":
		return job.WorkModeHybrid
	case "on-site", "onsite":
		return job.WorkModeOnsite
	}
	return job.WorkModeUnknown
}

func contract(s string) job.Contract {
	switch s {
	case "vacancy_type_effective":
		return job.ContractCLT
	case "vacancy_legal_entity":
		return job.ContractPJ
	case "vacancy_type_internship":
		return job.ContractInternship
	case "vacancy_type_freelancer", "vacancy_type_autonomous":
		return job.ContractFreelance
	}
	return job.ContractUnknown
}
