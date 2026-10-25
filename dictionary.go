package main

import (
	"bufio"
	"bytes"
	"sort"
	"strings"
)

// Entry is a single row of the Poznań dialect dictionary.  Dialect is the
// left column (Poznań dialect headword), Standard is the right column
// (standard Polish meaning).
type Entry struct {
	Dialect  string
	Standard string
}

// Candidate is one possible translation of a standard Polish surface form.
// Tag describes the grammatical form that was matched (nom|sg, pres|3|sg,
// ...) and is used only for diagnostics / ordering.  Target is the dialect
// surface form to emit.
type Candidate struct {
	Target        string
	StandardLemma string
	DialectLemma  string
	Tag           string
	Score         int
}

// Dictionary holds the parsed entries and the reverse index used by the
// translator.  Index maps a normalised standard Polish word/phrase (lower
// case, single spaces) to a list of candidate translations, sorted best
// first.
type Dictionary struct {
	Entries        []Entry
	Index          map[string][]Candidate
	MaxPhraseWords int

	seen map[string]map[string]int
}

// normalizeKey lower-cases text, normalises whitespace and trims it.  It is
// used both when indexing and when looking words up, so that "Ziemniaki,"
// and "  ziemniaki " resolve to the same key.
func normalizeKey(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, "\u00a0", " ")
	return strings.Join(strings.Fields(s), " ")
}

// ParseDictionary reads the markdown export of the Poznań dictionary and
// returns its entries.  Only table rows with two columns are considered.
func ParseDictionary(data []byte) []Entry {
	var entries []Entry
	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "| ") {
			continue
		}
		if strings.Contains(line, "Gwara poznańska") || strings.HasPrefix(line, "| ---") {
			continue
		}
		parts := strings.Split(line, "|")
		if len(parts) < 4 {
			continue
		}
		dialect := strings.TrimSpace(parts[1])
		standard := strings.TrimSpace(parts[2])
		if dialect == "" || standard == "" {
			continue
		}
		entries = append(entries, Entry{Dialect: dialect, Standard: standard})
	}
	return entries
}

// BuildDictionary indexes every sense extracted from every dictionary entry.
func BuildDictionary(entries []Entry) *Dictionary {
	d := &Dictionary{
		Entries:        entries,
		Index:          make(map[string][]Candidate),
		MaxPhraseWords: 1,
		seen:           make(map[string]map[string]int),
	}
	for _, e := range entries {
		for _, s := range extractSenses(e.Standard) {
			d.addMapping(s.text, e.Dialect, s.score)
		}
	}
	for key, list := range d.Index {
		sort.SliceStable(list, func(i, j int) bool {
			if list[i].Score != list[j].Score {
				return list[i].Score > list[j].Score
			}
			pi, pj := tagPriority(list[i].Tag), tagPriority(list[j].Tag)
			if pi != pj {
				return pi > pj
			}
			// Equal score and tag: keep insertion (dictionary) order so the
			// first headword listed in the source dictionary wins.
			return false
		})
		d.Index[key] = list
	}
	return d
}

// tagPriority is a tie-breaker for candidates with the same score.  Base
// forms and nominatives win over oblique cases so that an ambiguous standard
// form such as "ziemniak" (nominative and accusative of an inanimate noun)
// translates to "pyra" rather than "pyrę".
func tagPriority(tag string) int {
	if tag == "base" {
		return 10
	}
	priority := 1
	bump := func(v int) {
		if v > priority {
			priority = v
		}
	}
	for _, part := range strings.Split(tag, "|") {
		switch part {
		case "inf":
			bump(9)
		case "nom":
			bump(8)
		case "acc":
			bump(7)
		case "gen":
			bump(6)
		case "dat":
			bump(5)
		case "ins":
			bump(4)
		case "loc":
			bump(3)
		case "voc":
			bump(2)
		case "pres":
			bump(6)
		case "past":
			bump(5)
		case "imp":
			bump(4)
		}
	}
	return priority
}

// addMapping indexes one standard->dialect pair.  baseScore is the
// confidence of the extracted sense.  Inflected forms are paired by
// grammatical tag; when the two words have different parts of speech the
// inflected standard forms still map to the dialect base form so that e.g.
// "herbatę" can translate to a dialect phrase.
func (d *Dictionary) addMapping(stdText, dialectText string, baseScore int) {
	std := normalizeKey(stdText)
	dia := normalizeKey(dialectText)
	if std == "" || dia == "" {
		return
	}
	sm := analyzeWord(std)
	dm := analyzeWord(dia)

	d.addCandidate(std, Candidate{
		Target:        dia,
		StandardLemma: sm.Lemma,
		DialectLemma:  dm.Lemma,
		Tag:           "base",
		Score:         baseScore + 1000,
	})

	if baseScore < 40 {
		// Low-confidence senses (single person nouns or function words
		// scraped out of a longer description) are only indexed as exact
		// phrases.  Generating forms from them creates bogus matches such
		// as the locative of "no" being "nie".
		return
	}

	if sm.POS == dm.POS && sm.POS != "other" {
		sf := generateForms(sm)
		df := generateForms(dm)
		for key, sform := range sf {
			dform, ok := df[key]
			if !ok || sform == "" || dform == "" {
				continue
			}
			d.addCandidate(normalizeKey(sform), Candidate{
				Target:        dform,
				StandardLemma: sm.Lemma,
				DialectLemma:  dm.Lemma,
				Tag:           key,
				Score:         baseScore + 500,
			})
		}
		return
	}

	if sm.POS != "other" {
		for _, sform := range generateForms(sm) {
			if sform == "" {
				continue
			}
			d.addCandidate(normalizeKey(sform), Candidate{
				Target:        dia,
				StandardLemma: sm.Lemma,
				DialectLemma:  dm.Lemma,
				Tag:           "base",
				Score:         baseScore + 400,
			})
		}
	}
}

func (d *Dictionary) addCandidate(key string, c Candidate) {
	if key == "" || c.Target == "" {
		return
	}
	if n := len(strings.Fields(key)); n > d.MaxPhraseWords {
		d.MaxPhraseWords = n
	}
	sig := c.Target + "\x00" + c.Tag + "\x00" + c.DialectLemma
	if m, ok := d.seen[key]; ok {
		if idx, ok := m[sig]; ok {
			if c.Score > d.Index[key][idx].Score {
				d.Index[key][idx] = c
			}
			return
		}
		m[sig] = len(d.Index[key])
	} else {
		d.seen[key] = map[string]int{sig: 0}
	}
	d.Index[key] = append(d.Index[key], c)
}

// Lookup returns the candidates for an already normalised key.
func (d *Dictionary) Lookup(key string) []Candidate {
	return d.Index[normalizeKey(key)]
}
