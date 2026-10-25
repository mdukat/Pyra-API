package main

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// WordInfo describes the morphology of a single dictionary word (or the
// simple phrase patterns we know how to inflect).
type WordInfo struct {
	Lemma      string
	POS        string // noun, adj, verb, phrase, other
	Gender     string // m, f, n (nouns/adjectives)
	Animate    bool
	Personal   bool
	PluralOnly bool
	Phrase     *PhraseInfo
}

// PhraseInfo describes a small set of phrase patterns that can be inflected
// as a unit: adjective + noun, noun + adjective, two adjectives + noun,
// and verb + "się".
type PhraseInfo struct {
	Lemma    string
	Words    []string
	POS      string // adjnoun, adjadjnoun, verbrefl, other
	AdjFirst bool
	Adj      WordInfo
	Adjs     []WordInfo
	Noun     WordInfo
	Verb     WordInfo
}

var functionWords = setOf(
	"się", "nie", "tak", "i", "a", "ale", "albo", "lub", "czy", "że", "by",
	"aby", "żeby", "gdy", "kiedy", "gdzie", "dokąd", "stąd", "tam", "tu",
	"tutaj", "teraz", "potem", "zawsze", "nigdy", "często", "rzadko", "bardzo",
	"trochę", "dużo", "mało", "tylko", "jeszcze", "już", "też", "także",
	"nawet", "przecież", "jednak", "więc", "dlatego", "bo", "ponieważ",
	"jeśli", "jeżeli", "jak", "dlaczego", "ile", "tyle", "wiele", "kilka",
	"parę", "do", "w", "we", "z", "ze", "na", "o", "od", "po", "przy", "bez",
	"pod", "nad", "przed", "za", "przez", "między", "dla", "ku", "według",
	"oprócz", "wokół", "obok", "blisko", "daleko", "wczoraj", "dziś",
	"jutro", "wtedy", "gdzieś", "kiedyś", "jakoś", "prawie", "zupełnie",
	"całkiem", "chyba", "pewnie", "może", "razem", "osobno", "dookoła",
	"rano", "wieczorem", "dawniej", "obecnie", "szybko", "wolno", "dobrze",
	"źle", "ładnie", "brzydko", "głośno", "cicho", "ciepło", "zimno",
	"gorąco", "mokro", "sucho", "jasno", "ciemno", "wysoko", "nisko",
	"drogo", "tanio", "słodko", "kwaśno", "smacznie", "brudno", "czysto",
	"wesoło", "smutno", "łatwo", "trudno", "prosto", "krzywo", "gładko",
	"ostro", "długo", "krótko", "młodo", "staro", "nowo", "prędko",
	"wcześnie", "późno", "akurat", "właśnie", "przecież", "chyba", "ino",
	"jeno", "jino", "dyć", "cheba", "bogać", "hale", "aż", "już",
)

var pluralNouns = setOf(
	"spodnie", "okulary", "drzwi", "pieniądze", "nożyce", "sanki", "wrotki",
	"usta", "plecy", "urodziny", "imieniny", "wakacje", "ferie", "wczasy",
	"skrzypce", "dziąsła", "fusy", "pomyje", "grabie", "widły", "gable",
	"dokumenty", "cukierki", "drobiazgi", "firanki", "krzaki", "majtki",
	"nogi", "obierki", "prążki", "resztki", "skwarki", "tasiemki", "łydki",
	"mydliny", "tkaniny", "plwociny", "dąsy", "sanie", "schody", "urodziny",
	"piegi", "wąsy", "wrota", "wnętrzności", "pleśniawki",
	// Dialect plural-only words that appear as translation targets.
	"dudki", "piniuchy", "blyce", "bryle", "patrzydła", "sekulory",
	"hoje", "pory", "kujony", "kujuny", "okulaki", "okuloki", "pieruny",
	"pioruny", "badeje", "badejki", "badki",
)

// Nouns that are grammatically masculine despite ending in -a.
var mascA = setOf(
	"artysta", "kolega", "mężczyzna", "kierowca", "rowerzysta", "turysta",
	"dentysta", "specjalista", "pianista", "sportowiec", "morderca",
	"władca", "sędzia", "hrabia", "poeta", "satelita", "astronauta",
)

var animateMasc = setOf(
	"pies", "kot", "koń", "człowiek", "chłop", "chłopak", "mężczyzna",
	"brat", "ojciec", "syn", "mąż", "uczeń", "nauczyciel", "lekarz",
	"aktor", "kierowca", "rowerzysta", "ptak", "kozioł", "baran", "wół",
	"byk", "orzeł", "wilk", "lis", "zając", "jeleń", "niedźwiedź",
	"tygrys", "lew", "słoń", "wąż", "robak", "motyl", "pszczoła",
	"kejter", "kiejter", "kociamber", "bąbas", "bymbas", "gzubek",
	"gzubol", "knajder", "knajtek", "knajter", "boźdór", "chłopyszek",
	"synek", "dziadek", "wujek", "stryjek", "pan", "człek",
)

var nounExceptions = setOf(
	"paznokieć", "łokieć", "miłość", "kość", "część", "przyjemność",
	"radość", "smutek", "moc", "noc", "pomoc", "taniec", "koniec",
	"miesiąc", "październik", "listopad", "grudzień", "styczeń", "luty",
	"kwiecień", "czerwiec", "lipiec", "sierpień", "wrzesień",
)

var cVerbs = setOf("móc", "piec", "biec", "wlec", "strzec", "rzec", "siec", "tłuc")

// Adjectives that end in -a but are not feminine nouns.  This is intentionally
// small: phrase-level inflection is a bonus, not a requirement.
var feminineAdj = setOf("biała", "czarna", "czerwona", "zielona", "niebieska",
	"żółta", "duża", "mała", "nowa", "stara", "dobra", "zła", "ładna",
	"brzydka", "smaczna", "zimna", "ciepła", "droga", "tania", "polska",
	"mokra", "sucha", "wesoła", "smutna", "wysoka", "niska", "długa",
	"krótka", "młoda", "stara", "ciężka", "lekka", "szybka", "wolna")

func setOf(items ...string) map[string]bool {
	m := make(map[string]bool, len(items))
	for _, s := range items {
		m[s] = true
	}
	return m
}

