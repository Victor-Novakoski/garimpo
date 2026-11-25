// Package profile loads what the person is looking for: accepted work
// modes, contracts and levels, what to avoid and how much each skill is
// worth. The real profile stays out of the repository (profile.yaml is
// git-ignored); profile.example.yaml shows the format.
package profile

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"gopkg.in/yaml.v3"

	"github.com/victor-novakoski/garimpo/internal/job"
)

// Skill is one technology and the points a posting earns for asking it.
type Skill struct {
	Name    string   `yaml:"name"`
	Aliases []string `yaml:"aliases"`
	Points  int      `yaml:"points"`
}

// Profile is the parsed profile file.
type Profile struct {
	Name      string        `yaml:"name"`
	Seniority job.Seniority `yaml:"seniority"`
	Accept    Accept        `yaml:"accept"`
	Prefer    Prefer        `yaml:"prefer"`
	Avoid     Avoid         `yaml:"avoid"`
	Skills    []Skill       `yaml:"skills"`
	// About is free text for the AI step: experience, years, projects,
	// what the person is looking for. No rule reads it.
	About string `yaml:"about"`
	// AboutFiles are local text files (absolute paths or starting with ~)
	// whose text joins About for the AI step, so notes kept elsewhere stay
	// current without copying. No rule reads them; Load fills AboutTexts.
	AboutFiles []string `yaml:"about_files"`
	// AboutTexts is what Load read from AboutFiles, in the same order.
	AboutTexts []AboutText `yaml:"-"`
}

// AboutText is the text of one of the about files.
type AboutText struct {
	Name string // base name of the file, to tell the texts apart
	Text string
}

// Limits for the about files, in bytes. They keep the prompt, which every
// assessed job pays for, from growing without anyone noticing: about 5000
// tokens per file and 10000 in all.
const (
	MaxAboutFileBytes  = 20_000
	MaxAboutTotalBytes = 40_000
)

// Accept lists what may pass the hard filter. A job whose value the source
// did not give always passes; the score treats unknown as neutral.
type Accept struct {
	WorkModes   []job.WorkMode  `yaml:"work_modes"`
	Contracts   []job.Contract  `yaml:"contracts"`
	Seniorities []job.Seniority `yaml:"seniorities"`
}

// Prefer lists what earns full points among the accepted values.
type Prefer struct {
	WorkModes []job.WorkMode `yaml:"work_modes"`
	Contracts []job.Contract `yaml:"contracts"`
}

// Avoid drops a job when any word shows up in its title, company or tags,
// or its location mentions any of the locations.
type Avoid struct {
	Words     []string `yaml:"words"`
	Companies []string `yaml:"companies"`
	Locations []string `yaml:"locations"`
}

// MaxSkillPoints caps the skill part of the score.
const MaxSkillPoints = 40

// Load reads and validates a profile file, and reads its about files.
func Load(path string) (Profile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Profile{}, err
	}
	p, err := Parse(data)
	if err != nil {
		return p, err
	}
	texts, err := readAboutFiles(p.AboutFiles)
	if err != nil {
		return Profile{}, fmt.Errorf("profile: %w", err)
	}
	p.AboutTexts = texts
	return p, nil
}

// readAboutFiles reads each file whole. Errors name the path and never the
// content, which is personal.
func readAboutFiles(paths []string) ([]AboutText, error) {
	var texts []AboutText
	total := 0
	for _, raw := range paths {
		path, err := expandHome(raw)
		if err != nil {
			return nil, err
		}
		info, err := os.Stat(path)
		if err != nil {
			return nil, fmt.Errorf("about_files: %s: %w", raw, unwrapPath(err))
		}
		if !info.Mode().IsRegular() {
			return nil, fmt.Errorf("about_files: %s: not a regular file", raw)
		}
		if info.Size() > MaxAboutFileBytes {
			return nil, fmt.Errorf("about_files: %s has %d bytes, the limit per file is %d", raw, info.Size(), MaxAboutFileBytes)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("about_files: %s: %w", raw, unwrapPath(err))
		}
		if len(data) > MaxAboutFileBytes {
			return nil, fmt.Errorf("about_files: %s has %d bytes, the limit per file is %d", raw, len(data), MaxAboutFileBytes)
		}
		if !utf8.Valid(data) {
			return nil, fmt.Errorf("about_files: %s is not UTF-8 text", raw)
		}
		total += len(data)
		if total > MaxAboutTotalBytes {
			return nil, fmt.Errorf("about_files: together the files pass the limit of %d bytes (at %s)", MaxAboutTotalBytes, raw)
		}
		texts = append(texts, AboutText{Name: filepath.Base(path), Text: string(data)})
	}
	return texts, nil
}

// expandHome turns a leading ~ into the home directory. Relative paths are
// refused: they would depend on the folder the command runs from.
func expandHome(path string) (string, error) {
	if path == "~" || strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("about_files: %s: %w", path, err)
		}
		path = filepath.Join(home, strings.TrimPrefix(path, "~"))
	}
	if !filepath.IsAbs(path) {
		return "", fmt.Errorf("about_files: %q must be an absolute path or start with ~/", path)
	}
	return filepath.Clean(path), nil
}

// unwrapPath drops the *PathError wrapper, whose message repeats the
// expanded path; the error already names the path as written.
func unwrapPath(err error) error {
	var pe *os.PathError
	if errors.As(err, &pe) {
		return pe.Err
	}
	return err
}

// Parse reads and validates profile YAML. Unknown fields are an error, so
// a typo never silently turns a rule off.
func Parse(data []byte) (Profile, error) {
	var p Profile
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&p); err != nil {
		return Profile{}, fmt.Errorf("profile: %w", err)
	}
	return p, p.validate()
}

func (p Profile) validate() error {
	var errs []error
	if len(p.Accept.WorkModes) == 0 {
		errs = append(errs, errors.New("accept.work_modes is empty"))
	}
	if len(p.Accept.Contracts) == 0 {
		errs = append(errs, errors.New("accept.contracts is empty"))
	}
	if len(p.Accept.Seniorities) == 0 {
		errs = append(errs, errors.New("accept.seniorities is empty"))
	}
	for _, w := range p.Accept.WorkModes {
		switch w {
		case job.WorkModeRemote, job.WorkModeHybrid, job.WorkModeOnsite:
		default:
			errs = append(errs, fmt.Errorf("unknown work mode %q", w))
		}
	}
	for _, c := range p.Accept.Contracts {
		switch c {
		case job.ContractPJ, job.ContractCLT, job.ContractFreelance, job.ContractInternship:
		default:
			errs = append(errs, fmt.Errorf("unknown contract %q", c))
		}
	}
	for _, s := range append([]job.Seniority{p.Seniority}, p.Accept.Seniorities...) {
		switch s {
		case job.SeniorityJunior, job.SeniorityMid, job.SenioritySenior, job.SeniorityLead:
		default:
			errs = append(errs, fmt.Errorf("unknown seniority %q", s))
		}
	}
	for _, s := range p.Skills {
		if s.Name == "" || s.Points <= 0 {
			errs = append(errs, fmt.Errorf("skill %q needs a name and positive points", s.Name))
		}
	}
	if err := errors.Join(errs...); err != nil {
		return fmt.Errorf("profile: %w", err)
	}
	return nil
}
