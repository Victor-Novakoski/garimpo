package job

import "testing"

func TestKey(t *testing.T) {
	j := Job{Source: "gupy", SourceID: "123"}
	if got := j.Key(); got != "gupy:123" {
		t.Fatalf("Key() = %q, want %q", got, "gupy:123")
	}
}
