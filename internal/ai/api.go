package ai

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/anthropics/anthropic-sdk-go/v2"
	"github.com/anthropics/anthropic-sdk-go/v2/option"

	"github.com/victor-novakoski/garimpo/internal/job"
	"github.com/victor-novakoski/garimpo/internal/profile"
)

// DefaultAPIModel is the cheapest current Claude model, enough for picking
// one of five classes from a posting.
const DefaultAPIModel = "claude-haiku-5-5"

// apiHost is the only host the API scorer may talk to.
const apiHost = "api.anthropic.com"

// API scores with the Claude API through the official SDK. Credentials
// come from the environment (ANTHROPIC_API_KEY).
type API struct {
	client anthropic.Client
	system string
	letter string // the system prompt of the cover letter
	model  string
}

// NewAPI builds the API scorer. An empty model means DefaultAPIModel. Extra
// options are for tests (a fake server).
func NewAPI(p profile.Profile, model string, opts ...option.RequestOption) *API {
	if model == "" {
		model = DefaultAPIModel
	}
	base := []option.RequestOption{
		option.WithHTTPClient(&http.Client{
			Timeout:   2 * time.Minute,
			Transport: hostOnly{host: apiHost, base: http.DefaultTransport},
		}),
	}
	return &API{
		client: anthropic.NewClient(append(base, opts...)...),
		system: System(p),
		letter: LetterSystem(p),
		model:  model,
	}
}

func (a *API) Name() string { return "api:" + a.model }

func (a *API) Assess(ctx context.Context, j job.Job) (Assessment, error) {
	out, err := a.ask(ctx, a.system, Schema(), User(j))
	if err != nil {
		return Assessment{}, err
	}
	return Parse(out)
}

// Letter drafts the cover letter with the same model and limits.
func (a *API) Letter(ctx context.Context, j job.Job, n Notes) (string, error) {
	out, err := a.ask(ctx, a.letter, LetterSchema(), LetterUser(j, n))
	if err != nil {
		return "", err
	}
	return ParseLetter(out)
}

// ask sends one message and returns the JSON the model answered, still to
// be validated by the caller.
func (a *API) ask(ctx context.Context, system string, schema map[string]any, user string) ([]byte, error) {
	resp, err := a.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(a.model),
		MaxTokens: 8000,
		// The system prompt is the same for every job of a run.
		System: []anthropic.TextBlockParam{{
			Text:         system,
			CacheControl: anthropic.NewCacheControlEphemeralParam(),
		}},
		OutputConfig: anthropic.OutputConfigParam{
			Effort: anthropic.OutputConfigEffortMedium,
			Format: anthropic.JSONOutputFormatParam{Schema: schema},
		},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(user)),
		},
	})
	if err != nil {
		var apiErr *anthropic.Error
		if errors.As(err, &apiErr) {
			switch apiErr.StatusCode {
			case http.StatusUnauthorized, http.StatusForbidden:
				return nil, fmt.Errorf("%w: a API recusou a credencial (confira ANTHROPIC_API_KEY)", ErrUnavailable)
			case http.StatusNotFound:
				return nil, fmt.Errorf("%w: modelo %q não encontrado", ErrUnavailable, a.model)
			}
			return nil, fmt.Errorf("API do Claude: status %d", apiErr.StatusCode)
		}
		return nil, fmt.Errorf("API do Claude: %w", err)
	}
	switch resp.StopReason {
	case anthropic.StopReasonRefusal:
		return nil, errors.New("a IA se recusou a responder sobre esta vaga")
	case anthropic.StopReasonMaxTokens:
		return nil, errors.New("resposta da IA cortada por tamanho")
	}
	for _, block := range resp.Content {
		if text, ok := block.AsAny().(anthropic.TextBlock); ok {
			return []byte(text.Text), nil
		}
	}
	return nil, errors.New("resposta da IA sem texto")
}

// hostOnly refuses any request that is not HTTPS to the one allowed host,
// the same rule the collectors follow.
type hostOnly struct {
	host string
	base http.RoundTripper
}

func (t hostOnly) RoundTrip(r *http.Request) (*http.Response, error) {
	if r.URL.Scheme != "https" || r.URL.Hostname() != t.host {
		return nil, fmt.Errorf("ai: host não permitido: %s://%s", r.URL.Scheme, r.URL.Host)
	}
	return t.base.RoundTrip(r)
}
