package job

import "testing"

func TestKey(t *testing.T) {
	j := Job{Source: "gupy", SourceID: "123"}
	if got := j.Key(); got != "gupy:123" {
		t.Fatalf("Key() = %q, want %q", got, "gupy:123")
	}
}

func TestFingerprint(t *testing.T) {
	tests := []struct {
		name string
		a, b Job
		same bool
	}{
		{
			name: "case and spaces do not matter",
			a:    Job{Company: "Acme", Title: "Dev Go Pleno"},
			b:    Job{Company: "  ACME ", Title: "dev   go pleno"},
			same: true,
		},
		{
			name: "different title is a different opening",
			a:    Job{Company: "Acme", Title: "Dev Go Pleno"},
			b:    Job{Company: "Acme", Title: "Dev React Pleno"},
			same: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.a.Fingerprint() == tt.b.Fingerprint(); got != tt.same {
				t.Fatalf("same fingerprint = %v, want %v", got, tt.same)
			}
		})
	}
}

func TestFingerprintEmptyWhenMissingFields(t *testing.T) {
	if got := (Job{Title: "Dev Go"}).Fingerprint(); got != "" {
		t.Fatalf("Fingerprint() = %q, want empty", got)
	}
}
