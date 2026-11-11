// Package dbtest gives tests a Postgres of their own.
package dbtest

import (
	"context"
	"fmt"
	"math/rand/v2"
	"net/url"
	"os"
	"testing"

	"github.com/victor-novakoski/garimpo/internal/db"
)

// URLEnv points the tests at a Postgres they may fill and drop: a local
// one, or the service the CI starts. Without it the database tests are
// skipped, and the rules are still tested through the File store.
const URLEnv = "GARIMPO_TEST_DATABASE_URL"

// URL returns the connection string of a fresh, empty schema that goes
// away with the test, so tests never see each other's rows. It skips the
// test when URLEnv is not set.
func URL(t *testing.T) string {
	t.Helper()
	base := os.Getenv(URLEnv)
	if base == "" {
		t.Skipf("%s não definido: teste do Postgres pulado", URLEnv)
	}
	ctx := context.Background()
	admin, err := db.Open(ctx, base)
	if err != nil {
		t.Fatal(err)
	}
	schema := fmt.Sprintf("garimpo_test_%d", rand.Uint64())
	if _, err := admin.Exec(ctx, "create schema "+schema); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		admin.Exec(ctx, "drop schema "+schema+" cascade")
		admin.Close()
	})
	u, err := url.Parse(base)
	if err != nil {
		t.Fatal(err)
	}
	q := u.Query()
	q.Set("search_path", schema)
	u.RawQuery = q.Encode()
	return u.String()
}
