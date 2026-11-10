package httpx

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

// roundTrip answers every request without touching the network.
type roundTrip func(*http.Request) (*http.Response, error)

func (f roundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func reply(code int, body string) roundTrip {
	return func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: code,
			Body:       io.NopCloser(strings.NewReader(body)),
			Header:     http.Header{},
		}, nil
	}
}

func TestGetSendsUserAgentAndHeaders(t *testing.T) {
	var got *http.Request
	c := New(Options{
		Hosts:  []string{"example.com"},
		Header: http.Header{"Accept": {"application/json"}},
		Transport: roundTrip(func(r *http.Request) (*http.Response, error) {
			got = r
			return reply(200, "ok")(r)
		}),
	})
	body, err := c.Get(context.Background(), "https://example.com/a")
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "ok" {
		t.Fatalf("body = %q", body)
	}
	if got.Method != http.MethodGet {
		t.Errorf("method = %s, want GET", got.Method)
	}
	if ua := got.Header.Get("User-Agent"); ua != UserAgent {
		t.Errorf("User-Agent = %q", ua)
	}
	if a := got.Header.Get("Accept"); a != "application/json" {
		t.Errorf("Accept = %q", a)
	}
}

func TestGetRejectsOtherHostsAndHTTP(t *testing.T) {
	c := New(Options{Hosts: []string{"example.com"}, Transport: reply(200, "")})
	for _, u := range []string{"https://evil.com/", "http://example.com/"} {
		if _, err := c.Get(context.Background(), u); !errors.Is(err, ErrHostNotAllowed) {
			t.Errorf("Get(%s) err = %v, want ErrHostNotAllowed", u, err)
		}
	}
}

func TestGetSizeLimit(t *testing.T) {
	c := New(Options{Hosts: []string{"example.com"}, MaxBytes: 4, Transport: reply(200, "12345")})
	if _, err := c.Get(context.Background(), "https://example.com/"); !errors.Is(err, ErrTooLarge) {
		t.Fatalf("err = %v, want ErrTooLarge", err)
	}
}

func TestGetStatusError(t *testing.T) {
	c := New(Options{Hosts: []string{"example.com"}, Transport: reply(404, "nope")})
	_, err := c.Get(context.Background(), "https://example.com/")
	var se *StatusError
	if !errors.As(err, &se) || se.Code != 404 {
		t.Fatalf("err = %v, want StatusError 404", err)
	}
}

func TestPaceWaitsBetweenRequests(t *testing.T) {
	c := New(Options{Hosts: []string{"example.com"}, Interval: time.Second, Transport: reply(200, "")})
	clock := time.Unix(0, 0)
	c.now = func() time.Time { return clock }
	var waited []time.Duration
	c.wait = func(_ context.Context, d time.Duration) error {
		waited = append(waited, d)
		clock = clock.Add(d)
		return nil
	}
	for i := 0; i < 3; i++ {
		if _, err := c.Get(context.Background(), "https://example.com/"); err != nil {
			t.Fatal(err)
		}
		clock = clock.Add(300 * time.Millisecond)
	}
	if len(waited) != 2 || waited[0] != 700*time.Millisecond {
		t.Fatalf("waited = %v, want two waits of 700ms", waited)
	}
}
