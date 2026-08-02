package recommender

import (
	"fmt"
	"sort"
	"strings"
)

// language is the output language of the human-readable recommendation text. It
// is detected from the task text alone and is internal: it is never a JSON
// field, never a CLI flag, and never part of a normalized identifier.
//
// English is the zero value, so a Recommendation built without detection - by a
// caller outside this package, or by a test - renders in English, which is also
// the required fallback for every uncertain case.
type language int

const (
	languageEnglish language = iota
	languageGerman
)

func (l language) String() string {
	switch l {
	case languageGerman:
		return "German"
	default:
		return "English"
	}
}

// localizedText is one human-readable sentence written in both supported
// languages. Both spellings are written at the same place so that adding a
// sentence without translating it is visible at the call site and detectable by
// a completeness test, rather than degrading silently into English at runtime.
type localizedText struct {
	english string
	german  string
}

// in returns the sentence in lang. It panics rather than falling back to
// English, because an untranslated sentence must fail loudly instead of hiding
// drift in otherwise German output.
func (t localizedText) in(lang language) string {
	text := t.english
	if lang == languageGerman {
		text = t.german
	}
	if text == "" {
		panic(fmt.Sprintf("recommender: missing %s text for %q", lang, t.english))
	}
	return text
}

// germanMarkerThreshold is the number of distinct German-only words a task text
// must carry before the output switches to German.
//
// Detection is deliberately biased toward English: a single German loanword in
// an English prompt must not flip an English speaker's output, so one word is
// never enough. Evidence is counted in words rather than in vocabulary entries,
// and each word of the text is counted once, so neither a repeated word nor two
// entries spelling the same word can reach the threshold on their own.
const germanMarkerThreshold = 2

// germanFunctionWordMarkers are high-frequency German function words that have
// no ordinary English reading, so their presence is evidence about the language
// of the prompt rather than about its subject.
//
// Three groups are deliberately absent. Cross-language homographs - "die",
// "man", "was", "so", "in", "hat", "bin", "war", "also", "den" - occur in
// ordinary English text and would manufacture evidence out of English prompts.
// Personal pronouns are absent because short borrowed phrases carry them:
// "review the code für mich" is an English request and must stay English, which
// it only does while "mich" is not a marker.
//
// The third group is the identifiers and acronyms of English technical prose -
// "der" and "des" for DER and DES encodings, "das" for DAS storage, "ist" for
// an IST timestamp, "mit" for the MIT license, "aus" for the Australia country
// code, and "wo" for work orders. Detection is case-insensitive by requirement,
// so capitalization cannot disambiguate these spellings. Curating them out is
// what keeps the threshold honest: every registered marker counts, rather than
// some counting less than others depending on how a prompt is capitalized. The
// list is drawn from collisions English prompts actually carry and must grow
// when a new one shows up.
var germanFunctionWordMarkers = []string{
	"ein", "eine", "einen", "einem", "einer", "eines",
	"kein", "keine", "keinen", "diese", "dieser", "dieses", "diesen",
	"jede", "jeden", "jeder", "alle", "allen", "viele", "mehrere", "mehr",
	"und", "oder", "aber", "sowie", "weil", "damit", "dass", "wenn", "dann",
	"noch", "schon", "sehr", "auch", "bereits", "wieder", "außerdem", "zusätzlich",
	"deshalb", "jeweils", "bitte", "danke", "hier", "dort", "jetzt", "immer",
	"für", "ohne", "gegen", "über", "unter", "vor", "nach", "bei", "beim",
	"zum", "zur", "vom", "seit", "während", "zwischen", "durch",
	"nicht", "soll", "sollen", "sollte", "muss", "müssen", "kann", "können",
	"könnte", "darf", "dürfen", "wird", "werden", "wurde", "wurden",
	"sind", "waren", "haben", "habe", "hatte", "hatten",
	"möchte", "wollen", "brauche", "brauchen",
	"wie", "warum", "weshalb", "wieso", "welche", "welcher", "welches",
}

// englishSignalWords is every word used by an English signal term, split on the
// same word boundaries the matcher uses. It is the vocabulary an English prompt
// is expected to contain, and it is derived from the classifier so that it grows
// with it.
var englishSignalWords = buildEnglishSignalWords()

