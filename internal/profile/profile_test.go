package profile

import (
	"os"
	"path/filepath"
	"reflect"
	"strconv"
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

const minimal = "seniority: mid\naccept: {work_modes: [remote], contracts: [pj], seniorities: [mid]}\n"

// writeProfile writes a profile with the given about_files into dir.
func writeProfile(t *testing.T, dir string, files ...string) string {
	t.Helper()
	yml := minimal + "about: Escrito à mão.\nabout_files:\n"
	for _, f := range files {
		yml += "  - " + strconv.Quote(f) + "\n"
	}
	path := filepath.Join(dir, "profile.yaml")
	if err := os.WriteFile(path, []byte(yml), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadAboutFiles(t *testing.T) {
	dir := t.TempDir()
	cv := filepath.Join(dir, "curriculo.md")
	goals := filepath.Join(dir, "metas.md")
	if err := os.WriteFile(cv, []byte("# Currículo\nGo em produção."), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(goals, []byte("Quero remoto."), 0o600); err != nil {
		t.Fatal(err)
	}
	p, err := Load(writeProfile(t, dir, cv, goals))
	if err != nil {
		t.Fatal(err)
	}
	if p.About != "Escrito à mão." {
		t.Errorf("about = %q, want the hand-written text untouched", p.About)
	}
	want := []AboutText{{Name: "curriculo.md", Text: "# Currículo\nGo em produção."}, {Name: "metas.md", Text: "Quero remoto."}}
	if !reflect.DeepEqual(p.AboutTexts, want) {
		t.Errorf("about texts = %+v, want %+v", p.AboutTexts, want)
	}
}

func TestLoadAboutFilesExpandsHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.WriteFile(filepath.Join(home, "notas.md"), []byte("texto"), 0o600); err != nil {
		t.Fatal(err)
	}
	p, err := Load(writeProfile(t, t.TempDir(), "~/notas.md"))
	if err != nil {
		t.Fatal(err)
	}
	if len(p.AboutTexts) != 1 || p.AboutTexts[0].Text != "texto" {
		t.Errorf("about texts = %+v", p.AboutTexts)
	}
}

func TestLoadAboutFilesErrors(t *testing.T) {
	dir := t.TempDir()
	secret := "conteúdo pessoal que não pode vazar"
	big := filepath.Join(dir, "grande.md")
	if err := os.WriteFile(big, []byte(secret+strings.Repeat("x", MaxAboutFileBytes)), 0o600); err != nil {
		t.Fatal(err)
	}
	half := filepath.Join(dir, "metade.md")
	if err := os.WriteFile(half, []byte(secret+strings.Repeat("x", MaxAboutTotalBytes/3)), 0o600); err != nil {
		t.Fatal(err)
	}
	binary := filepath.Join(dir, "binario.md")
	if err := os.WriteFile(binary, []byte{0xff, 0xfe, 0x00}, 0o600); err != nil {
		t.Fatal(err)
	}
	missing := filepath.Join(dir, "nao-existe.md")
	tests := map[string]struct {
		files []string
		want  string
	}{
		"missing":        {[]string{missing}, missing},
		"relative":       {[]string{"notas.md"}, "absolute"},
		"directory":      {[]string{dir}, "not a regular file"},
		"too big":        {[]string{big}, "limit per file"},
		"too big in all": {[]string{half, half, half}, "together"},
		"not text":       {[]string{binary}, "UTF-8"},
	}
	for name, tc := range tests {
		_, err := Load(writeProfile(t, t.TempDir(), tc.files...))
		if err == nil || !strings.Contains(err.Error(), tc.want) || !strings.Contains(err.Error(), "about_files") {
			t.Errorf("%s: err = %v, want one naming %q", name, err, tc.want)
			continue
		}
		if strings.Contains(err.Error(), secret) {
			t.Errorf("%s: the error shows the file content: %v", name, err)
		}
	}
}

func TestParseDoesNotReadAboutFiles(t *testing.T) {
	p, err := Parse([]byte(minimal + "about_files: [/nao/existe.md]\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(p.AboutFiles) != 1 || p.AboutTexts != nil {
		t.Errorf("profile = %+v", p)
	}
}