func analyzeWord(word string) WordInfo {
	w := normalizeKey(word)
	info := WordInfo{Lemma: w}
	if w == "" {
		info.POS = "other"
		return info
	}
	if strings.Contains(w, " ") {
		pi := analyzePhrase(w)
		info.POS = "phrase"
		info.Phrase = &pi
		return info
	}
	if functionWords[w] {
		info.POS = "other"
		return info
	}
	if pluralNouns[w] {
		info.POS = "noun"
		info.Gender = "n"
		info.PluralOnly = true
		return info
	}
	if mascA[w] {
		info.POS = "noun"
		info.Gender = "m"
		info.Animate = true
		info.Personal = true
		return info
	}
	if feminineAdj[w] {
		info.POS = "adj"
		return info
	}
	if isVerbLike(w) {
		info.POS = "verb"
		return info
	}
	if isAdjectiveLike(w) {
		info.POS = "adj"
		return info
	}
	if strings.HasSuffix(w, "ość") {
		info.POS = "noun"
		info.Gender = "f"
		return info
	}
	if strings.HasSuffix(w, "a") {
		info.POS = "noun"
		info.Gender = "f"
		return info
	}
	if strings.HasSuffix(w, "o") || strings.HasSuffix(w, "e") ||
		strings.HasSuffix(w, "ę") || strings.HasSuffix(w, "um") {
		info.POS = "noun"
		info.Gender = "n"
		return info
	}
	if isConsonantEnd(w) {
		info.POS = "noun"
		info.Gender = "m"
		info.Animate = animateMasc[w]
		info.Personal = animateMasc[w] && isPersonWord(w)
		return info
	}
	info.POS = "other"
	return info
}

func isPersonWord(w string) bool {
	switch w {
	case "człowiek", "chłop", "chłopak", "mężczyzna", "brat", "ojciec", "syn",
		"mąż", "uczeń", "nauczyciel", "lekarz", "aktor", "kierowca",
		"rowerzysta", "pan", "dziadek", "wujek", "stryjek", "synek":
		return true
	}
	return false
}

func isVerbLike(w string) bool {
	if nounExceptions[w] {
		return false
	}
	if strings.HasSuffix(w, "ować") || strings.HasSuffix(w, "iwać") ||
		strings.HasSuffix(w, "ywać") || strings.HasSuffix(w, "nąć") ||
		strings.HasSuffix(w, "ąć") {
		return true
	}
	if strings.HasSuffix(w, "eć") {
		return true
	}
	if strings.HasSuffix(w, "ić") || strings.HasSuffix(w, "yć") {
		return true
	}
	if strings.HasSuffix(w, "ać") {
		return true
	}
	if strings.HasSuffix(w, "ść") && !strings.HasSuffix(w, "ość") {
		return true
	}
	if cVerbs[w] {
		return true
	}
	return false
}

func isAdjectiveLike(w string) bool {
	for _, suf := range []string{"owy", "ski", "cki", "isty", "aty", "liwy",
		"alny", "elny", "any", "ony", "ęty", "szy", "ższy", "ejszy", "ny",
		"ty", "ły"} {
		if strings.HasSuffix(w, suf) {
			return true
		}
	}
	if strings.HasSuffix(w, "y") || strings.HasSuffix(w, "i") {
		return true
	}
	return false
}

func isConsonantEnd(w string) bool {
	r, _ := utf8.DecodeLastRuneInString(w)
	if !unicode.IsLetter(r) {
		return false
	}
	return !isVowel(r)
}

func isVowel(r rune) bool {
	switch r {
	case 'a', 'e', 'i', 'o', 'u', 'y', 'ą', 'ę':
		return true
	}
	return false
}

func analyzePhrase(s string) PhraseInfo {
	words := strings.Fields(s)
	pi := PhraseInfo{Lemma: s, Words: words, POS: "other"}
	if len(words) == 2 && words[1] == "się" {
		v := analyzeWord(words[0])
		if v.POS == "verb" {
			pi.POS = "verbrefl"
			pi.Verb = v
			return pi
		}
	}
	if len(words) == 2 {
		a, b := analyzeWord(words[0]), analyzeWord(words[1])
		if a.POS == "adj" && b.POS == "noun" {
			pi.POS = "adjnoun"
			pi.AdjFirst = true
			pi.Adj = a
			pi.Noun = b
			return pi
		}
		if a.POS == "noun" && b.POS == "adj" {
			pi.POS = "adjnoun"
			pi.AdjFirst = false
			pi.Adj = b
			pi.Noun = a
			return pi
		}
	}
	if len(words) == 3 {
		a, b, c := analyzeWord(words[0]), analyzeWord(words[1]), analyzeWord(words[2])
		if a.POS == "adj" && b.POS == "adj" && c.POS == "noun" {
			pi.POS = "adjadjnoun"
			pi.Adjs = []WordInfo{a, b}
			pi.Noun = c
			return pi
		}
	}
	return pi
}

// generateForms returns form-key -> surface-form.  Keys are shared between
// source and target words so that forms can be paired:
//
//	noun:      sg|nom, pl|gen, ...
//	adjective: sg|nom|m, pl|nom|np, ...
//	verb:      inf, pres|1|sg, past|sg|m, ...
func generateForms(info WordInfo) map[string]string {
	switch info.POS {
	case "noun":
		return nounForms(info)
	case "adj":
		return adjForms(info)
	case "verb":
		return verbForms(info)
	case "phrase":
		if info.Phrase == nil {
			return map[string]string{"base": info.Lemma}
		}
		return phraseForms(*info.Phrase)
	default:
		return map[string]string{"base": info.Lemma}
	}
}

