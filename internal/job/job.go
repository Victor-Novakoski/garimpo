// Package job defines the common job posting format every collector produces.
package job

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"
)

// WorkMode says where the work happens.
type WorkMode string

const (
	WorkModeUnknown WorkMode = ""
	WorkModeRemote  WorkMode = "remote"
	WorkModeHybrid  WorkMode = "hybrid"
	WorkModeOnsite  WorkMode = "onsite"
)

// Contract is the hiring model.
type Contract string

const (
	ContractUnknown    Contract = ""
	ContractPJ         Contract = "pj"
	ContractCLT        Contract = "clt"
	ContractFreelance  Contract = "freelance"
	ContractInternship Contract = "internship"
)

// Seniority is the level the posting asks for.
type Seniority string

const (
	SeniorityUnknown Seniority = ""
	SeniorityJunior  Seniority = "junior"
	SeniorityMid     Seniority = "mid"
	SenioritySenior  Seniority = "senior"
)

// Job is one posting, normalized from any source.
type Job struct {
	Source      string // collector name, e.g. "backend-br"
	SourceID    string // id inside the source
	URL         string // where a person applies
	Title       string
	Company     string
	Location    string
	Description string
	Tags        []string
	WorkMode    WorkMode
	Contract    Contract
	Seniority   Seniority
	PostedAt    time.Time
}

// Key identifies a job inside its source.
func (j Job) Key() string {
	return j.Source + ":" + j.SourceID
}

// Fingerprint identifies the same opening across sources, using company
// and title normalized. Empty when either is missing.
func (j Job) Fingerprint() string {
	company := normalize(j.Company)
	title := normalize(j.Title)
	if company == "" || title == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(company + "|" + title))
	return hex.EncodeToString(sum[:8])
}

func normalize(s string) string {
	return strings.Join(strings.Fields(strings.ToLower(s)), " ")
}
