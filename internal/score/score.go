// Package score decides which jobs fit a profile and how well. The hard
// filter runs first and nothing later can undo it. The deterministic score
// goes from 0 to 70; the remaining 30 points belong to the AI step. Every
// point carries a reason a person can read.
package score

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/victor-novakoski/garimpo/internal/job"
	"github.com/victor-novakoski/garimpo/internal/normalize"
	"github.com/victor-novakoski/garimpo/internal/profile"
)

// Point budgets for each part of the deterministic score.
const (
	SkillMax     = profile.MaxSkillPoints
	WorkModeMax  = 10
	ContractMax  = 10
	SeniorityMax = 10
	Max          = SkillMax + WorkModeMax + ContractMax + SeniorityMax // 70
)

// Reason is one line of the explanation: what earned (or did not earn)
// points and how many.
type Reason struct {
	Points int    `json:"points"`
	Text   string `json:"text"`
}

func (r Reason) String() string { return fmt.Sprintf("%+d %s", r.Points, r.Text) }

// Result is the outcome for one job.
type Result struct {
	Job     job.Job  `json:"job"`
	Dropped string   `json:"dropped,omitempty"` // why the hard filter removed it
	Points  int      `json:"points"`
	Reasons []Reason `json:"reasons,omitempty"`
}

// Filter applies the hard rules. It returns "" when the job passes, or the
// reason it was dropped.
func Filter(p profile.Profile, j job.Job) string {
	if j.WorkMode != job.WorkModeUnknown && !slices.Contains(p.Accept.WorkModes, j.WorkMode) {
		return "modelo " + string(j.WorkMode) + " não aceito"
	}
	if j.Contract != job.ContractUnknown && !slices.Contains(p.Accept.Contracts, j.Contract) {
		return "contrato " + string(j.Contract) + " não aceito"
	}
	if j.Seniority != job.SeniorityUnknown && !slices.Contains(p.Accept.Seniorities, j.Seniority) {
		return "nível " + string(j.Seniority) + " não aceito"
	}
	text := " " + strings.Join(normalize.Words(j.Title+" "+j.Company+" "+strings.Join(j.Tags, " ")), " ") + " "
	for _, w := range p.Avoid.Words {
		if strings.Contains(text, " "+normalize.Fold(w)+" ") {
			return "palavra evitada: " + w
		}
	}
	company := normalize.Fold(j.Company)
	for _, c := range p.Avoid.Companies {
		if company != "" && company == normalize.Fold(c) {
			return "empresa evitada: " + j.Company
		}
	}
	location := " " + strings.Join(normalize.Words(j.Location+" "+j.Title), " ") + " "
	for _, l := range p.Avoid.Locations {
		if strings.Contains(location, " "+strings.Join(normalize.Words(l), " ")+" ") {
			return "local evitado: " + l
		}
	}
	return ""
}

// Score runs the filter and, when the job passes, the deterministic score.
func Score(p profile.Profile, j job.Job) Result {
	r := Result{Job: j}
	if r.Dropped = Filter(p, j); r.Dropped != "" {
		return r
	}
	r.Reasons = append(r.Reasons, skills(p, j)...)
	if len(p.Skills) > 0 && r.Reasons[0].Points == 0 {
		// Sales, data science or security roles at tech companies pass the
		// other rules; with none of the profile's skills they are noise.
		r.Reasons = nil
		r.Dropped = "nenhuma tecnologia do perfil"
		return r
	}
	r.Reasons = append(r.Reasons, workMode(p, j), contract(p, j), seniority(p, j))
	for _, reason := range r.Reasons {
		r.Points += reason.Points
	}
	return r
}

// skills gives each profile skill the posting asks for its points, capped
// at SkillMax. A skill in the title, the labels or the requirements section
// of the description earns full points; one found only elsewhere in the
// description earns half, since descriptions often carry a company-wide
// stack paragraph ("our stack: java, kotlin, go, react...").
func skills(p profile.Profile, j job.Job) []Reason {
	head := skillText(j.Title + " " + strings.Join(j.Tags, " "))
	body := skillText(j.Description)
	reqs := requirements(body)
	var reasons []Reason
	total := 0
	for _, s := range p.Skills {
		pts, where := 0, ""
		switch {
		case mentions(head, s):
			pts, where = s.Points, "pede "+s.Name
		case mentions(reqs, s):
			pts, where = s.Points, "pede "+s.Name+" nos requisitos"
		case mentions(body, s):
			pts, where = (s.Points+1)/2, "cita "+s.Name+" na descrição"
		default:
			continue
		}
		pts = min(pts, SkillMax-total)
		if pts <= 0 {
			break
		}
		total += pts
		reasons = append(reasons, Reason{pts, where})
	}
	if len(reasons) == 0 {
		reasons = append(reasons, Reason{0, "nenhuma tecnologia do perfil"})
	}
	return reasons
}