func phraseForms(pi PhraseInfo) map[string]string {
	if pi.POS == "verbrefl" {
		base := verbForms(pi.Verb)
		out := make(map[string]string, len(base))
		for k, v := range base {
			out[k] = v + " się"
		}
		return out
	}
	if pi.POS != "adjnoun" && pi.POS != "adjadjnoun" {
		return map[string]string{"base": pi.Lemma}
	}
	out := make(map[string]string)
	for _, c := range cases {
		for _, n := range numbers {
			noun := nounForms(pi.Noun)[c+"|"+n]
			if noun == "" {
				continue
			}
			g := pi.Noun.Gender
			if n == "pl" {
				g = "np"
			}
			if pi.POS == "adjadjnoun" {
				a1 := adjForms(pi.Adjs[0])[c+"|"+n+"|"+g]
				a2 := adjForms(pi.Adjs[1])[c+"|"+n+"|"+g]
				if a1 == "" || a2 == "" {
					continue
				}
				out[c+"|"+n] = a1 + " " + a2 + " " + noun
				continue
			}
			adj := adjForms(pi.Adj)[c+"|"+n+"|"+g]
			if adj == "" {
				continue
			}
			if pi.AdjFirst {
				out[c+"|"+n] = adj + " " + noun
			} else {
				out[c+"|"+n] = noun + " " + adj
			}
		}
	}
	if len(out) == 0 {
		out["base"] = pi.Lemma
	}
	return out
}

var cases = []string{"nom", "gen", "dat", "acc", "ins", "loc", "voc"}
var numbers = []string{"sg", "pl"}

// ---------------------------------------------------------------------------
// Nouns
// ---------------------------------------------------------------------------

var nounOverrides = map[string]map[string]string{
	"pies": {
		"nom|sg": "pies", "gen|sg": "psa", "dat|sg": "psu", "acc|sg": "psa",
		"ins|sg": "psem", "loc|sg": "psie", "voc|sg": "psie",
		"nom|pl": "psy", "gen|pl": "psów", "dat|pl": "psom", "acc|pl": "psy",
		"ins|pl": "psami", "loc|pl": "psach", "voc|pl": "psy",
	},
	"koń": {
		"nom|sg": "koń", "gen|sg": "konia", "dat|sg": "koniowi", "acc|sg": "konia",
		"ins|sg": "koniem", "loc|sg": "koniu", "voc|sg": "koniu",
		"nom|pl": "konie", "gen|pl": "koni", "dat|pl": "koniom", "acc|pl": "konie",
		"ins|pl": "końmi", "loc|pl": "koniach", "voc|pl": "konie",
	},
	"człowiek": {
		"nom|sg": "człowiek", "gen|sg": "człowieka", "dat|sg": "człowiekowi",
		"acc|sg": "człowieka", "ins|sg": "człowiekiem", "loc|sg": "człowieku",
		"voc|sg": "człowieku", "nom|pl": "ludzie", "gen|pl": "ludzi",
		"dat|pl": "ludziom", "acc|pl": "ludzi", "ins|pl": "ludźmi",
		"loc|pl": "ludziach", "voc|pl": "ludzie",
	},
	"dziecko": {
		"nom|sg": "dziecko", "gen|sg": "dziecka", "dat|sg": "dziecku",
		"acc|sg": "dziecko", "ins|sg": "dzieckiem", "loc|sg": "dziecku",
		"voc|sg": "dziecko", "nom|pl": "dzieci", "gen|pl": "dzieci",
		"dat|pl": "dzieciom", "acc|pl": "dzieci", "ins|pl": "dziećmi",
		"loc|pl": "dzieciach", "voc|pl": "dzieci",
	},
	"brat": {
		"nom|sg": "brat", "gen|sg": "brata", "dat|sg": "bratu", "acc|sg": "brata",
		"ins|sg": "bratem", "loc|sg": "bracie", "voc|sg": "bracie",
		"nom|pl": "bracia", "gen|pl": "braci", "dat|pl": "braciom",
		"acc|pl": "braci", "ins|pl": "braćmi", "loc|pl": "braciach",
		"voc|pl": "bracia",
	},
	"przyjaciel": {
		"nom|sg": "przyjaciel", "gen|sg": "przyjaciela", "dat|sg": "przyjacielowi",
		"acc|sg": "przyjaciela", "ins|sg": "przyjacielem", "loc|sg": "przyjacielu",
		"voc|sg": "przyjacielu", "nom|pl": "przyjaciele", "gen|pl": "przyjaciół",
		"dat|pl": "przyjaciołom", "acc|pl": "przyjaciół", "ins|pl": "przyjaciółmi",
		"loc|pl": "przyjaciołach", "voc|pl": "przyjaciele",
	},
	"kejter": {
		"nom|sg": "kejter", "gen|sg": "kejtra", "dat|sg": "kejtrowi",
		"acc|sg": "kejtra", "ins|sg": "kejtrem", "loc|sg": "kejtrze",
		"voc|sg": "kejtrze", "nom|pl": "kejtry", "gen|pl": "kejtrów",
		"dat|pl": "kejtrom", "acc|pl": "kejtry", "ins|pl": "kejtrami",
		"loc|pl": "kejtrach", "voc|pl": "kejtry",
	},
	"kiejter": {
		"nom|sg": "kiejter", "gen|sg": "kiejtra", "dat|sg": "kiejtrowi",
		"acc|sg": "kiejtra", "ins|sg": "kiejtrem", "loc|sg": "kiejtrze",
		"voc|sg": "kiejtrze", "nom|pl": "kiejtry", "gen|pl": "kiejtrów",
		"dat|pl": "kiejtrom", "acc|pl": "kiejtry", "ins|pl": "kiejtrami",
		"loc|pl": "kiejtrach", "voc|pl": "kiejtry",
	},
	"kociamber": {
		"nom|sg": "kociamber", "gen|sg": "kociambra", "dat|sg": "kociambrowi",
		"acc|sg": "kociambra", "ins|sg": "kociambrem", "loc|sg": "kociambrze",
		"voc|sg": "kociambrze", "nom|pl": "kociambry", "gen|pl": "kociambrów",
		"dat|pl": "kociambrom", "acc|pl": "kociambry", "ins|pl": "kociambrami",
		"loc|pl": "kociambrach", "voc|pl": "kociambry",
	},
	"spodnie": {
		"nom|pl": "spodnie", "gen|pl": "spodni", "dat|pl": "spodniom",
		"acc|pl": "spodnie", "ins|pl": "spodniami", "loc|pl": "spodniach",
		"voc|pl": "spodnie",
	},
	"okulary": {
		"nom|pl": "okulary", "gen|pl": "okularów", "dat|pl": "okularom",
		"acc|pl": "okulary", "ins|pl": "okularami", "loc|pl": "okularach",
		"voc|pl": "okulary",
	},
	"pieniądze": {
		"nom|pl": "pieniądze", "gen|pl": "pieniędzy", "dat|pl": "pieniądzom",
		"acc|pl": "pieniądze", "ins|pl": "pieniędzmi", "loc|pl": "pieniądzach",
		"voc|pl": "pieniądze",
	},
	"drzwi": {
		"nom|pl": "drzwi", "gen|pl": "drzwi", "dat|pl": "drzwiom",
		"acc|pl": "drzwi", "ins|pl": "drzwiami", "loc|pl": "drzwiach",
		"voc|pl": "drzwi",
	},
	"usta": {
		"nom|pl": "usta", "gen|pl": "ust", "dat|pl": "ustom",
		"acc|pl": "usta", "ins|pl": "ustami", "loc|pl": "ustach", "voc|pl": "usta",
	},
	"plecy": {
		"nom|pl": "plecy", "gen|pl": "pleców", "dat|pl": "plecom",
		"acc|pl": "plecy", "ins|pl": "plecami", "loc|pl": "plecach", "voc|pl": "plecy",
	},
	"ludzie": {
		"nom|pl": "ludzie", "gen|pl": "ludzi", "dat|pl": "ludziom",
		"acc|pl": "ludzi", "ins|pl": "ludźmi", "loc|pl": "ludziach", "voc|pl": "ludzie",
	},
}

