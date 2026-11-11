// Package lever collects postings from companies that publish their jobs
// on Lever, through the public Postings API.
package lever

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/victor-novakoski/garimpo/internal/collector/greenhouse"
	"github.com/victor-novakoski/garimpo/internal/httpx"
	"github.com/victor-novakoski/garimpo/internal/job"
)

// Host is the only host this collector calls.
const Host = "api.lever.co"

// Site is one company's job site.
type Site struct {
	Slug    string // the name in jobs.lever.co/<slug>
	Company string // display name
}

// Sites are companies with open jobs in Brazil on Lever (checked on
// 11/11/2026).
var Sites = []Site{
	{Slug: "neon", Company: "Neon"},
	{Slug: "starkbank", Company: "Stark Bank"},
	{Slug: "swile", Company: "Swile"},
}

// Getter is the part of httpx.Client the collector needs.
type Getter interface {
	Get(ctx context.Context, rawURL string) ([]byte, error)
}

// NewClient builds the HTTP client for the Lever API.
func NewClient() *httpx.Client {
	return httpx.New(httpx.Options{Hosts: []string{Host}})
}

// Collector reads every open job on one site.
type Collector struct {
	Site   Site
	Client Getter
}

// Name is "lever:<slug>".
func (c Collector) Name() string { return "lever:" + c.Site.Slug }

type posting struct {
	ID               string `json:"id"`
	Text             string `json:"text"`
	HostedURL        string `json:"hostedUrl"`
	CreatedAt        int64  `json:"createdAt"` // milliseconds
	Country          string `json:"country"`
	WorkplaceType    string `json:"workplaceType"`
	DescriptionPlain string `json:"descriptionPlain"`
	AdditionalPlain  string `json:"additionalPlain"`
	Categories       struct {
		Commitment string `json:"commitment"`
		Department string `json:"department"`
		Location   string `json:"location"`
		Team       string `json:"team"`
	} `json:"categories"`
	Lists []struct {
		Text    string `json:"text"`
		Content string `json:"content"`
	} `json:"lists"`
}

// Collect fetches the site in one request. Postings that Lever marks with
// another country are left out: these sites also hire in France and Mexico.
func (c Collector) Collect(ctx context.Context) ([]job.Job, error) {
	u := fmt.Sprintf("https://%s/v0/postings/%s?mode=json", Host, c.Site.Slug)
	body, err := c.Client.Get(ctx, u)
	if err != nil {
		return nil, err
	}
	var postings []posting
	if err := json.Unmarshal(body, &postings); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}
	jobs := make([]job.Job, 0, len(postings))
	for _, p := range postings {
		if p.Country != "" && !strings.EqualFold(p.Country, "BR") {
			continue
		}
		var tags []string
		for _, t := range []string{p.Categories.Department, p.Categories.Team, p.Categories.Commitment} {
			if t = strings.TrimSpace(t); t != "" {
				tags = append(tags, t)
			}
		}
		j := job.Job{
			Source:      c.Name(),
			SourceID:    p.ID,
			URL:         p.HostedURL,
			Title:       strings.TrimSpace(p.Text),
			Company:     c.Site.Company,
			Location:    p.Categories.Location,
			Description: description(p),
			Tags:        tags,
			WorkMode:    workMode(p.WorkplaceType),
		}
		if p.CreatedAt > 0 {
			j.PostedAt = time.UnixMilli(p.CreatedAt).UTC()
		}
		jobs = append(jobs, j)
	}
	return jobs, nil
}

// description joins the parts Lever splits a posting into. The lists hold
// the requirements, so they cannot be left out.
func description(p posting) string {
	parts := []string{strings.TrimSpace(p.DescriptionPlain)}
	for _, l := range p.Lists {
		parts = append(parts, strings.TrimSpace(l.Text)+"\n"+greenhouse.PlainText(l.Content))
	}
	parts = append(parts, strings.TrimSpace(p.AdditionalPlain))
	var kept []string
	for _, s := range parts {
		if s = strings.TrimSpace(s); s != "" {
			kept = append(kept, s)
		}
	}
	return strings.Join(kept, "\n\n")
}

// workMode trusts Lever's own field; anything else is left for the
// classifier to work out from the text.
func workMode(s string) job.WorkMode {
	switch strings.ToLower(s) {
	case "remote":
		return job.WorkModeRemote
	case "hybrid":
		return job.WorkModeHybrid
	case "onsite", "on-site":
		return job.WorkModeOnsite
	}
	return job.WorkModeUnknown
}
