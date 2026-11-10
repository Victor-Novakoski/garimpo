package score

import (
	"strings"
	"testing"

	"github.com/victor-novakoski/garimpo/internal/job"
	"github.com/victor-novakoski/garimpo/internal/profile"
)

func example(t *testing.T) profile.Profile {
	t.Helper()
	p, err := profile.Load("../../profile.example.yaml")
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestFilter(t *testing.T) {
	p := example(t)
	tests := []struct {
		name string
		job  job.Job
		want string // substring of the reason; "" means it passes
	}{
		{"passes", job.Job{Title: "Dev Go", WorkMode: job.WorkModeRemote}, ""},
		{"unknown values pass", job.Job{Title: "Dev Go"}, ""},
		{"hybrid dropped", job.Job{Title: "Dev Go", WorkMode: job.WorkModeHybrid}, "modelo hybrid"},
		{"internship dropped", job.Job{Title: "Estágio Go", Contract: job.ContractInternship}, "contrato internship"},
		{"avoided word", job.Job{Title: "Dev Go - Logística"}, "palavra evitada"},
		{"avoided location", job.Job{Title: "Software Engineer", Location: "Ciudad de México, Mexico"}, "local evitado"},
	}
	for _, tt := range tests {
		got := Filter(p, tt.job)
		if (tt.want == "") != (got == "") || !strings.Contains(got, tt.want) {
			t.Errorf("%s: Filter = %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestScore(t *testing.T) {
	p := example(t)
	best := job.Job{
		Title:     "Dev Full Stack Pleno - Golang, React, TypeScript, PostgreSQL",
		Tags:      []string{"Docker"},
		WorkMode:  job.WorkModeRemote,
		Contract:  job.ContractPJ,
		Seniority: job.SeniorityMid,
	}
	r := Score(p, best)
	// Go 15 + React 12 + TypeScript 6 + PostgreSQL 5 + Docker 2 = 40, then 10+10+10.
	if r.Points != Max {
		t.Fatalf("Points = %d, want %d; reasons %v", r.Points, Max, r.Reasons)
	}

	weak := job.Job{Title: "Dev Java Sênior", Description: "Docker", Contract: job.ContractCLT, Seniority: job.SenioritySenior}
	r = Score(p, weak)
	// Docker in the description 1 + unknown mode 2 + clt 4 + senior 6 = 13
	if r.Points != 13 {
		t.Fatalf("Points = %d, want 13; reasons %v", r.Points, r.Reasons)
	}

	none := Score(p, job.Job{Title: "GTM Engineer", WorkMode: job.WorkModeRemote})
	if none.Dropped != "nenhuma tecnologia do perfil" {
		t.Errorf("job with no profile skill: Dropped = %q", none.Dropped)
	}
}

func TestSkillsCap(t *testing.T) {
	p := example(t)
	p.Skills = append(p.Skills, profile.Skill{Name: "Kafka", Points: 30})
	j := job.Job{Title: "Dev go react typescript postgres kafka"}
	total := 0
	for _, r := range skills(p, j) {
		total += r.Points
	}
	if total != SkillMax {
		t.Fatalf("skill points = %d, want %d", total, SkillMax)
	}
}

func TestRank(t *testing.T) {
	p := example(t)
	jobs := []job.Job{
		{Source: "a", SourceID: "1", Title: "Dev Java Docker"},
		{Source: "a", SourceID: "2", Title: "Dev Go React", WorkMode: job.WorkModeRemote},
		{Source: "a", SourceID: "3", Title: "Dev Go", WorkMode: job.WorkModeOnsite},
		{Source: "a", SourceID: "4", Title: "Analista Comercial"},
	}
	kept, dropped := Rank(p, jobs)
	if dropped != 2 || len(kept) != 2 || kept[0].Job.SourceID != "2" {
		t.Fatalf("kept = %v, dropped = %d", kept, dropped)
	}
}

func TestSkillsDescriptionAndFalseFriends(t *testing.T) {
	p := example(t)
	points := func(j job.Job) map[string]int {
		got := map[string]int{}
		for _, r := range skills(p, j) {
			got[r.Text] = r.Points
		}
		return got
	}
	got := points(job.Job{Title: "Software Engineer", Description: "Our tech stack: java, kotlin, node.js, go, python, react"})
	if got["cita Go na descrição"] != 8 || got["cita React na descrição"] != 6 {
		t.Errorf("description-only = %v, want half points", got)
	}
	got = points(job.Job{Title: "Especialista GTM", Description: "a engenharia por trás do nosso go-to-market"})
	if _, ok := got["cita Go na descrição"]; ok {
		t.Errorf("go-to-market counted as Go: %v", got)
	}
	got = points(job.Job{Title: "Mobile Developer React Native"})
	if _, ok := got["pede React"]; ok {
		t.Errorf("React Native counted as React: %v", got)
	}
	got = points(job.Job{Title: "Dev React e React Native"})
	if got["pede React"] != 12 {
		t.Errorf("React next to React Native = %v", got)
	}
}

func TestLeadIsDropped(t *testing.T) {
	p := example(t)
	if got := Filter(p, job.Job{Title: "Head of Engineering", Seniority: job.SeniorityLead}); got == "" {
		t.Fatal("lead role passed a profile that does not accept it")
	}
}

func TestSkillsRequirementsAndURLs(t *testing.T) {
	p := example(t)
	points := func(j job.Job) map[string]int {
		got := map[string]int{}
		for _, r := range skills(p, j) {
			got[r.Text] = r.Points
		}
		return got
	}
	desc := "Sobre nós: nossa stack tem Go e Java.\n\nRequisitos:\n- Experiência com React.js\n\nDiferenciais:\n- TypeScript\n\nBenefícios: plano de saúde"
	got := points(job.Job{Title: "Dev Fullstack", Description: desc})
	want := map[string]int{
		"pede React nos requisitos":    12, // full: listed as a requirement
		"cita Go na descrição":         8,  // half: company stack paragraph
		"cita TypeScript na descrição": 3,  // half: nice to have
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s = %d, want %d (all: %v)", k, got[k], v, got)
		}
	}

	got = points(job.Job{
		Title:       "Mobile Developer React Native",
		Description: "Candidate-se: https://coodesh.com/vagas/mobile-react-native?utm_medium=react-brasil-vagas",
	})
	if _, ok := got["cita React na descrição"]; ok {
		t.Errorf("React counted from a URL: %v", got)
	}
}