func buildEnglishSignalWords() map[string]bool {
	words := make(map[string]bool)
	for _, name := range sortedSignalFamilyNames() {
		for _, term := range allSignalFamilies[name].english {
			for _, word := range splitWords(strings.ToLower(term)) {
				words[word] = true
			}
		}
	}
	return words
}

// readsAsEnglish reports whether every word of a term is also an English signal
// word. German technical writing borrows English vocabulary wholesale, so a
// German signal list carries compounds such as "software-design" and
// "pull-request-review" that an English prompt spells exactly the same way.
// Those terms classify a task correctly in either language but say nothing about
// which language it is written in, so they must not count as language evidence.
//
// Deciding this against the classifier's own English vocabulary rather than a
// hand-kept list is what keeps the rule current: a borrowed compound added to a
// German list is filtered out the moment its parts are English signal terms.
func readsAsEnglish(term string) bool {
	words := splitWords(term)
	if len(words) == 0 {
		return false
	}
	for _, word := range words {
		if !readsAsEnglishWord(word) {
			return false
		}
	}
	return true
}

// readsAsEnglishWord also recognizes closed compounds made entirely from
// English signal words. German writes compounds such as "Systemdesign" without
// separators, but the same spelling is a common English code identifier such as
// SystemDesign and therefore is not German-only evidence.
func readsAsEnglishWord(word string) bool {
	if englishSignalWords[word] {
		return true
	}
	for _, r := range word {
		if r > 127 {
			return false
		}
	}

	parts := make([]int, len(word)+1)
	for i := range parts {
		parts[i] = -1
	}
	parts[0] = 0
	for end := 2; end <= len(word); end++ {
		for start := 0; start <= end-2; start++ {
			if parts[start] < 0 || !englishSignalWords[word[start:end]] {
				continue
			}
			if candidate := parts[start] + 1; candidate > parts[end] {
				parts[end] = candidate
			}
		}
	}
	return parts[len(word)] >= 2
}

func splitWords(term string) []string {
	return strings.FieldsFunc(term, func(r rune) bool { return !isWordRune(r) })
}

// germanRequestMarkers are the everyday verbs and nouns of a German request that
// no classifier signal family carries, because they describe how the developer
// asks rather than what the task is. Without them a short, ordinary prompt has
// only its articles to go on: "Übersetze einen Text" and "Erstelle eine Tabelle"
// reach a single marker each and fall back to English.
//
// Every term here is German-only. Words that German and English spell alike are
// deliberately absent even where the German reading is the common one in
// prompts - "plane", "liste"n, "lade", "laden", "gib", "wort" - because an
// English prompt containing two of them would flip the output language.
var germanRequestMarkers = []string{
	"erstelle", "erstellen", "erstellung", "erzeuge", "erzeugen", "generiere", "generieren",
	"schreibe", "schreiben", "formuliere", "formulieren",
	"übersetze", "übersetzen", "übersetzung",
	"erkläre", "erklären", "erklärung", "beschreibe", "beschreiben", "beschreibung",
	"zeig", "zeige", "zeigen", "nenne", "nennen",
	"mach", "mache", "machen", "finde", "finden", "suche", "suchen",
	"lese", "lesen", "sende", "senden", "speichere", "speichern",
	"starte", "starten", "beende", "beenden",
	"planen", "planung", "entwirf", "entwerfen", "entwurf",
	"berechne", "berechnen", "berechnung", "konvertiere", "konvertieren", "umwandeln",
	"verwende", "verwenden", "nutze", "nutzen", "benötige", "benötigen",
	"entscheide", "entscheiden", "entscheidung",
	"verbessere", "verbessern", "verbesserung", "vereinfache", "vereinfachen",
	"strukturiere", "strukturieren", "gliedere", "gliedern",
	"überlege", "überlegen", "vorschlagen", "hilf", "helfen", "hilfe",

	"tabelle", "tabellen", "liste", "beispiel", "beispiele",
	"vorschlag", "vorschläge", "frage", "fragen", "antwort", "antworten",
	"aufgabe", "aufgaben", "bericht", "berichte", "notiz", "notizen",
	"absatz", "satz", "sätze", "wörter", "seite", "seiten",
	"diagramm", "diagramme", "grafik", "grafiken", "vorlage", "vorlagen",
	"lösung", "lösungen", "schritt", "schritte", "möglichkeit", "möglichkeiten",
}

