package profile

import (
	"strings"
	"testing"

	"github.com/victor-novakoski/garimpo/internal/job"
)

func TestLoadExample(t *testing.T) {
	p, err := Load("../../profile.example.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if p.Seniority != job.SeniorityMid || len(p.Skills) == 0 {
		t.Fatalf("profile = %+v", p)
	}
	if p.Skills[0].Name != "Go" || p.Skills[0].Aliases[0] != "golang" {
		t.Errorf("first skill = %+v", p.Skills[0])
	}
}

func TestParseRejectsTyposAndBadValues(t *testing.T) {
	tests := map[string]string{
		"unknown field": "seniority: mid\naccept: {work_modes: [remote], contracts: [pj], seniorities: [mid]}\nskils: []\n",
		"bad mode":      "seniority: mid\naccept: {work_modes: [anywhere], contracts: [pj], seniorities: [mid]}\n",
		"empty accept":  "seniority: mid\naccept: {work_modes: [remote], contracts: [], seniorities: [mid]}\n",
		"bad skill":     "seniority: mid\naccept: {work_modes: [remote], contracts: [pj], seniorities: [mid]}\nskills: [{name: Go}]\n",
	}
	for name, yml := range tests {
		if _, err := Parse([]byte(yml)); err == nil || !strings.Contains(err.Error(), "profile") {
			t.Errorf("%s: err = %v, want a profile error", name, err)
		}
	}
}
