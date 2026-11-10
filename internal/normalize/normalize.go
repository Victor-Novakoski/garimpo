// Package normalize turns raw postings into comparable ones: it reads work
// mode, contract and seniority from labels and text, and merges duplicates.
package normalize

import (
	"strings"
	"unicode"

	"github.com/victor-novakoski/garimpo/internal/job"
)

// Fold lowercases s, strips accents and collapses spaces, so "Sênior",
// "SENIOR" and "senior" compare equal.
func Fold(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if plain, ok := accents[r]; ok {
			r = plain
		}
		b.WriteRune(r)
	}
	return strings.Join(strings.Fields(b.String()), " ")
}

// accents maps the accented letters used in Portuguese to plain ones.
var accents = map[rune]rune{
	'á': 'a', 'à': 'a', 'â': 'a', 'ã': 'a', 'ä': 'a',
	'é': 'e', 'è': 'e', 'ê': 'e', 'ë': 'e',
	'í': 'i', 'ì': 'i', 'î': 'i', 'ï': 'i',
	'ó': 'o', 'ò': 'o', 'ô': 'o', 'õ': 'o', 'ö': 'o',
	'ú': 'u', 'ù': 'u', 'û': 'u', 'ü': 'u',
	'ç': 'c', 'ñ': 'n',
}

// Words splits folded text into words, keeping symbols used in tech names
// (c#, c++, node.js).
func Words(s string) []string {
	return strings.FieldsFunc(Fold(s), func(r rune) bool {
		return !(unicode.IsLetter(r) || unicode.IsDigit(r) || strings.ContainsRune("#+.", r))
	})
}

// rule maps words to a value.
type rule[V any] struct {
	words []string
	value V
}

var workModeWords = []rule[job.WorkMode]{
	// Hybrid first: "remoto/híbrido" postings are not fully remote.
	{[]string{"hibrido", "hibrida", "hybrid"}, job.WorkModeHybrid},
	{[]string{"presencial", "onsite", "on site"}, job.WorkModeOnsite},
	{[]string{"remoto", "remota", "remote", "home office", "anywhere"}, job.WorkModeRemote},
}

var contractWords = []rule[job.Contract]{
	{[]string{"estagio", "estagiario", "internship"}, job.ContractInternship},
	{[]string{"pj", "pessoa juridica"}, job.ContractPJ},
	{[]string{"freelancer", "freelance", "freela"}, job.ContractFreelance},
	{[]string{"clt"}, job.ContractCLT},
}

// seniorityWords lists Pleno first: a "Pleno/Sênior" range is reachable
// for a mid-level person, so ranges resolve to their lowest common level.
var seniorityWords = []rule[job.Seniority]{
	{[]string{"pleno", "mid", "mid level"}, job.SeniorityMid},
	{[]string{"senior", "sr", "especialista", "lead", "staff", "principal"}, job.SenioritySenior},
	{[]string{"junior", "jr", "trainee"}, job.SeniorityJunior},
}

// Classify fills WorkMode, Contract and Seniority when the source left them
// unknown. The title wins over the location, which wins over the labels:
// community repos often carry contradicting labels ("Remoto" and "Híbrido")
// while the title states the real one. Description text is not used: it
// mentions other levels and models too often.
func Classify(j job.Job) job.Job {
	title := wordText(j.Title)
	location := wordText(j.Location)
	tags := wordText(strings.Join(j.Tags, " "))
	if j.WorkMode == job.WorkModeUnknown {
		j.WorkMode = firstMatch(workModeWords, title, location, tags)
	}
	if j.Contract == job.ContractUnknown {
		j.Contract = firstMatch(contractWords, title, tags)
	}
	if j.Seniority == job.SeniorityUnknown {
		j.Seniority = firstMatch(seniorityWords, title, tags)
	}
	return j
}

// firstMatch walks the texts in order of trust and returns the value of
// the first table entry found in the first text that matches any entry.
func firstMatch[V any](rules []rule[V], texts ...string) V {
	for _, text := range texts {
		for _, r := range rules {
			if containsAny(text, r.words) {
				return r.value
			}
		}
	}
	var zero V
	return zero
}

// wordText is s as space-separated words with a space on each side, ready
// for whole-word matching.
func wordText(s string) string {
	return " " + strings.Join(Words(s), " ") + " "
}

func containsAny(text string, words []string) bool {
	for _, w := range words {
		if strings.Contains(text, " "+w+" ") {
			return true
		}
	}
	return false
}

// Dedupe drops repeated postings, keeping the first seen. Two postings are
// the same when they share a Key (same source and id) or a Fingerprint
// (same company and title in different sources).
func Dedupe(jobs []job.Job) []job.Job {
	seenKey := make(map[string]bool, len(jobs))
	seenPrint := make(map[string]bool, len(jobs))
	out := make([]job.Job, 0, len(jobs))
	for _, j := range jobs {
		if seenKey[j.Key()] {
			continue
		}
		fp := j.Fingerprint()
		if fp != "" && seenPrint[fp] {
			continue
		}
		seenKey[j.Key()] = true
		if fp != "" {
			seenPrint[fp] = true
		}
		out = append(out, j)
	}
	return out
}

// techWords mark a posting as a technology job. Checked against the title
// and the source's labels or departments.
var techWords = []string{
	"engineer", "engineering", "engenheiro", "engenheira", "engenharia de software",
	"developer", "desenvolvedor", "desenvolvedora", "dev", "programador", "programadora",
	"software", "backend", "back end", "frontend", "front end", "fullstack", "full stack",
	"mobile", "android", "ios", "devops", "sre", "qa", "tech lead", "arquiteto", "arquiteta",
	"data", "dados", "machine learning", "golang", "go", "react", "node", "java", "python",
}

// IsTech reports whether the posting is a technology job. Community repos
// only post tech jobs; company boards mix in sales, finance and the rest.
func IsTech(j job.Job) bool {
	return containsAny(wordText(j.Title+" "+strings.Join(j.Tags, " ")), techWords)
}