func nounForms(info WordInfo) map[string]string {
	if ov, ok := nounOverrides[info.Lemma]; ok {
		out := make(map[string]string, len(ov))
		for k, v := range ov {
			out[k] = v
		}
		return out
	}
	if info.PluralOnly || info.Gender == "pl" {
		return pluralNounForms(info)
	}
	switch info.Gender {
	case "f":
		if strings.HasSuffix(info.Lemma, "ość") {
			return feminineConsonantForms(info.Lemma)
		}
		return feminineNounForms(info.Lemma)
	case "n":
		return neuterNounForms(info.Lemma)
	default:
		return masculineNounForms(info)
	}
}

func feminineNounForms(lemma string) map[string]string {
	stem := strings.TrimSuffix(lemma, "a")
	genSg := stem + feminineGenSuffix(stem)
	oblique := feminineOblique(stem)
	nomPl := feminineNomPl(stem)
	forms := map[string]string{
		"nom|sg": lemma,
		"gen|sg": genSg,
		"dat|sg": oblique,
		"acc|sg": stem + "ę",
		"ins|sg": stem + "ą",
		"loc|sg": oblique,
		"voc|sg": stem + "o",
		"nom|pl": nomPl,
		"gen|pl": feminineGenPl(lemma, stem),
		"dat|pl": stem + "om",
		"acc|pl": nomPl,
		"ins|pl": stem + "ami",
		"loc|pl": stem + "ach",
		"voc|pl": nomPl,
	}
	return forms
}

func feminineGenSuffix(stem string) string {
	r, _ := utf8.DecodeLastRuneInString(stem)
	switch r {
	case 'k', 'g', 'i', 'y', 'ś', 'ć', 'ź', 'ń', 'j':
		return "i"
	case 'c', 'ż', 'l':
		return "y"
	}
	return "y"
}

func feminineOblique(stem string) string {
	if strings.HasSuffix(stem, "ch") {
		return stem[:len(stem)-2] + "sze"
	}
	for _, suf := range []string{"cz", "sz", "rz", "dz"} {
		if strings.HasSuffix(stem, suf) {
			return stem + "y"
		}
	}
	r, _ := utf8.DecodeLastRuneInString(stem)
	switch r {
	case 'i', 'y':
		return stem
	case 'c', 'ż', 'l', 'j', 'ń', 'ś', 'ć', 'ź':
		return stem + "y"
	case 'k':
		return stem[:len(stem)-1] + "ce"
	case 'g':
		return stem[:len(stem)-1] + "dze"
	}
	return softenStem(stem) + "e"
}

func feminineNomPl(stem string) string {
	r, _ := utf8.DecodeLastRuneInString(stem)
	switch r {
	case 'k', 'g':
		return stem + "i"
	case 'c', 'ż', 'l', 'j', 'ń', 'ś', 'ć', 'ź', 'i', 'y':
		return stem + "e"
	}
	return stem + "y"
}

func feminineGenPl(lemma, stem string) string {
	if strings.HasSuffix(lemma, "ia") {
		return strings.TrimSuffix(lemma, "ia")
	}
	r, _ := utf8.DecodeLastRuneInString(stem)
	if r == 'k' || r == 'g' {
		if len(stem) >= 2 {
			prev, _ := utf8.DecodeLastRuneInString(stem[:len(stem)-1])
			if !isVowel(prev) {
				return stem[:len(stem)-1] + "e" + string(r)
			}
		}
	}
	return stem
}

func feminineConsonantForms(lemma string) map[string]string {
	stem := strings.TrimSuffix(lemma, "ć") // miłoś
	return map[string]string{
		"nom|sg": lemma,
		"gen|sg": stem + "ci",
		"dat|sg": stem + "ci",
		"acc|sg": lemma,
		"ins|sg": stem + "cią",
		"loc|sg": stem + "ci",
		"voc|sg": lemma,
		"nom|pl": stem + "ci",
		"gen|pl": stem + "ci",
		"dat|pl": stem + "ciom",
		"acc|pl": stem + "ci",
		"ins|pl": stem + "ciami",
		"loc|pl": stem + "ciach",
		"voc|pl": stem + "ci",
	}
}

