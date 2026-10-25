package main

import (
	"strings"
	"testing"
)

func testDictionary(t *testing.T) *Dictionary {
	t.Helper()
	entries := ParseDictionary(dictionaryMarkdown)
	if len(entries) != 3266 {
		t.Fatalf("expected 3266 dictionary entries, got %d", len(entries))
	}
	return BuildDictionary(entries)
}

func TestParseDictionary(t *testing.T) {
	entries := ParseDictionary(dictionaryMarkdown)
	if len(entries) != 3266 {
		t.Fatalf("expected 3266 entries, got %d", len(entries))
	}
	if entries[0].Dialect != "abo" || entries[0].Standard != "albo" {
		t.Fatalf("unexpected first entry: %+v", entries[0])
	}
}

func TestTranslateInflection(t *testing.T) {
	dict := testDictionary(t)
	opts := TranslateOptions{PreserveCase: true, MaxAlternatives: 3}
	cases := []struct {
		in   string
		want string
	}{
		// Singular case forms of ziemniak -> pyra.
		{"ziemniak", "pyra"},
		{"ziemniaka", "pyry"},
		{"ziemniakiem", "pyrą"},
		{"ziemniakowi", "pyrze"},
		{"ziemniakach", "pyrach"},
		{"ziemniaków", "pyr"},
		{"ziemniaki", "pyry"},
		// Other nouns.
		{"pies", "kejter"},
		{"psa", "kejtra"},
		{"chleb", "byb"},
		{"chleba", "byba"},
		{"wódka", "berbela"},
		{"wódkę", "berbelę"},
		{"wódką", "berbelą"},
		{"tramwaj", "bimba"},
		{"tramwajem", "bimbą"},
		{"rowerem", "kołem"},
		{"pieniędzy", "dudków"},
		{"okulary", "blyce"},
		// Verbs.
		{"spać", "garować"},
		{"śpi", "garuje"},
		{"pracować", "funktować"},
		{"pracuje", "funktuje"},
		{"pracują", "funktują"},
		// Unknown words are preserved.
		{"Poznań", "Poznań"},
	}
	for _, tc := range cases {
		got := dict.Translate(tc.in, opts).Translation
		if got != tc.want {
			t.Errorf("Translate(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestTranslateSentences(t *testing.T) {
	dict := testDictionary(t)
	opts := TranslateOptions{PreserveCase: true, MaxAlternatives: 3}
	cases := []struct {
		in   string
		want string
	}{
		{"Ziemniaki są w kuchni.", "Pyry są wew kuchni."},
		{"Mam ziemniaki.", "Mam pyry."},
		{"Nie ma ziemniaków.", "Nie ma pyr."},
		{"Widzę psa.", "Widzę kejtra."},
		{"On śpi.", "On garuje."},
		{"Jadę tramwajem.", "Jacham bimbą."},
		{"Piję wódkę.", "Piję berbelę."},
		{"Nie ma chleba.", "Nie ma byba."},
		{"Mówię o ziemniakach.", "Mówię łe pyrach."},
		{"Moje okulary są nowe.", "Moje blyce są nowe."},
		{"Oni pracują w sklepie.", "Oni funktują wew interesie."},
		{"To jest mój pies.", "To jest mój kejter."},
		{"Kocham Poznań.", "Kocham Poznań."},
		{"Nie mam pieniędzy.", "Nie mam dudków."},
	}
	for _, tc := range cases {
		got := dict.Translate(tc.in, opts).Translation
		if got != tc.want {
			t.Errorf("Translate(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestPreserveCase(t *testing.T) {
	dict := testDictionary(t)
	on := dict.Translate("Ziemniaki są smaczne.", TranslateOptions{PreserveCase: true})
	if on.Translation != "Pyry są smaczne." {
		t.Errorf("preserve case on: got %q", on.Translation)
	}
	off := dict.Translate("Ziemniaki są smaczne.", TranslateOptions{PreserveCase: false})
	if off.Translation != "pyry są smaczne." {
		t.Errorf("preserve case off: got %q", off.Translation)
	}
	upper := dict.Translate("ZIEMNIAKI", TranslateOptions{PreserveCase: true})
	if upper.Translation != "PYRY" {
		t.Errorf("upper case: got %q", upper.Translation)
	}
}

func TestAlternativesAndSegments(t *testing.T) {
	dict := testDictionary(t)
	res := dict.Translate("Ziemniaki.", TranslateOptions{PreserveCase: true, MaxAlternatives: 3})
	if res.Translation != "Pyry." {
		t.Fatalf("translation = %q", res.Translation)
	}
	if len(res.Segments) != 1 || !res.Segments[0].Matched {
		t.Fatalf("segments = %+v", res.Segments)
	}
	seg := res.Segments[0]
	if seg.Standard != "ziemniak" || seg.Dialect != "pyra" {
		t.Errorf("unexpected segment metadata: %+v", seg)
	}
	if len(seg.Alternatives) == 0 {
		t.Errorf("expected alternatives, got none")
	}
	found := false
	for _, alt := range seg.Alternatives {
		if strings.EqualFold(alt, "pyrki") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected pyrki among alternatives, got %v", seg.Alternatives)
	}
	if res.Stats.Words != 1 || res.Stats.TranslatedWords != 1 || res.Stats.Matches != 1 {
		t.Errorf("unexpected stats: %+v", res.Stats)
	}
}

func TestUnknownWordsPreserved(t *testing.T) {
	dict := testDictionary(t)
	in := "Kocham Poznań i Piotra."
	res := dict.Translate(in, TranslateOptions{PreserveCase: true})
	if res.Translation != in {
		t.Errorf("expected %q, got %q", in, res.Translation)
	}
	if res.Stats.TranslatedWords != 0 {
		t.Errorf("expected zero translated words, got %+v", res.Stats)
	}
}
