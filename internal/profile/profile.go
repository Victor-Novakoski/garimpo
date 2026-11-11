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
}

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

// Load reads and validates a profile file.
func Load(path string) (Profile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Profile{}, err
	}
	return Parse(data)
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