func masculineNounForms(info WordInfo) map[string]string {
	stem := info.Lemma
	gen := masculineGenForm(stem)
	nomPl := masculineNomPl(stem)
	genPl := masculineGenPl(stem)
	acc := stem
	accPl := nomPl
	if info.Animate {
		acc = gen
		if info.Personal {
			accPl = genPl
		}
	}
	ins := stem + "em"
	if strings.HasSuffix(stem, "k") || strings.HasSuffix(stem, "g") {
		ins = stem + "iem"
	}
	return map[string]string{
		"nom|sg": stem,
		"gen|sg": gen,
		"dat|sg": stem + "owi",
		"acc|sg": acc,
		"ins|sg": ins,
		"loc|sg": masculineLoc(stem),
		"voc|sg": masculineVoc(stem),
		"nom|pl": nomPl,
		"gen|pl": genPl,
		"dat|pl": stem + "om",
		"acc|pl": accPl,
		"ins|pl": stem + "ami",
		"loc|pl": stem + "ach",
		"voc|pl": nomPl,
	}
}

var masculineGenU = setOf(
	"sklep", "interes", "tramwaj", "rower", "telefon", "samochód", "dom",
	"cukier", "miód", "stół", "las", "bór", "bank", "hotel", "szpital",
	"miesiąc", "rok", "wieczór", "poranek", "dzień", "tydzień", "kraj",
	"świat", "grzyb", "deszcz", "śnieg", "lód", "miód", "ocet", "talerz",
	"nóż", "widelec", "garnek", "kubek", "telefon", "komputer", "telewizor",
)

var masculineGenMap = map[string]string{
	"chleb": "chleba", "ser": "sera", "pies": "psa", "kot": "kota",
	"koń": "konia", "kejter": "kejtra", "kiejter": "kiejtra",
	"kociamber": "kociambra", "byb": "byba", "śrup": "śrupa",
	"bakalorz": "bakalarza", "rower": "roweru", "tramwaj": "tramwaju",
	"sklep": "sklepu", "interes": "interesu",
}

func masculineGenForm(stem string) string {
	if g, ok := masculineGenMap[stem]; ok {
		return g
	}
	if masculineGenU[stem] {
		return stem + "u"
	}
	return stem + "a"
}

func masculineLoc(stem string) string {
	if strings.HasSuffix(stem, "ch") {
		return stem + "u"
	}
	for _, suf := range []string{"cz", "sz", "rz", "dz"} {
		if strings.HasSuffix(stem, suf) {
			return stem + "u"
		}
	}
	r, _ := utf8.DecodeLastRuneInString(stem)
	switch r {
	case 'k', 'g', 'h', 'c', 'ż', 'l', 'j', 'i', 'y':
		return stem + "u"
	}
	return softenStem(stem) + "e"
}

func masculineVoc(stem string) string {
	if strings.HasSuffix(stem, "ch") {
		return stem + "u"
	}
	for _, suf := range []string{"cz", "sz", "rz", "dz"} {
		if strings.HasSuffix(stem, suf) {
			return stem + "u"
		}
	}
	r, _ := utf8.DecodeLastRuneInString(stem)
	switch r {
	case 'k', 'g', 'h', 'c', 'ż', 'l', 'j':
		return stem + "u"
	}
	return softenStem(stem) + "e"
}

func masculineNomPl(stem string) string {
	for _, suf := range []string{"cz", "sz", "rz", "dz"} {
		if strings.HasSuffix(stem, suf) {
			return stem + "e"
		}
	}
	r, _ := utf8.DecodeLastRuneInString(stem)
	switch r {
	case 'k', 'g':
		return stem + "i"
	case 'j', 'l', 'ż', 'c', 'ś', 'ć', 'ź', 'ń':
		return stem + "e"
	}
	return stem + "y"
}

func masculineGenPl(stem string) string {
	if strings.HasSuffix(stem, "er") && len(stem) > 3 {
		prev, _ := utf8.DecodeLastRuneInString(stem[:len(stem)-2])
		if !isVowel(prev) && stem != "rower" && stem != "kuter" && stem != "sweter" {
			return stem[:len(stem)-2] + "rów"
		}
	}
	if strings.HasSuffix(stem, "eł") {
		return stem[:len(stem)-2] + "łów"
	}
	return stem + "ów"
}

func neuterNounForms(lemma string) map[string]string {
	if strings.HasSuffix(lemma, "ę") {
		return neuterNasalForms(lemma)
	}
	stem := lemma
	if strings.HasSuffix(lemma, "o") || strings.HasSuffix(lemma, "e") {
		stem = lemma[:len(lemma)-1]
	}
	_, lastSize := utf8.DecodeLastRuneInString(lemma)
	_ = lastSize
	loc := stem + "e"
	if strings.HasSuffix(stem, "k") || strings.HasSuffix(stem, "g") ||
		strings.HasSuffix(stem, "ch") || strings.HasSuffix(stem, "h") {
		loc = stem + "u"
	} else {
		loc = softenStem(stem) + "e"
	}
	nomPl := stem + "a"
	genPl := neutralizeCluster(stem)
	return map[string]string{
		"nom|sg": lemma,
		"gen|sg": stem + "a",
		"dat|sg": stem + "u",
		"acc|sg": lemma,
		"ins|sg": stem + "em",
		"loc|sg": loc,
		"voc|sg": lemma,
		"nom|pl": nomPl,
		"gen|pl": genPl,
		"dat|pl": stem + "om",
		"acc|pl": nomPl,
		"ins|pl": stem + "ami",
		"loc|pl": stem + "ach",
		"voc|pl": nomPl,
	}
}

func neuterNasalForms(lemma string) map[string]string {
	stem := strings.TrimSuffix(lemma, "ę")
	return map[string]string{
		"nom|sg": lemma,
		"gen|sg": stem + "ęcia",
		"dat|sg": stem + "ęciu",
		"acc|sg": lemma,
		"ins|sg": stem + "ęciem",
		"loc|sg": stem + "ęciu",
		"voc|sg": lemma,
		"nom|pl": stem + "ęta",
		"gen|pl": stem + "ąt",
		"dat|pl": stem + "ętom",
		"acc|pl": stem + "ęta",
		"ins|pl": stem + "ętami",
		"loc|pl": stem + "ętach",
		"voc|pl": stem + "ęta",
	}
}

func pluralNounForms(info WordInfo) map[string]string {
	stem := pluralStem(info.Lemma)
	gen := pluralGen(info.Lemma, stem)
	return map[string]string{
		"nom|pl": info.Lemma,
		"gen|pl": gen,
		"dat|pl": stem + "om",
		"acc|pl": info.Lemma,
		"ins|pl": stem + "ami",
		"loc|pl": stem + "ach",
		"voc|pl": info.Lemma,
	}
}

