package main

import (
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

// TranslateOptions controls sentence translation.
type TranslateOptions struct {
	// PreserveCase keeps the capitalisation of the source word (Ziemniaki
	// -> Pyry, ZIEMNIAKI -> PYRY).  Defaults to true.
	PreserveCase bool
	// MaxAlternatives is the maximum number of alternatives returned per
	// matched segment (including the chosen translation).  Defaults to 5.
	MaxAlternatives int
}

// Segment describes one piece of the translated sentence.
type Segment struct {
	Source       string   `json:"source"`
	Translation  string   `json:"translation"`
	Matched      bool     `json:"matched"`
	Standard     string   `json:"standard,omitempty"`
	Dialect      string   `json:"dialect,omitempty"`
	Alternatives []string `json:"alternatives,omitempty"`
}

// Stats summarises a translation.
type Stats struct {
	Words             int `json:"words"`
	TranslatedWords   int `json:"translated_words"`
	UntranslatedWords int `json:"untranslated_words"`
	Matches           int `json:"matches"`
}

// TranslationResult is the JSON response of the translation endpoint.
type TranslationResult struct {
	Input       string    `json:"input"`
	Translation string    `json:"translation"`
	Segments    []Segment `json:"segments"`
	Stats       Stats     `json:"stats"`
}

var wordRe = regexp.MustCompile(`[\p{L}\p{N}]+(?:['’\-][\p{L}\p{N}]+)*`)

type wordSpan struct {
	raw        string
	norm       string
	start, end int
}

func findWords(text string) []wordSpan {
	idx := wordRe.FindAllStringIndex(text, -1)
	if len(idx) == 0 {
		return nil
	}
	spans := make([]wordSpan, 0, len(idx))
	for _, p := range idx {
		raw := text[p[0]:p[1]]
		spans = append(spans, wordSpan{raw: raw, norm: normalizeKey(raw), start: p[0], end: p[1]})
	}
	return spans
}

// Translate translates standard Polish text into Poznań dialect.  It uses a
// greedy longest-match strategy over the dictionary index, so multi-word
// entries win over single words.  Punctuation, whitespace and unknown words
// are preserved.
func (d *Dictionary) Translate(text string, opts TranslateOptions) TranslationResult {
	if opts.MaxAlternatives <= 0 {
		opts.MaxAlternatives = 5
	}
	if opts.MaxAlternatives > 20 {
		opts.MaxAlternatives = 20
	}

	spans := findWords(text)
	result := TranslationResult{Input: text}
	result.Stats.Words = len(spans)
	if len(spans) == 0 {
		result.Translation = text
		return result
	}

	var out strings.Builder
	out.Grow(len(text))
	last := 0
	i := 0
	for i < len(spans) {
		matched := false
		maxN := d.MaxPhraseWords
		if maxN > len(spans)-i {
			maxN = len(spans) - i
		}
		for n := maxN; n >= 1; n-- {
			if n > 1 {
				// Do not match phrases across punctuation.
				gapOK := true
				for k := 1; k < n; k++ {
					gap := text[spans[i+k-1].end:spans[i+k].start]
					if strings.TrimSpace(gap) != "" {
						gapOK = false
						break
					}
				}
				if !gapOK {
					continue
				}
			}
			key := spans[i].norm
			for k := 1; k < n; k++ {
				key += " " + spans[i+k].norm
			}
			list := d.Index[key]
			if len(list) == 0 {
				continue
			}
			best := list[0]
			target := best.Target
			if opts.PreserveCase {
				target = applyCase(spans[i].raw, target)
			}
			var alternatives []string
			if opts.MaxAlternatives > 1 {
				seen := map[string]bool{target: true}
				for _, c := range list[1:] {
					alt := c.Target
					if opts.PreserveCase {
						alt = applyCase(spans[i].raw, alt)
					}
					if seen[alt] {
						continue
					}
					seen[alt] = true
					alternatives = append(alternatives, alt)
					if len(alternatives) >= opts.MaxAlternatives-1 {
						break
					}
				}
			}
			out.WriteString(text[last:spans[i].start])
			out.WriteString(target)
			last = spans[i+n-1].end
			result.Segments = append(result.Segments, Segment{
				Source:       text[spans[i].start:spans[i+n-1].end],
				Translation:  target,
				Matched:      true,
				Standard:     best.StandardLemma,
				Dialect:      best.DialectLemma,
				Alternatives: alternatives,
			})
			result.Stats.TranslatedWords += n
			result.Stats.Matches++
			i += n
			matched = true
			break
		}
		if !matched {
			result.Segments = append(result.Segments, Segment{
				Source:      spans[i].raw,
				Translation: spans[i].raw,
				Matched:     false,
			})
			i++
		}
	}
	out.WriteString(text[last:])
	result.Translation = out.String()
	result.Stats.UntranslatedWords = result.Stats.Words - result.Stats.TranslatedWords
	return result
}

// applyCase copies the capitalisation of the source word onto the target
// phrase: all-caps stays all-caps, a leading capital stays a leading
// capital, everything else is left alone.
func applyCase(source, target string) string {
	if source == "" || target == "" {
		return target
	}
	if isAllUpper(source) {
		return strings.ToUpper(target)
	}
	r, _ := utf8.DecodeRuneInString(source)
	if unicode.IsUpper(r) {
		return capitalizeFirst(target)
	}
	return target
}

func capitalizeFirst(s string) string {
	r, size := utf8.DecodeRuneInString(s)
	if r == utf8.RuneError && size == 0 {
		return s
	}
	return string(unicode.ToUpper(r)) + s[size:]
}

func isAllUpper(s string) bool {
	hasLetter := false
	for _, r := range s {
		if unicode.IsLetter(r) {
			hasLetter = true
			if !unicode.IsUpper(r) {
				return false
			}
		}
	}
	return hasLetter
}
