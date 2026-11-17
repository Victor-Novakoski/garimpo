// Package greenhouse collects postings from companies that publish their
// jobs on Greenhouse, through the public Job Board API.
package greenhouse

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/victor-novakoski/garimpo/internal/httpx"
	"github.com/victor-novakoski/garimpo/internal/job"
	"github.com/victor-novakoski/garimpo/internal/normalize"
)

// Host is the only host this collector calls.
const Host = "boards-api.greenhouse.io"

// Board is one company's job board.
type Board struct {
	Token   string // the board token in boards.greenhouse.io/<token>
	Company string // display name
}

// Boards are companies with open jobs in Brazil on Greenhouse (checked on
// 11/11/2026). Many of their jobs are not in tech or not in Brazil; the
// filter handles that.
var Boards = []Board{
	{Token: "stone", Company: "Stone"},
	{Token: "quintoandar", Company: "QuintoAndar"},
	{Token: "wildlifestudios", Company: "Wildlife Studios"},
	{Token: "arcoeducacao", Company: "Arco Educação"},
	{Token: "btgpactual", Company: "BTG Pactual"},
	{Token: "c6bank", Company: "C6 Bank"},
	{Token: "clara", Company: "Clara"},
	{Token: "ebanx", Company: "EBANX"},
	{Token: "getnet", Company: "Getnet"},
	{Token: "gympass", Company: "Wellhub"},
	{Token: "incognia", Company: "Incognia"},
	{Token: "inter", Company: "Inter"},
	{Token: "jusbrasil", Company: "Jusbrasil"},
	{Token: "rdstation", Company: "RD Station"},
	{Token: "vtex", Company: "VTEX"},
	{Token: "xpinc", Company: "XP Inc."},
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
		Metadata []metadata `json:"metadata"`
	} `json:"jobs"`
}

// metadata is one custom field the company added to its board. The value
// is a string, a list, a number, a boolean or null, depending on the field.
type metadata struct {
	Name  string          `json:"name"`
	Value json.RawMessage `json:"value"`
}

// workModeFields are the names companies give to the custom field that
// holds the work mode, folded. Wildlife calls it "Work Model".
var workModeFields = []string{"work model", "work mode", "workplace", "modelo de trabalho"}

// workMode reads the work mode from the company's own field, when the board
// has one. Most boards do not; those are left for the normalizer, which
// reads the title, the location and the description.
func workMode(fields []metadata) job.WorkMode {
	for _, f := range fields {
		if !slices.Contains(workModeFields, normalize.Fold(f.Name)) {
			continue
		}
		var value string
		if err := json.Unmarshal(f.Value, &value); err != nil {
			continue // null or not a text field
		}
		if m := normalize.WorkMode(value); m != job.WorkModeUnknown {
			return m
		}
	}
	return job.WorkModeUnknown
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
			WorkMode:    workMode(gj.Metadata),
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