func pluralStem(lemma string) string {
	for _, suf := range []string{"e", "y", "i", "a", "o"} {
		if strings.HasSuffix(lemma, suf) {
			return strings.TrimSuffix(lemma, suf)
		}
	}
	return lemma
}

func pluralGen(lemma, stem string) string {
	switch {
	case strings.HasSuffix(lemma, "y"), strings.HasSuffix(lemma, "i"):
		return stem + "ów"
	case strings.HasSuffix(lemma, "a"):
		return stem
	}
	return stem
}

// neutralizeCluster inserts an "e" in a final consonant + k/g cluster, which
// is the most common Polish genitive-plural pattern (gruszka -> gruszek).
func neutralizeCluster(stem string) string {
	r, _ := utf8.DecodeLastRuneInString(stem)
	if r == 'k' || r == 'g' {
		if len(stem) >= 2 {
			prev, _ := utf8.DecodeLastRuneInString(stem[:len(stem)-1])
			if !isVowel(prev) {
				return stem[:len(stem)-1] + "e" + string(r)
			}
		}
	}
	return stem
}

// ---------------------------------------------------------------------------
// Adjectives
// ---------------------------------------------------------------------------

func adjForms(info WordInfo) map[string]string {
	lemma := info.Lemma
	var stem string
	if strings.HasSuffix(lemma, "y") || strings.HasSuffix(lemma, "i") {
		stem = lemma[:len(lemma)-1]
	} else {
		stem = lemma
	}
	soft := adjectiveSoft(lemma, stem)
	insSg := stem + "ym"
	locSg := stem + "ym"
	datPl := stem + "ym"
	genPl := stem + "ych"
	insPl := stem + "ymi"
	if soft {
		insSg = stem + "im"
		locSg = stem + "im"
		datPl = stem + "im"
		genPl = stem + "ich"
		insPl = stem + "imi"
	}
	neut := stem + "e"
	if soft {
		neut = stem + "ie"
	}
	forms := map[string]string{
		"sg|nom|m": lemma, "sg|gen|m": stem + "ego", "sg|dat|m": stem + "emu",
		"sg|acc|m": lemma, "sg|ins|m": insSg, "sg|loc|m": locSg, "sg|voc|m": lemma,
		"sg|nom|f": stem + "a", "sg|gen|f": stem + "ej", "sg|dat|f": stem + "ej",
		"sg|acc|f": stem + "ą", "sg|ins|f": stem + "ą", "sg|loc|f": stem + "ej",
		"sg|voc|f": stem + "a",
		"sg|nom|n": neut, "sg|gen|n": stem + "ego", "sg|dat|n": stem + "emu",
		"sg|acc|n": neut, "sg|ins|n": insSg, "sg|loc|n": locSg, "sg|voc|n": neut,
		"pl|nom|np": neut, "pl|gen|np": genPl, "pl|dat|np": datPl,
		"pl|acc|np": neut, "pl|ins|np": insPl, "pl|loc|np": genPl, "pl|voc|np": neut,
		"pl|nom|mp": stem + "i", "pl|gen|mp": genPl, "pl|dat|mp": datPl,
		"pl|acc|mp": genPl, "pl|ins|mp": insPl, "pl|loc|mp": genPl,
		"pl|voc|mp": stem + "i",
	}
	// The accusative of a masculine inanimate adjective is the same as the
	// nominative; the animate variant is the genitive.  We index both, which
	// is harmless because the input form disambiguates them.
	forms["sg|acc|ma"] = stem + "ego"
	forms["pl|acc|mp"] = genPl
	return forms
}

func adjectiveSoft(lemma, stem string) bool {
	if strings.HasSuffix(lemma, "ski") || strings.HasSuffix(lemma, "cki") ||
		strings.HasSuffix(lemma, "gi") || strings.HasSuffix(lemma, "ki") {
		return true
	}
	if strings.HasSuffix(lemma, "i") {
		return true
	}
	r, _ := utf8.DecodeLastRuneInString(stem)
	switch r {
	case 'ś', 'ć', 'ź', 'ń', 'j':
		return true
	}
	return false
}

// ---------------------------------------------------------------------------
// Verbs
// ---------------------------------------------------------------------------

type irregularVerb struct {
	pres   [6]string
	pastM  string
	pastF  string
	pastN  string
	pastPM string
	pastPN string
}