// requirementsStart and requirementsEnd are headings, as folded words, that
// open and close the part of a description listing what the job asks for.
var (
	requirementsStart = []string{
		"requisitos", "requirements", "qualificacoes", "qualifications",
		"o que esperamos", "o que buscamos", "o que voce precisa", "voce precisa ter",
		"what you ll need", "what we re looking for", "what you bring", "must have",
	}
	requirementsEnd = []string{
		"diferenciais", "desejavel", "desejaveis", "nice to have", "bonus points",
		"beneficios", "benefits", "o que oferecemos", "what we offer",
		"sobre a empresa", "sobre nos", "about us",
	}
)

// requirements returns the requirements section of a skillText, or "" when
// the description has no such heading.
func requirements(text string) string {
	start := firstIndex(text, requirementsStart)
	if start < 0 {
		return ""
	}
	rest := text[start:]
	if end := firstIndex(rest[1:], requirementsEnd); end >= 0 {
		rest = rest[:end+2] // keep the space before the closing heading
	}
	return rest
}

// firstIndex is the position of the earliest whole-word phrase in text.
func firstIndex(text string, phrases []string) int {
	best := -1
	for _, ph := range phrases {
		if i := strings.Index(text, " "+ph+" "); i >= 0 && (best < 0 || i < best) {
			best = i
		}
	}
	return best
}

// notSkills are phrases that contain a skill name but mean something else.
var notSkills = strings.NewReplacer(
	" go to market ", " gtm ",
	" react native ", " reactnative ", // mobile; add "React Native" as its own skill
)

// urls match links, whose paths and tracking parameters often carry words
// like "react" that say nothing about the job.
var urls = regexp.MustCompile(`(?i)(https?://|www\.)[^\s)\]>"']+`)

func skillText(s string) string {
	t := " " + strings.Join(normalize.Words(urls.ReplaceAllString(s, " ")), " ") + " "
	// Replace twice so overlapping matches ("go to market go to market") go too.
	return notSkills.Replace(notSkills.Replace(t))
}

func mentions(text string, s profile.Skill) bool {
	for _, n := range append([]string{s.Name}, s.Aliases...) {
		w := notSkills.Replace(" " + strings.Join(normalize.Words(n), " ") + " ")
		if strings.Contains(text, w) {
			return true
		}
	}
	return false
}

func workMode(p profile.Profile, j job.Job) Reason {
	switch {
	case j.WorkMode == job.WorkModeUnknown && normalize.City(j) != "":
		// "[Belo Horizonte, Brasil] Senior Software Engineer" is rarely remote.
		return Reason{0, "modelo não informado e a vaga cita " + normalize.City(j) + " (pode ser presencial)"}
	case j.WorkMode == job.WorkModeUnknown:
		// Low on purpose: for a remote-only search an unmarked job is often
		// on-site, and it should not outrank one that says remote.
		return Reason{WorkModeMax / 5, "modelo de trabalho não informado"}
	case slices.Contains(p.Prefer.WorkModes, j.WorkMode):
		return Reason{WorkModeMax, "modelo " + string(j.WorkMode)}
	default:
		return Reason{WorkModeMax * 2 / 5, "modelo " + string(j.WorkMode) + " (aceito, não preferido)"}
	}
}

func contract(p profile.Profile, j job.Job) Reason {
	switch {
	case j.Contract == job.ContractUnknown:
		return Reason{ContractMax / 2, "contrato não informado"}
	case slices.Contains(p.Prefer.Contracts, j.Contract):
		return Reason{ContractMax, "contrato " + string(j.Contract)}
	default:
		return Reason{ContractMax * 2 / 5, "contrato " + string(j.Contract) + " (aceito, não preferido)"}
	}
}

var levelOrder = map[job.Seniority]int{
	job.SeniorityJunior: 0,
	job.SeniorityMid:    1,
	job.SenioritySenior: 2,
	job.SeniorityLead:   3,
}

func seniority(p profile.Profile, j job.Job) Reason {
	switch {
	case j.Seniority == job.SeniorityUnknown:
		return Reason{SeniorityMax / 2, "nível não informado"}
	case j.Seniority == p.Seniority:
		return Reason{SeniorityMax, "nível " + string(j.Seniority)}
	case abs(levelOrder[j.Seniority]-levelOrder[p.Seniority]) == 1:
		return Reason{SeniorityMax * 3 / 5, "nível " + string(j.Seniority) + " (um acima ou abaixo)"}
	default:
		return Reason{0, "nível " + string(j.Seniority) + " (distante)"}
	}
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// Rank scores every job and returns the ones that passed, best first, and
// how many the filter dropped.
func Rank(p profile.Profile, jobs []job.Job) (kept []Result, dropped int) {
	for _, j := range jobs {
		r := Score(p, j)
		if r.Dropped != "" {
			dropped++
			continue
		}
		kept = append(kept, r)
	}
	slices.SortStableFunc(kept, func(a, b Result) int { return b.Points - a.Points })
	return kept, dropped
}
