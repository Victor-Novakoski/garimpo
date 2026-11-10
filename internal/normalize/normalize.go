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

var workModeWords = []struct {
	words []string
	mode  job.WorkMode
}{
	// Hybrid first: "remoto/híbrido" postings are not fully remote.
	{[]string{"hibrido", "hibrida", "hybrid"}, job.WorkModeHybrid},
	{[]string{"presencial", "onsite", "on-site"}, job.WorkModeOnsite},
	{[]string{"remoto", "remota", "remote", "home office", "anywhere"}, job.WorkModeRemote},
}

var contractWords = []struct {
	words    []string
	contract job.Contract
}{
	{[]string{"estagio", "estagiario", "internship"}, job.ContractInternship},
	{[]string{"pj", "pessoa juridica"}, job.ContractPJ},
	{[]string{"freelancer", "freelance", "freela"}, job.ContractFreelance},
	{[]string{"clt"}, job.ContractCLT},
}

var seniorityWords = []struct {
	words     []string
	seniority job.Seniority
}{
	{[]string{"senior", "sr", "especialista", "lead", "staff", "principal"}, job.SenioritySenior},
	{[]string{"pleno", "pl", "mid", "mid-level"}, job.SeniorityMid},
	{[]string{"junior", "jr", "trainee"}, job.SeniorityJunior},
}

// Classify fills WorkMode, Contract and Seniority when the source left them
// unknown, looking at tags first and then the title. Description text is
// not used: it mentions other levels and models too often.
func Classify(j job.Job) job.Job {
	text := " " + strings.Join(Words(strings.Join(j.Tags, " ")+" "+j.Title), " ") + " "
	if j.WorkMode == job.WorkModeUnknown {
		for _, w := range workModeWords {
			if containsAny(text, w.words) {
				j.WorkMode = w.mode
				break
			}
		}
	}
	if j.Contract == job.ContractUnknown {
		for _, c := range contractWords {
			if containsAny(text, c.words) {
				j.Contract = c.contract
				break
			}
		}
	}
	if j.Seniority == job.SeniorityUnknown {
		for _, s := range seniorityWords {
			if containsAny(text, s.words) {
				j.Seniority = s.seniority
				break
			}
		}
	}
	return j
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