var irregularVerbs = map[string]irregularVerb{
	"być":  {[6]string{"jestem", "jesteś", "jest", "jesteśmy", "jesteście", "są"}, "był", "była", "było", "byli", "były"},
	"mieć": {[6]string{"mam", "masz", "ma", "mamy", "macie", "mają"}, "miał", "miała", "miało", "mieli", "miały"},
	"iść":  {[6]string{"idę", "idziesz", "idzie", "idziemy", "idziecie", "idą"}, "szedł", "szła", "szło", "szli", "szły"},
	"jeść": {[6]string{"jem", "jesz", "je", "jemy", "jecie", "jedzą"}, "jadł", "jadła", "jadło", "jedli", "jadły"},
	"spać": {[6]string{"śpię", "śpisz", "śpi", "śpimy", "śpicie", "śpią"}, "spał", "spała", "spało", "spali", "spały"},
	"chcieć": {[6]string{"chcę", "chcesz", "chce", "chcemy", "chcecie", "chcą"},
		"chciał", "chciała", "chciało", "chcieli", "chciały"},
	"móc": {[6]string{"mogę", "możesz", "może", "możemy", "możecie", "mogą"},
		"mógł", "mogła", "mogło", "mogli", "mogły"},
	"dać": {[6]string{"dam", "dasz", "da", "damy", "dacie", "dadzą"},
		"dał", "dała", "dało", "dali", "dały"},
	"brać": {[6]string{"biorę", "bierzesz", "bierze", "bierzemy", "bierzecie", "biorą"},
		"brał", "brała", "brało", "brali", "brały"},
	"pić": {[6]string{"piję", "pijesz", "pije", "pijemy", "pijecie", "piją"},
		"pił", "piła", "piło", "pili", "piły"},
	"widzieć": {[6]string{"widzę", "widzisz", "widzi", "widzimy", "widzicie", "widzą"},
		"widział", "widziała", "widziało", "widzieli", "widziały"},
	"wiedzieć": {[6]string{"wiem", "wiesz", "wie", "wiemy", "wiecie", "wiedzą"},
		"wiedział", "wiedziała", "wiedziało", "wiedzieli", "wiedziały"},
	"jechać": {[6]string{"jadę", "jedziesz", "jedzie", "jedziemy", "jedziecie", "jadą"},
		"jechał", "jechała", "jechało", "jechali", "jechały"},
	"wziąć": {[6]string{"wezmę", "weźmiesz", "weźmie", "weźmiemy", "weźmiecie", "wezmą"},
		"wziął", "wzięła", "wzięło", "wzięli", "wzięły"},
	"zacząć": {[6]string{"zacznę", "zaczniesz", "zacznie", "zaczniemy", "zaczniecie", "zaczną"},
		"zaczął", "zaczęła", "zaczęło", "zaczęli", "zaczęły"},
	"przyjść": {[6]string{"przyjdę", "przyjdziesz", "przyjdzie", "przyjdziemy", "przyjdziecie", "przyjdą"},
		"przyszedł", "przyszła", "przyszło", "przyszli", "przyszły"},
	"odejść": {[6]string{"odejdę", "odejdziesz", "odejdzie", "odejdziemy", "odejdziecie", "odejdą"},
		"odszedł", "odeszła", "odeszło", "odeszli", "odeszły"},
	"pójść": {[6]string{"pójdę", "pójdziesz", "pójdzie", "pójdziemy", "pójdziecie", "pójdą"},
		"poszedł", "poszła", "poszło", "poszli", "poszły"},
	"powiedzieć": {[6]string{"powiem", "powiesz", "powie", "powiemy", "powiecie", "powiedzą"},
		"powiedział", "powiedziała", "powiedziało", "powiedzieli", "powiedziały"},
	"myśleć": {[6]string{"myślę", "myślisz", "myśli", "myślimy", "myślicie", "myślą"},
		"myślał", "myślała", "myślało", "myśleli", "myślały"},
	"siedzieć": {[6]string{"siedzę", "siedzisz", "siedzi", "siedzimy", "siedzicie", "siedzą"},
		"siedział", "siedziała", "siedziało", "siedzieli", "siedziały"},
	"leżeć": {[6]string{"leżę", "leżysz", "leży", "leżymy", "leżycie", "leżą"},
		"leżał", "leżała", "leżało", "leżeli", "leżały"},
	"stać": {[6]string{"stoję", "stoisz", "stoi", "stoimy", "stoicie", "stoją"},
		"stał", "stała", "stało", "stali", "stały"},
	"bać": {[6]string{"boję", "boisz", "boi", "boimy", "boicie", "boją"},
		"bał", "bała", "bało", "bali", "bały"},
	"śmiać": {[6]string{"śmieję", "śmiejesz", "śmieje", "śmiejemy", "śmiejecie", "śmieją"},
		"śmiał", "śmiała", "śmiało", "śmieli", "śmiały"},
	"prać": {[6]string{"piorę", "pierzesz", "pierze", "pierzemy", "pierzecie", "piorą"},
		"prał", "prała", "prało", "prali", "prały"},
	"jeździć": {[6]string{"jeżdżę", "jeździsz", "jeździ", "jeździmy", "jeździcie", "jeżdżą"},
		"jeździł", "jeździła", "jeździło", "jeździli", "jeździły"},
	"chodzić": {[6]string{"chodzę", "chodzisz", "chodzi", "chodzimy", "chodzicie", "chodzą"},
		"chodził", "chodziła", "chodziło", "chodzili", "chodziły"},
	"słyszeć": {[6]string{"słyszę", "słyszysz", "słyszy", "słyszymy", "słyszycie", "słyszą"},
		"słyszał", "słyszała", "słyszało", "słyszeli", "słyszały"},
	"umieć": {[6]string{"umiem", "umiesz", "umie", "umiemy", "umiecie", "umieją"},
		"umiał", "umiała", "umiało", "umieli", "umiały"},
	"rozumieć": {[6]string{"rozumiem", "rozumiesz", "rozumie", "rozumiemy", "rozumiecie", "rozumieją"},
		"rozumiał", "rozumiała", "rozumiało", "rozumieli", "rozumiały"},
	"pisać": {[6]string{"piszę", "piszesz", "pisze", "piszemy", "piszecie", "piszą"},
		"pisał", "pisała", "pisało", "pisali", "pisały"},
	"myć": {[6]string{"myję", "myjesz", "myje", "myjemy", "myjecie", "myją"},
		"mył", "myła", "myło", "myli", "myły"},
	"dostać": {[6]string{"dostanę", "dostaniesz", "dostanie", "dostaniemy", "dostaniecie", "dostaną"},
		"dostał", "dostała", "dostało", "dostali", "dostały"},
	"dawać": {[6]string{"daję", "dajesz", "daje", "dajemy", "dajecie", "dają"},
		"dawał", "dawała", "dawało", "dawali", "dawały"},
	"znać": {[6]string{"znam", "znasz", "zna", "znamy", "znacie", "znają"},
		"znał", "znała", "znało", "znali", "znały"},
	"zapomnieć": {[6]string{"zapomnę", "zapomnisz", "zapomni", "zapomnimy", "zapomnicie", "zapomną"},
		"zapomniał", "zapomniała", "zapomniało", "zapomnieli", "zapomniały"},
}

func verbForms(info WordInfo) map[string]string {
	lemma := info.Lemma
	reflexive := false
	if strings.HasSuffix(lemma, " się") {
		lemma = strings.TrimSuffix(lemma, " się")
		reflexive = true
	}
	var forms map[string]string
	if irr, ok := irregularVerbs[lemma]; ok {
		forms = irregularVerbForms(lemma, irr)
	} else {
		forms = regularVerbForms(lemma)
	}
	if reflexive {
		for k, v := range forms {
			forms[k] = v + " się"
		}
	}
	return forms
}

