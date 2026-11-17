package ai

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/victor-novakoski/garimpo/internal/job"
)

// cacheTTL is how long an answer is kept. Postings rarely stay open longer.
const cacheTTL = 60 * 24 * time.Hour

type cacheEntry struct {
	Assessment Assessment `json:"assessment"`
	At         time.Time  `json:"at"`
}

// Cache keeps answers on disk, so a posting is assessed once and the daily
// run only pays for what is new. The key covers the scorer, the prompt
// (which includes the profile) and the posting: change any of them and the
// job is assessed again.
type Cache struct {
	path string
	now  func() time.Time

	mu      sync.Mutex
	entries map[string]cacheEntry
	dirty   bool
}

// DefaultCachePath is the cache file in the user's cache directory.
func DefaultCachePath() (string, error) {
	dir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "garimpo", "ai.json"), nil
}

// OpenCache loads the cache file. A missing or unreadable file starts an
// empty cache: losing it only costs assessing again.
func OpenCache(path string) *Cache {
	c := &Cache{path: path, now: time.Now, entries: map[string]cacheEntry{}}
	if data, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(data, &c.entries)
		if c.entries == nil {
			c.entries = map[string]cacheEntry{}
		}
	}
	return c
}

// Save writes the cache back when something changed, dropping old entries.
func (c *Cache) Save() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.dirty {
		return nil
	}
	for k, e := range c.entries {
		if c.now().Sub(e.At) > cacheTTL {
			delete(c.entries, k)
		}
	}
	data, err := json.Marshal(c.entries)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(c.path), 0o700); err != nil {
		return err
	}
	// The answers quote the profile, so the file is private to the user.
	tmp := c.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmp, c.path); err != nil {
		return err
	}
	c.dirty = false
	return nil
}

// Wrap returns a Scorer that answers from the cache when it can.
func (c *Cache) Wrap(s Scorer, system string) Scorer {
	return &cached{Scorer: s, cache: c, system: system}
}

type cached struct {
	Scorer
	cache  *Cache
	system string
}

func (s *cached) Assess(ctx context.Context, j job.Job) (Assessment, error) {
	key := cacheKey(s.Name(), s.system, User(j))
	c := s.cache
	c.mu.Lock()
	e, ok := c.entries[key]
	c.mu.Unlock()
	if ok && c.now().Sub(e.At) <= cacheTTL {
		return e.Assessment, nil
	}
	a, err := s.Scorer.Assess(ctx, j)
	if err != nil {
		return a, err // failures are not cached: the next run tries again
	}
	c.mu.Lock()
	c.entries[key] = cacheEntry{Assessment: a, At: c.now()}
	c.dirty = true
	c.mu.Unlock()
	return a, nil
}

func cacheKey(parts ...string) string {
	h := sha256.New()
	h.Write([]byte(promptVersion))
	for _, p := range parts {
		h.Write([]byte{0})
		h.Write([]byte(p))
	}
	return hex.EncodeToString(h.Sum(nil))
}
