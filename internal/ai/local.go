package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/victor-novakoski/garimpo/internal/job"
	"github.com/victor-novakoski/garimpo/internal/profile"
)

// DefaultLocalModel is the Claude Code alias used when none is given: the
// fastest model, which also spends the least of the plan's usage.
const DefaultLocalModel = "haiku"

// Local scores with the Claude Code CLI ("claude -p") already signed in on
// the machine, so it uses the person's Claude plan instead of an API key.
//
// A posting is text from the internet, so the CLI runs with every tool
// off, no MCP server, no project files and no saved session, in an empty
// working directory. All a hostile posting can reach is its own class.
type Local struct {
	system  string
	model   string
	command string
	timeout time.Duration
	// run executes the CLI; tests replace it.
	run func(ctx context.Context, name string, args []string, stdin string) ([]byte, error)
}

// NewLocal builds the local scorer. An empty model means DefaultLocalModel.
func NewLocal(p profile.Profile, model string) *Local {
	if model == "" {
		model = DefaultLocalModel
	}
	return &Local{
		system:  System(p),
		model:   model,
		command: "claude",
		timeout: 3 * time.Minute,
		run:     runCommand,
	}
}

func (l *Local) Name() string { return "local:" + l.model }

func (l *Local) args() []string {
	schema, _ := json.Marshal(Schema())
	return []string{
		"-p",
		"--output-format", "json",
		"--json-schema", string(schema),
		"--system-prompt", l.system,
		"--model", l.model,
		"--tools", "", // no tool at all: nothing to read, write or run
		"--strict-mcp-config", // and no MCP server from the user's config
		"--safe-mode",         // no CLAUDE.md, hooks, skills or plugins
		"--disable-slash-commands",
		"--no-session-persistence",
	}
}

// envelope is what "claude -p --output-format json" prints.
type envelope struct {
	IsError          bool            `json:"is_error"`
	Subtype          string          `json:"subtype"`
	Result           string          `json:"result"`
	StructuredOutput json.RawMessage `json:"structured_output"`
}

func (l *Local) Assess(ctx context.Context, j job.Job) (Assessment, error) {
	ctx, cancel := context.WithTimeout(ctx, l.timeout)
	defer cancel()
	out, err := l.run(ctx, l.command, l.args(), User(j))
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return Assessment{}, fmt.Errorf("%w: comando %q não encontrado (instale o Claude Code e faça login)", ErrUnavailable, l.command)
		}
		if strings.Contains(string(out), "unknown option") {
			return Assessment{}, fmt.Errorf("%w: este Claude Code não conhece uma das opções usadas; atualize com \"claude update\": %s", ErrUnavailable, firstLine(out))
		}
		return Assessment{}, fmt.Errorf("claude -p: %w: %s", err, firstLine(out))
	}
	var env envelope
	if err := json.Unmarshal(out, &env); err != nil {
		return Assessment{}, fmt.Errorf("claude -p: saída inesperada: %s", firstLine(out))
	}
	if env.IsError {
		msg := cut(clean(env.Result), 200)
		if strings.Contains(strings.ToLower(msg), "login") || strings.Contains(strings.ToLower(msg), "api key") {
			return Assessment{}, fmt.Errorf("%w: %s", ErrUnavailable, msg)
		}
		return Assessment{}, fmt.Errorf("claude -p: %s: %s", env.Subtype, msg)
	}
	if len(env.StructuredOutput) > 0 && string(env.StructuredOutput) != "null" {
		return Parse(env.StructuredOutput)
	}
	return Parse([]byte(env.Result))
}

// runCommand runs the CLI in an empty directory with the prompt on stdin.
// ANTHROPIC_API_KEY is removed from its environment: with it set the CLI
// would bill the API instead of using the plan, which is the whole point
// of the local mode.
func runCommand(ctx context.Context, name string, args []string, stdin string) ([]byte, error) {
	dir, err := os.MkdirTemp("", "garimpo-ai-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.Stdin = strings.NewReader(stdin)
	for _, kv := range os.Environ() {
		if !strings.HasPrefix(kv, "ANTHROPIC_API_KEY=") && !strings.HasPrefix(kv, "ANTHROPIC_AUTH_TOKEN=") {
			cmd.Env = append(cmd.Env, kv)
		}
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		if stdout.Len() > 0 { // the CLI reports its own errors as JSON on stdout
			return stdout.Bytes(), nil
		}
		return stderr.Bytes(), err
	}
	return stdout.Bytes(), nil
}

func firstLine(b []byte) string {
	s := strings.TrimSpace(string(b))
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	return cut(s, 200)
}
