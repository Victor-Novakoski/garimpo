// Package httpx is the only way the collectors reach the network. It
// enforces the collection rules from docs/SOURCES.md: GET only, fixed
// hosts, a timeout, a response size limit, a pause between requests and a
// User-Agent that identifies the project.
package httpx

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"
)

// UserAgent identifies Garimpo to every source.
const UserAgent = "garimpo/0.1 (+https://github.com/victor-novakoski/garimpo)"

// Defaults used when Options leaves a field at zero.
const (
	DefaultTimeout  = 15 * time.Second
	DefaultMaxBytes = 5 << 20 // 5 MiB
	DefaultInterval = time.Second
)

// ErrHostNotAllowed is returned for a URL outside the client's host list.
var ErrHostNotAllowed = errors.New("host not allowed")

// ErrTooLarge is returned when a response passes the size limit.
var ErrTooLarge = errors.New("response too large")

// StatusError is returned for any non-2xx response.
type StatusError struct {
	URL  string
	Code int
}

func (e *StatusError) Error() string {
	return fmt.Sprintf("GET %s: status %d", e.URL, e.Code)
}

// Options configures a Client.
type Options struct {
	Hosts    []string      // hosts the client may call; required
	Timeout  time.Duration // per request
	MaxBytes int64         // largest body accepted
	Interval time.Duration // minimum pause between requests
	Header   http.Header   // extra headers sent on every request
	// Transport lets tests swap the network; nil uses http.DefaultTransport.
	Transport http.RoundTripper
}

// Client is a polite HTTP client. It is safe for concurrent use, but
// requests are serialized so the interval holds.
type Client struct {
	hosts    map[string]bool
	maxBytes int64
	interval time.Duration
	header   http.Header
	http     *http.Client

	mu   sync.Mutex
	last time.Time
	now  func() time.Time
	wait func(context.Context, time.Duration) error
}

// New builds a Client from opts.
func New(opts Options) *Client {
	c := &Client{
		hosts:    make(map[string]bool, len(opts.Hosts)),
		maxBytes: opts.MaxBytes,
		interval: opts.Interval,
		header:   opts.Header.Clone(),
		now:      time.Now,
		wait:     sleep,
	}
	for _, h := range opts.Hosts {
		c.hosts[h] = true
	}
	if c.maxBytes <= 0 {
		c.maxBytes = DefaultMaxBytes
	}
	if c.interval <= 0 {
		c.interval = DefaultInterval
	}
	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	c.http = &http.Client{
		Timeout:   timeout,
		Transport: opts.Transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 3 {
				return errors.New("too many redirects")
			}
			if !c.hosts[req.URL.Hostname()] {
				return fmt.Errorf("redirect to %s: %w", req.URL.Hostname(), ErrHostNotAllowed)
			}
			return nil
		},
	}
	return c
}

// Get fetches rawURL and returns the body. Any non-2xx status is a
// *StatusError.
func (c *Client) Get(ctx context.Context, rawURL string) ([]byte, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("parse url: %w", err)
	}
	if u.Scheme != "https" || !c.hosts[u.Hostname()] {
		return nil, fmt.Errorf("%s: %w", u.Host, ErrHostNotAllowed)
	}
	if err := c.pace(ctx); err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	for k, v := range c.header {
		req.Header[k] = v
	}
	req.Header.Set("User-Agent", UserAgent)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, c.maxBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}
	if int64(len(body)) > c.maxBytes {
		return nil, fmt.Errorf("GET %s: %w", u, ErrTooLarge)
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, &StatusError{URL: u.String(), Code: resp.StatusCode}
	}
	return body, nil
}

// pace blocks until the interval since the previous request has passed.
func (c *Client) pace(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.last.IsZero() {
		if d := c.interval - c.now().Sub(c.last); d > 0 {
			if err := c.wait(ctx, d); err != nil {
				return err
			}
		}
	}
	c.last = c.now()
	return nil
}

func sleep(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}