// crossLanguageMarkerExclusions are terms that read as ordinary English but that
// readsAsEnglish cannot recognize, because the English signal lists happen to
// carry a different form of the word ("code review", not "code-reviews";
// "refactor", not "refactoring") or none at all. Terms belong here only when the
// structural rule misses them.
var crossLanguageMarkerExclusions = map[string]bool{
	"aus":          true,
	"code-reviews": true,
	"codereview":   true,
	"refactoring":  true,
	"standard":     true,
	"wo":           true,
}

// germanMarker is one vocabulary entry together with the evidence it carries.
//
// A term is always matched whole, but it is weighed by its German-only words:
// "viele dateien" is two German words and therefore two pieces of evidence,
// while "code prüfen" is one German word next to a word English spells the same.
// Weighing entries rather than counting them is what lets a phrase claim the
// text it covers - so that "prüfen" cannot count the same word again - without
// the phrase hiding the German words inside it.
type germanMarker struct {
	term     string
	evidence int
}

// germanMarkers is the deduplicated marker vocabulary: the function words and
// request vocabulary above plus every German classifier signal term, minus the
// terms that read as English. Reusing the classifier's German vocabulary keeps
// the two in step automatically, so a German term added for classification also
// becomes evidence for the output language.
//
// It is ordered longest term first, so that the most specific entry covering a
// piece of text is the one that claims it.
var germanMarkers = buildGermanMarkers()

func buildGermanMarkers() []germanMarker {
	seen := make(map[string]bool)
	markers := make([]germanMarker, 0, len(germanFunctionWordMarkers)+len(germanRequestMarkers))

	add := func(terms []string) {
		for _, term := range terms {
			term = strings.ToLower(strings.TrimSpace(term))
			if term == "" || seen[term] || crossLanguageMarkerExclusions[term] || readsAsEnglish(term) {
				continue
			}
			seen[term] = true
			markers = append(markers, germanMarker{term: term, evidence: germanOnlyWordCount(term)})
		}
	}

	add(germanFunctionWordMarkers)
	add(germanRequestMarkers)
	for _, name := range sortedSignalFamilyNames() {
		add(allSignalFamilies[name].german)
	}

	sort.Slice(markers, func(i, j int) bool {
		if len(markers[i].term) != len(markers[j].term) {
			return len(markers[i].term) > len(markers[j].term)
		}
		return markers[i].term < markers[j].term
	})
	return markers
}

// germanOnlyWordCount reports how many of a term's words are not also English
// signal vocabulary. Every registered term has at least one, because a term
// written entirely from English signal words never becomes a marker.
func germanOnlyWordCount(term string) int {
	count := 0
	for _, word := range splitWords(term) {
		if !readsAsEnglishWord(word) {
			count++
		}
	}
	return count
}

// detectLanguage reports the output language for a task description. It reads
// the task text only, never the flags, and never influences classification or
// selection: a German prompt and its English equivalent reach the same model
// and the same effort level.
//
// Matching is case-insensitive, so "Fehler beheben", "FEHLER beheben", and
// "fehler beheben" are one and the same request. German needs
// germanMarkerThreshold German-only words; every other text is English.
func detectLanguage(task string) language {
	text := strings.ToLower(task)

	var claimed []textSpan
	evidence := 0
	for _, marker := range germanMarkers {
		occurrence, ok := unclaimedFirstOccurrence(text, marker.term, claimed)
		if !ok {
			continue
		}

		claimed = append(claimed, occurrence)
		evidence += marker.evidence
		if evidence >= germanMarkerThreshold {
			return languageGerman
		}
	}
	return languageEnglish
}

// unclaimedFirstOccurrence returns term's first whole-word occurrence, unless a
// term counted earlier already covers that text.
//
// Every term is weighed once, where it first appears, so that a repeated word or
// a repeated phrase is one piece of evidence however often it is written. The
// occurrence must also be its own: "code prüfen" and "prüfen" are two vocabulary
// entries over one German word, and counting both would let a single German
// token in an English prompt reach the threshold. Since terms are weighed
// longest first, the phrase claims the text and its parts add nothing.
func unclaimedFirstOccurrence(text, term string, claimed []textSpan) (textSpan, bool) {
	var first textSpan
	found := false
	visitTermOccurrences(text, term, func(occurrence textSpan) bool {
		first, found = occurrence, true
		return false
	})
	if !found {
		return textSpan{}, false
	}

	for _, span := range claimed {
		if first.overlaps(span) {
			return textSpan{}, false
		}
	}
	return first, true
}
