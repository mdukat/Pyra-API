package main

import (
	"regexp"
	"strings"
)

// sense is one candidate standard Polish meaning extracted from a dictionary
// gloss, together with a confidence score.
type sense struct {
	text  string
	score int
}

var (
	parenRe    = regexp.MustCompile(`\([^)]*\)`)
	bracketRe  = regexp.MustCompile(`\[[^\]]*\]`)
	senseSplit = regexp.MustCompile(`\s*(?:;|,|\blub\b|\balbo\b)\s*`)
	spacesRe   = regexp.MustCompile(`\s+`)
	wordOnlyRe = regexp.MustCompile(`^[a-ząćęłńóśźż]+(?:[-'][a-ząćęłńóśźż]+)*$`)
)

// senseQualifiers are descriptive lead-ins that never belong to a synonym.
var senseQualifiers = []string{
	"rodzaj ", "gatunek ", "określenie ", "nazwa ", "wyraz ", "wyrażenie ",
	"zwrot ", "forma ", "typ ", "w dawnym ", "w dawnych ", "w języku ",
	"w gwarze ", "w okresie ", "w czasie ", "jako ", "dawniej ", "obecnie ",
	"dziś ", "zwłaszcza ", "najczęściej ", "zwykle ", "przeważnie ",
	"głównie ", "bardzo ", "zupełnie ", "całkiem ", "także ", "też ",
	"już ", "jeszcze ", "tylko ", "np. ", "itp. ", "itd. ",
}

// descriptiveStart holds words that begin a description rather than a
// synonym.  Short phrases beginning with one of them are not indexed as
// inflectable phrases.
var descriptiveStart = setOf(
	"ktoś", "coś", "osoba", "człowiek", "mężczyzna", "kobieta", "dziecko",
	"chłopak", "dziewczyna", "chłopiec", "rzecz", "przedmiot", "określenie",
	"nazwa", "wyraz", "wyrażenie", "zwrot", "forma", "rodzaj", "gatunek",
	"część", "element", "miejsce", "stan", "czynność", "ubranie", "ubrudzony",
	"dawniej", "obecnie", "dziś", "także", "też", "zwłaszcza", "najczęściej",
	"zwykle", "bardzo", "zupełnie", "całkiem", "w", "z", "do", "na", "o",
	"od", "po", "przy", "bez", "jako", "który", "która", "które", "kiedy",
	"gdy", "że", "aby", "żeby", "jeśli", "jeżeli", "gdyż", "bo", "ponieważ",
	"taki", "taka", "takie", "ten", "ta", "te", "jakiś", "jakaś", "jakieś",
	"pewien", "pewna", "pewne",
)

var descriptiveWord = setOf(
	"się", "itp", "itd", "np", "tj", "zwłaszcza", "najczęściej", "zwykle",
	"przeważnie", "głównie", "dawniej", "obecnie", "dziś", "także", "też",
	"bardzo", "zupełnie", "całkiem", "który", "która", "które",
)

var personSingles = setOf(
	"kobieta", "mężczyzna", "dziewczyna", "chłopak", "dziecko", "człowiek",
	"osoba", "chłopiec", "kolega", "koleżanka", "pan", "pani",
)

// extractSenses turns a raw dictionary gloss into a list of candidate
// standard Polish terms.  The heuristics favour exact single words and
// short phrases, and merely index long descriptions as exact phrases.
func extractSenses(gloss string) []sense {
	g := parenRe.ReplaceAllString(gloss, " ")
	g = bracketRe.ReplaceAllString(g, " ")

	parts := senseSplit.Split(g, -1)
	cleaned := make([]string, 0, len(parts))
	for _, p := range parts {
		p = cleanSense(p)
		if p != "" {
			cleaned = append(cleaned, p)
		}
	}

	// Deduplicate while preserving order.
	uniq := cleaned[:0]
	seenPart := make(map[string]bool, len(cleaned))
	for _, p := range cleaned {
		if !seenPart[p] {
			seenPart[p] = true
			uniq = append(uniq, p)
		}
	}
	cleaned = uniq

	good := 0
	for _, p := range cleaned {
		if isCleanSingle(p) || isShortPhrase(p) {
			good++
		}
	}

	var out []sense
	seen := make(map[string]bool)
	add := func(t string, s int) {
		t = cleanSense(t)
		if t == "" || seen[t] {
			return
		}
		seen[t] = true
		out = append(out, sense{text: t, score: s})
	}

	for _, p := range cleaned {
		switch {
		case isCleanSingle(p):
			switch {
			case len(cleaned) == 1:
				add(p, 100)
			case isStopSingle(p):
				add(p, 15)
			case personSingles[p] && good < 2:
				// A single person noun in a descriptive list is often
				// just the subject of the description, not a synonym.
				add(p, 20)
			case good >= 2:
				add(p, 60)
			default:
				add(p, 35)
			}
		case isShortPhrase(p):
			switch {
			case len(cleaned) == 1:
				add(p, 80)
			case good >= 2:
				add(p, 45)
			default:
				add(p, 25)
			}
		default:
			if len(strings.Fields(p)) <= 8 {
				add(p, 5)
			}
		}
	}

	full := cleanSense(g)
	if full != "" && len(strings.Fields(full)) <= 8 {
		add(full, 10)
	}
	return out
}

func cleanSense(s string) string {
	s = spacesRe.ReplaceAllString(strings.TrimSpace(s), " ")
	s = strings.Trim(s, " .,;:!?\"'()[]…–—-")
	s = strings.TrimSpace(s)
	s = strings.ToLower(s)
	for strings.HasPrefix(s, "...") {
		s = strings.TrimSpace(strings.TrimPrefix(s, "..."))
	}
	for changed := true; changed; {
		changed = false
		for _, q := range senseQualifiers {
			if strings.HasPrefix(s, q) {
				s = strings.TrimSpace(s[len(q):])
				changed = true
				break
			}
		}
	}
	if i := strings.LastIndex(s, ":"); i >= 0 {
		tail := strings.TrimSpace(s[i+1:])
		if tail != "" && len(strings.Fields(tail)) <= 4 {
			s = tail
		}
	}
	return strings.TrimSpace(s)
}

func isCleanSingle(s string) bool {
	return !strings.Contains(s, " ") && wordOnlyRe.MatchString(s)
}

func isShortPhrase(s string) bool {
	fields := strings.Fields(s)
	if len(fields) < 2 || len(fields) > 4 {
		return false
	}
	for _, f := range fields {
		if f == "się" {
			continue
		}
		if !wordOnlyRe.MatchString(f) {
			return false
		}
	}
	if descriptiveStart[fields[0]] {
		return false
	}
	for _, f := range fields {
		if descriptiveWord[f] {
			return false
		}
	}
	return true
}

func isStopSingle(s string) bool {
	if functionWords[s] {
		return true
	}
	switch s {
	case "być", "mieć", "ktoś", "coś", "osoba", "człowiek", "rzecz",
		"przedmiot", "itp", "itd", "np", "tj",
		"no", "ależ", "otóż", "ano", "jeno", "ino", "to", "ta", "te",
		"ten", "ci", "ów", "owa", "owo", "jakiś", "jakaś", "jakieś",
		"każdy", "każda", "każde", "wszyscy", "wszystko", "nic",
		"nikt", "gdzieś", "kiedyś", "jakoś", "tutaj", "teraz", "potem":
		return true
	}
	return false
}
