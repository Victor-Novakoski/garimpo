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
		Title:       "Dev Full Stack Pleno",
		Description: "Stack: Golang, React, TypeScript e PostgreSQL. Docker é um plus.",
		WorkMode:    job.WorkModeRemote,
		Contract:    job.ContractPJ,
		Seniority:   job.SeniorityMid,
	}
	r := Score(p, best)
	// Go 15 + React 12 + TypeScript 6 + PostgreSQL 5 + Docker 2 = 40, then 10+10+10.
	if r.Points != Max {
		t.Fatalf("Points = %d, want %d; reasons %v", r.Points, Max, r.Reasons)
	}

	weak := job.Job{Title: "Dev Java Sênior", Contract: job.ContractCLT, Seniority: job.SenioritySenior}
	r = Score(p, weak)
	// no skills 0 + unknown mode 5 + clt 4 + senior 6 = 15
	if r.Points != 15 {
		t.Fatalf("Points = %d, want 15; reasons %v", r.Points, r.Reasons)
	}
	if r.Reasons[0].Text != "nenhuma tecnologia do perfil" {
		t.Errorf("first reason = %v", r.Reasons[0])
	}
}

func TestSkillsCap(t *testing.T) {
	p := example(t)
	p.Skills = append(p.Skills, profile.Skill{Name: "Kafka", Points: 30})
	j := job.Job{Title: "Dev", Description: "go react typescript postgres kafka"}
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
		{Source: "a", SourceID: "1", Title: "Dev Java"},
		{Source: "a", SourceID: "2", Title: "Dev Go React", WorkMode: job.WorkModeRemote},
		{Source: "a", SourceID: "3", Title: "Dev Go", WorkMode: job.WorkModeOnsite},
	}
	kept, dropped := Rank(p, jobs)
	if dropped != 1 || len(kept) != 2 || kept[0].Job.SourceID != "2" {
		t.Fatalf("kept = %v, dropped = %d", kept, dropped)
	}
}
