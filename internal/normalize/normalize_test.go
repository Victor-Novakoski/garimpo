package normalize

import (
	"reflect"
	"testing"

	"github.com/victor-novakoski/garimpo/internal/job"
)

func TestFold(t *testing.T) {
	if got := Fold("  Desenvolvedor   SÊNIOR Híbrido "); got != "desenvolvedor senior hibrido" {
		t.Fatalf("Fold = %q", got)
	}
}

func TestWords(t *testing.T) {
	got := Words("[Remoto] Dev C#/Node.js e .NET - PJ. Fim.")
	want := []string{"remoto", "dev", "c#", "node.js", "e", ".net", "pj", "fim"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Words = %q, want %q", got, want)
	}
}

func TestClassify(t *testing.T) {
	tests := []struct {
		name string
		in   job.Job
		want job.Job
	}{
		{
			name: "from labels",
			in:   job.Job{Title: "Dev Go", Tags: []string{"Remoto", "PJ", "Pleno"}},
			want: job.Job{WorkMode: job.WorkModeRemote, Contract: job.ContractPJ, Seniority: job.SeniorityMid},
		},
		{
			name: "from title, hybrid wins over remote",
			in:   job.Job{Title: "[Remoto/Híbrido] Desenvolvedor Sênior React - CLT"},
			want: job.Job{WorkMode: job.WorkModeHybrid, Contract: job.ContractCLT, Seniority: job.SenioritySenior},
		},
		{
			name: "short forms",
			in:   job.Job{Title: "Dev Backend Jr - Presencial SP"},
			want: job.Job{WorkMode: job.WorkModeOnsite, Seniority: job.SeniorityJunior},
		},
		{
			name: "keeps what the source set",
			in:   job.Job{Title: "Dev Sênior presencial", WorkMode: job.WorkModeRemote},
			want: job.Job{WorkMode: job.WorkModeRemote, Seniority: job.SenioritySenior},
		},
		{
			name: "title wins over contradicting labels",
			in:   job.Job{Title: "[Remoto] Fullstack Developer - Jazida.com", Tags: []string{"Remoto", "Híbrido"}},
			want: job.Job{WorkMode: job.WorkModeRemote},
		},
		{
			name: "seniority range resolves to pleno",
			in:   job.Job{Title: "Mobile Developer React Native (Pleno)", Tags: []string{"Pleno", "Senior"}},
			want: job.Job{Seniority: job.SeniorityMid},
		},
		{
			name: "labels used when the title is silent",
			in:   job.Job{Title: "Dev Go", Tags: []string{"Remoto", "Híbrido", "Sênior"}},
			want: job.Job{WorkMode: job.WorkModeHybrid, Seniority: job.SenioritySenior},
		},
		{
			name: "location gives the work mode",
			in:   job.Job{Title: "Software Engineer", Location: "Remote - Brazil"},
			want: job.Job{WorkMode: job.WorkModeRemote},
		},
		{
			name: "levels as roman numerals",
			in:   job.Job{Title: "[Crédito] Software Engineer III - C#"},
			want: job.Job{Seniority: job.SeniorityMid},
		},
		{
			name: "level I is junior",
			in:   job.Job{Title: "SOFTWARE ENGINEER I .NET"},
			want: job.Job{Seniority: job.SeniorityJunior},
		},
		{
			name: "head is senior",
			in:   job.Job{Title: "Head of Engineering"},
			want: job.Job{Seniority: job.SenioritySenior},
		},
		{
			name: "pl/sql is not pleno",
			in:   job.Job{Title: "Analista PL/SQL"},
			want: job.Job{},
		},
		{
			name: "no false match inside words",
			in:   job.Job{Title: "Desenvolvedor Golang para plataforma de pjogos"},
			want: job.Job{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Classify(tt.in)
			if got.WorkMode != tt.want.WorkMode || got.Contract != tt.want.Contract || got.Seniority != tt.want.Seniority {
				t.Fatalf("Classify = (%q, %q, %q), want (%q, %q, %q)",
					got.WorkMode, got.Contract, got.Seniority,
					tt.want.WorkMode, tt.want.Contract, tt.want.Seniority)
			}
		})
	}
}

func TestDedupe(t *testing.T) {
	jobs := []job.Job{
		{Source: "a", SourceID: "1", Company: "Acme", Title: "Dev Go"},
		{Source: "a", SourceID: "1", Company: "Acme", Title: "Dev Go"},          // same key
		{Source: "b", SourceID: "9", Company: "ACME", Title: "[Remoto] dev go"}, // same opening elsewhere
		{Source: "b", SourceID: "10", Title: "Dev Go"},                          // no company: kept
		{Source: "b", SourceID: "11", Title: "Dev Go"},                          // no company: kept
		{Source: "b", SourceID: "12", Company: "Acme", Title: "Dev React"},      // other opening
		{Source: "c", SourceID: "1", Company: "WeatherWise",
			Title: "[Remoto] Desenvolvedor(a) Full Stack Sênior — AI & Agentes | WeatherWise | 4 dias / 32h"},
		{Source: "d", SourceID: "2", Company: "WeatherWise",
			Title: "[Remoto] Desenvolvedor(a) Full Stack Sênior - AI & Agentes - WeatherWise"},
		{Source: "e", SourceID: "3", Company: "WeatherWise",
			Title: "Full Stack Sênior AI & Agentes"},
		{Source: "e", SourceID: "4", Company: "WeatherWise",
			Title: "Designer de Produto"},
	}
	got := Dedupe(jobs)
	var keys []string
	for _, j := range got {
		keys = append(keys, j.Key())
	}
	want := []string{"a:1", "b:10", "b:11", "b:12", "c:1", "e:4"}
	if !reflect.DeepEqual(keys, want) {
		t.Fatalf("Dedupe keys = %v, want %v", keys, want)
	}
}

func TestIsTech(t *testing.T) {
	tests := []struct {
		job  job.Job
		want bool
	}{
		{job.Job{Title: "SENIOR SOFTWARE ENGINEER - GOLANG"}, true},
		{job.Job{Title: "Desenvolvedor(a) Backend Pleno"}, true},
		{job.Job{Title: "Analista", Tags: []string{"Engineering"}}, true},
		{job.Job{Title: "Back-end developer Node.js"}, true},
		{job.Job{Title: "Agente Stone - Consultor(a) Comercial Externo | Toledo"}, false},
		{job.Job{Title: "Account Executive - Hunter (Hibrido/SP)", Tags: []string{"Sales"}}, false},
		{job.Job{Title: "Analista Financeiro"}, false},
		{job.Job{Title: "Vendedor(a) Externo (6 horas) | Rio Verde/GO"}, false},
		{job.Job{Title: "Executivo(a) Comercial Externo | Anápolis/GO"}, false},
	}
	for _, tt := range tests {
		if got := IsTech(tt.job); got != tt.want {
			t.Errorf("IsTech(%q) = %v, want %v", tt.job.Title, got, tt.want)
		}
	}
}
