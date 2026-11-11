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
	fields := strings.FieldsFunc(Fold(s), func(r rune) bool {
		return !(unicode.IsLetter(r) || unicode.IsDigit(r) || strings.ContainsRune("#+.", r))
	})
	words := fields[:0]
	for _, f := range fields {
		// A dot ends sentences ("PostgreSQL.") but starts ".net".
		f = strings.TrimRight(f, ".")
		if f != "" && f != "." {
			words = append(words, f)
		}
	}
	return words
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

// seniorityWords lists leadership first, since "Staff Software Engineer II"
// is a staff role whatever the number says. Pleno comes next: a
// "Pleno/Sênior" or "JR/PL" range is reachable for a mid-level person, so
// ranges resolve to Pleno.
var seniorityWords = []rule[job.Seniority]{
	{[]string{"staff", "principal", "head", "lead", "tech lead", "manager", "gerente", "coordenador", "coordenadora"}, job.SeniorityLead},
	{[]string{"pleno", "pl", "mid", "mid level", "ii", "iii"}, job.SeniorityMid},
	{[]string{"senior", "sr", "especialista", "iv"}, job.SenioritySenior},
	{[]string{"junior", "jr", "trainee", "associate", "i"}, job.SeniorityJunior},
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
	t := " " + strings.Join(Words(s), " ") + " "
	// "PL/SQL" is a language, not Pleno.
	return strings.ReplaceAll(t, " pl sql ", " plsql ")
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
// the same when they share a Key (same source and id), or when they come
// from the same company with titles that match after dropping tags, work
// mode, contract and level words (the same opening posted in several
// community repos with small title changes).
func Dedupe(jobs []job.Job) []job.Job {
	kept, _ := DedupeGroups(jobs)
	return kept
}

// DedupeGroups is Dedupe that also says which copies each kept posting
// stands for: copies maps the Key of a kept job to the Keys of the
// postings dropped as its duplicates. Whoever tracks a posting over time
// uses it to know that all those keys are the same opening.
func DedupeGroups(jobs []job.Job) (kept []job.Job, copies map[string][]string) {
	type seenTitle struct {
		tokens map[string]bool
		key    string
	}
	seenKey := make(map[string]bool, len(jobs))
	byCompany := make(map[string][]seenTitle)
	kept = make([]job.Job, 0, len(jobs))
	copies = make(map[string][]string)
	for _, j := range jobs {
		if seenKey[j.Key()] {
			continue
		}
		seenKey[j.Key()] = true
		company := Fold(j.Company)
		if company != "" {
			tokens := titleTokens(j.Title, company)
			original := ""
			for _, e := range byCompany[company] {
				if similar(tokens, e.tokens) {
					original = e.key
					break
				}
			}
			if original != "" {
				copies[original] = append(copies[original], j.Key())
				continue
			}
			byCompany[company] = append(byCompany[company], seenTitle{tokens, j.Key()})
		}
		kept = append(kept, j)
	}
	return kept, copies
}

// noiseWords say nothing about which opening a title is.
var noiseWords = map[string]bool{
	"remoto": true, "remota": true, "remote": true, "hibrido": true, "hybrid": true,
	"presencial": true, "pj": true, "clt": true, "vaga": true, "de": true, "da": true,
	"do": true, "e": true, "em": true, "para": true, "na": true, "no": true, "a": true,
	"o": true, "com": true, "and": true, "the": true, "for": true, "brasil": true,
}

func titleTokens(title, company string) map[string]bool {
	companyWords := make(map[string]bool)
	for _, w := range Words(company) {
		companyWords[w] = true
	}
	tokens := make(map[string]bool)
	for _, w := range Words(title) {
		if noiseWords[w] || companyWords[w] {
			continue
		}
		tokens[w] = true
	}
	return tokens
}

// similar reports whether two titles are the same opening: a Jaccard
// similarity of at least 0.6, or one title containing the other when the
// shorter has at least three words.
func similar(tokens, e map[string]bool) bool {
	common := 0
	for w := range tokens {
		if e[w] {
			common++
		}
	}
	union := len(tokens) + len(e) - common
	if union == 0 {
		return true
	}
	if float64(common)/float64(union) >= 0.6 {
		return true
	}
	shorter := min(len(tokens), len(e))
	return shorter >= 3 && common == shorter
}

// techWords mark a posting as a technology job. Checked against the title
// and the source's labels or departments.
var techWords = []string{
	"engineer", "engineering", "engenheiro", "engenheira", "engenharia de software",
	"developer", "desenvolvedor", "desenvolvedora", "dev", "programador", "programadora",
	"software", "backend", "back end", "frontend", "front end", "fullstack", "full stack",
	"mobile", "android", "ios", "devops", "sre", "qa", "tech lead", "arquiteto", "arquiteta",
	"data", "dados", "machine learning", "golang", "react", "node", "java", "python",
}

// Plain "go" is not a tech word: it is also the state code of Goiás
// ("Vendedor Externo | Rio Verde/GO"). Go jobs say developer or engineer.

// IsTech reports whether the posting is a technology job. Community repos
// only post tech jobs; company boards mix in sales, finance and the rest.
func IsTech(j job.Job) bool {
	return containsAny(wordText(j.Title+" "+strings.Join(j.Tags, " ")), techWords)
}

// cities are Brazilian cities with many tech jobs. A posting that names one
// and does not say remote is often on-site or hybrid.
var cities = []string{
	"São Paulo", "Rio de Janeiro", "Belo Horizonte", "Curitiba", "Porto Alegre",
	"Florianópolis", "Campinas", "Recife", "Salvador", "Fortaleza", "Brasília",
	"Goiânia", "Manaus", "Belém", "Joinville", "Blumenau", "Londrina",
	"Maringá", "Ribeirão Preto", "São José dos Campos", "Uberlândia",
	"João Pessoa", "Maceió", "Aracaju", "Teresina", "São Luís", "Cuiabá",
	"Campo Grande", "Sorocaba", "Barueri", "Osasco", "São Carlos",
}

// City returns the first city from the list named in the title or the
// location, or "".
func City(j job.Job) string {
	text := wordText(j.Title + " " + j.Location)
	for _, c := range cities {
		if strings.Contains(text, " "+strings.Join(Words(c), " ")+" ") {
			return c
		}
	}
	return ""
}