func irregularVerbForms(lemma string, irr irregularVerb) map[string]string {
	forms := map[string]string{
		"inf":       lemma,
		"pres|1|sg": irr.pres[0], "pres|2|sg": irr.pres[1], "pres|3|sg": irr.pres[2],
		"pres|1|pl": irr.pres[3], "pres|2|pl": irr.pres[4], "pres|3|pl": irr.pres[5],
		"past|sg|m": irr.pastM, "past|sg|f": irr.pastF, "past|sg|n": irr.pastN,
		"past|pl|mp": irr.pastPM, "past|pl|np": irr.pastPN,
		"imp|2|sg": imperativeFromPres(irr.pres[1]),
	}
	return forms
}

func regularVerbForms(lemma string) map[string]string {
	forms := map[string]string{"inf": lemma}
	var pres [6]string
	var pastM string
	matched := true
	switch {
	case strings.HasSuffix(lemma, "ować"):
		stem := strings.TrimSuffix(lemma, "ować")
		p := stem + "uj"
		pres = [6]string{p + "ę", p + "esz", p + "e", p + "emy", p + "ecie", p + "ą"}
		pastM = stem + "ował"
	case strings.HasSuffix(lemma, "iwać"), strings.HasSuffix(lemma, "ywać"):
		stem := strings.TrimSuffix(lemma, "ać")
		pres = [6]string{stem + "am", stem + "asz", stem + "a", stem + "amy", stem + "acie", stem + "ają"}
		pastM = stem + "ał"
	case strings.HasSuffix(lemma, "ać"):
		stem := strings.TrimSuffix(lemma, "ać")
		pastM = strings.TrimSuffix(lemma, "ć") + "ł"
		if strings.HasSuffix(stem, "a") {
			stem = stem[:len(stem)-1]
		}
		pres = [6]string{stem + "am", stem + "asz", stem + "a", stem + "amy", stem + "acie", stem + "ają"}
	case strings.HasSuffix(lemma, "ić"):
		stem := strings.TrimSuffix(lemma, "ić")
		pres = [6]string{stem + "ię", stem + "isz", stem + "i", stem + "imy", stem + "icie", stem + "ą"}
		pastM = stem + "ił"
	case strings.HasSuffix(lemma, "yć"):
		stem := strings.TrimSuffix(lemma, "yć")
		pres = [6]string{stem + "ę", stem + "ysz", stem + "y", stem + "ymy", stem + "ycie", stem + "ą"}
		pastM = stem + "ył"
	case strings.HasSuffix(lemma, "eć"):
		stem := strings.TrimSuffix(lemma, "eć")
		if strings.HasSuffix(stem, "cz") || strings.HasSuffix(stem, "sz") ||
			strings.HasSuffix(stem, "ż") || strings.HasSuffix(stem, "rz") {
			pres = [6]string{stem + "ę", stem + "ysz", stem + "y", stem + "ymy", stem + "ycie", stem + "ą"}
		} else {
			pres = [6]string{stem + "ę", stem + "isz", stem + "i", stem + "imy", stem + "icie", stem + "ą"}
		}
		pastM = stem + "ał"
	case strings.HasSuffix(lemma, "nąć"):
		stem := strings.TrimSuffix(lemma, "nąć")
		pres = [6]string{stem + "nę", stem + "niesz", stem + "nie", stem + "niemy", stem + "niecie", stem + "ną"}
		pastM = stem + "nął"
	case strings.HasSuffix(lemma, "ąć"):
		stem := strings.TrimSuffix(lemma, "ąć")
		pres = [6]string{stem + "nę", stem + "niesz", stem + "nie", stem + "niemy", stem + "niecie", stem + "ną"}
		pastM = stem + "nął"
	default:
		matched = false
	}
	if !matched {
		return forms
	}
	forms["pres|1|sg"] = pres[0]
	forms["pres|2|sg"] = pres[1]
	forms["pres|3|sg"] = pres[2]
	forms["pres|1|pl"] = pres[3]
	forms["pres|2|pl"] = pres[4]
	forms["pres|3|pl"] = pres[5]
	forms["past|sg|m"] = pastM
	forms["past|sg|f"] = pastFeminine(pastM)
	forms["past|sg|n"] = pastNeuter(pastM)
	forms["past|pl|mp"] = pastPluralMP(pastM)
	forms["past|pl|np"] = pastPluralNP(pastM)
	forms["imp|2|sg"] = imperativeFromPres(pres[1])
	return forms
}

func pastFeminine(m string) string {
	if strings.HasSuffix(m, "ł") {
		return m[:len(m)-2] + "ła"
	}
	return m + "a"
}

func pastNeuter(m string) string {
	if strings.HasSuffix(m, "ł") {
		return m[:len(m)-2] + "ło"
	}
	return m + "o"
}

func pastPluralMP(m string) string {
	if strings.HasSuffix(m, "ł") {
		return m[:len(m)-2] + "li"
	}
	return m + "i"
}

func pastPluralNP(m string) string {
	if strings.HasSuffix(m, "ł") {
		return m[:len(m)-2] + "ły"
	}
	return m + "y"
}

func imperativeFromPres(pres2sg string) string {
	return pres2sg
}

// ---------------------------------------------------------------------------
// String helpers
// ---------------------------------------------------------------------------

func softenStem(stem string) string {
	if stem == "" {
		return stem
	}
	if strings.HasSuffix(stem, "ch") {
		return stem[:len(stem)-2] + "sz"
	}
	for _, suf := range []string{"sz", "cz", "rz", "dz"} {
		if strings.HasSuffix(stem, suf) {
			return stem
		}
	}
	r, size := utf8.DecodeLastRuneInString(stem)
	m := map[rune]string{
		't': "ci", 'd': "dzi", 'k': "c", 'g': "dz", 'r': "rz", 'ł': "l",
		'p': "pi", 'b': "bi", 'f': "fi", 'w': "wi", 'm': "mi", 'n': "ni",
		's': "si", 'z': "zi", 'c': "c", 'j': "j", 'l': "l", 'ń': "ń",
		'ś': "ś", 'ć': "ć", 'ź': "ź", 'ż': "ż",
	}
	if v, ok := m[r]; ok {
		return stem[:len(stem)-size] + v
	}
	return stem
}
