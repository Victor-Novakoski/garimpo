// Package greenhouse collects postings from companies that publish their
// jobs on Greenhouse, through the public Job Board API.
package greenhouse

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/victor-novakoski/garimpo/internal/httpx"
	"github.com/victor-novakoski/garimpo/internal/job"
)

// Host is the only host this collector calls.
const Host = "boards-api.greenhouse.io"

// Board is one company's job board.
type Board struct {
	Token   string // the board token in boards.greenhouse.io/<token>
	Company string // display name
}

// Boards are Brazilian companies with an active Greenhouse board
// (checked on 10/11/2026). Many of their jobs are not in tech; the filter
// in stage 2 handles that.
var Boards = []Board{
	{Token: "stone", Company: "Stone"},
	{Token: "quintoandar", Company: "QuintoAndar"},
	{Token: "wildlifestudios", Company: "Wildlife Studios"},
}

// Getter is the part of httpx.Client the collector needs.
type Getter interface {
	Get(ctx context.Context, rawURL string) ([]byte, error)
}

// NewClient builds the HTTP client for the Greenhouse API.
func NewClient() *httpx.Client {
	return httpx.New(httpx.Options{Hosts: []string{Host}})
}

// Collector reads every open job on one board.
type Collector struct {
	Board  Board
	Client Getter
}

// Name is "greenhouse:<token>".
func (c Collector) Name() string { return "greenhouse:" + c.Board.Token }

type response struct {
	Jobs []struct {
		ID          int64     `json:"id"`
		Title       string    `json:"title"`
		AbsoluteURL string    `json:"absolute_url"`
		UpdatedAt   time.Time `json:"updated_at"`
		Content     string    `json:"content"`
		Location    struct {
			Name string `json:"name"`
		} `json:"location"`
		Departments []struct {
			Name string `json:"name"`
		} `json:"departments"`
	} `json:"jobs"`
}

// Collect fetches the board in one request.
func (c Collector) Collect(ctx context.Context) ([]job.Job, error) {
	u := fmt.Sprintf("https://%s/v1/boards/%s/jobs?content=true", Host, c.Board.Token)
	body, err := c.Client.Get(ctx, u)
	if err != nil {
		return nil, err
	}
	var r response
	if err := json.Unmarshal(body, &r); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}
	jobs := make([]job.Job, 0, len(r.Jobs))
	for _, gj := range r.Jobs {
		var tags []string
		for _, d := range gj.Departments {
			tags = append(tags, d.Name)
		}
		jobs = append(jobs, job.Job{
			Source:      c.Name(),
			SourceID:    strconv.FormatInt(gj.ID, 10),
			URL:         gj.AbsoluteURL,
			Title:       strings.TrimSpace(gj.Title),
			Company:     c.Board.Company,
			Location:    gj.Location.Name,
			Description: PlainText(gj.Content),
			Tags:        tags,
			PostedAt:    gj.UpdatedAt,
		})
	}
	return jobs, nil
}

var (
	tagRe    = regexp.MustCompile(`<[^>]*>`)
	blankRe  = regexp.MustCompile(`[ \t]+`)
	breaksRe = regexp.MustCompile(`\n{3,}`)
)

// PlainText turns Greenhouse content (HTML escaped as entities) into text.
func PlainText(s string) string {
	s = html.UnescapeString(s) // the API escapes the HTML once
	s = strings.NewReplacer("<br>", "\n", "<br/>", "\n", "<br />", "\n", "</p>", "\n\n", "</li>", "\n", "<li>", "- ").Replace(s)
	s = tagRe.ReplaceAllString(s, "")
	s = html.UnescapeString(s) // entities inside the HTML itself
	s = blankRe.ReplaceAllString(s, " ")
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		lines[i] = strings.TrimSpace(l)
	}
	return strings.TrimSpace(breaksRe.ReplaceAllString(strings.Join(lines, "\n"), "\n\n"))
}
