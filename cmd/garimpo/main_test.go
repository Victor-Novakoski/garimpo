package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRun(t *testing.T) {
	tests := []struct {
		args   []string
		code   int
		stdout string
		stderr string
	}{
		{args: nil, code: 0, stdout: "Uso:"},
		{args: []string{"version"}, code: 0, stdout: "dev"},
		{args: []string{"nope"}, code: 2, stderr: "comando desconhecido"},
	}
	for _, tt := range tests {
		var out, errOut bytes.Buffer
		code := run(tt.args, &out, &errOut)
		if code != tt.code {
			t.Errorf("run(%v) code = %d, want %d", tt.args, code, tt.code)
		}
		if !strings.Contains(out.String(), tt.stdout) {
			t.Errorf("run(%v) stdout = %q, want it to contain %q", tt.args, out.String(), tt.stdout)
		}
		if !strings.Contains(errOut.String(), tt.stderr) {
			t.Errorf("run(%v) stderr = %q, want it to contain %q", tt.args, errOut.String(), tt.stderr)
		}
	}
}
