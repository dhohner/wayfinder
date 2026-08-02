package recommender

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

var allLanguages = []language{languageEnglish, languageGerman}

const (
	germanCodingTask  = "implementiere einen kleinen Go-API-Endpunkt"
	englishCodingTask = "implement a small Go API endpoint"
)

func TestGermanPromptIsAnsweredInGerman(t *testing.T) {
	rec := RecommendWithOptimization(germanCodingTask, OptimizeValue)

	if rec.Language != languageGerman {
		t.Fatalf("expected %q to be detected as German, got %s", germanCodingTask, rec.Language)
	}

	out := Format(rec)
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected three lines, got %d in %q", len(lines), out)
	}
	if !strings.HasPrefix(lines[0], "Modell: ") || !strings.HasPrefix(lines[1], "Reasoning: ") || !strings.HasPrefix(lines[2], "Begründung: ") {
		t.Fatalf("expected German labels Modell, Reasoning, and Begründung, got:\n%s", out)
	}
	assertNotContainsAny(t, out, "Model: ", "Reason: ")
	assertReasonLanguage(t, rec.Reason, languageGerman)
}

func TestEnglishPromptIsAnsweredInEnglish(t *testing.T) {
	rec := RecommendWithOptimization(englishCodingTask, OptimizeValue)

	if rec.Language != languageEnglish {
		t.Fatalf("expected %q to be detected as English, got %s", englishCodingTask, rec.Language)
	}

	out := Format(rec)
	assertContainsAll(t, out, "Model: ", "Reasoning: ", "Reason: ")
	assertNotContainsAny(t, out, "Modell:", "Begründung:")
	assertReasonLanguage(t, rec.Reason, languageEnglish)
}

// TestMixedPromptWithoutClearGermanEvidenceStaysEnglish covers the point of the
// English bias: a borrowed German word inside an English request is not evidence
// that the developer wants a German answer.
func TestMixedPromptWithoutClearGermanEvidenceStaysEnglish(t *testing.T) {
	rec := RecommendWithOptimization("review the code für mich", OptimizeValue)

	if rec.Language != languageEnglish {
		t.Fatalf("expected a single German word to leave the output in English, got %s", rec.Language)
	}
	assertContainsAll(t, Format(rec), "Model: ", "Reason: ")
	assertReasonLanguage(t, rec.Reason, languageEnglish)
}

func TestDetectLanguageRequiresTwoDistinctGermanMarkers(t *testing.T) {
	cases := []struct {
		name string
		task string
		want language
	}{
		{"no marker", englishCodingTask, languageEnglish},
		{"one marker", "review the code für mich", languageEnglish},
		{"one marker repeated three times", "review the code für, für, für", languageEnglish},
		{"two markers", "Fehler beheben", languageGerman},
		{"three markers", germanCodingTask, languageGerman},
		{"cross-language homographs only", "die man was so in hat bin", languageEnglish},
		{"homograph plus one marker", "die man was für ein bug", languageGerman},
		{"borrowed english technical terms only", "refactor the module with a design-system standard", languageEnglish},
		{"borrowed english compounds only", "update the pull-request-review workflow for the software-design module", languageEnglish},
		{"acronyms spelling german function words", "compare DER and DES encodings", languageEnglish},
		{"acronyms mixed into an english sentence", "check that the MIT license header IST valid", languageEnglish},
		{"technical identifiers spelling german function words", "support AUS tax rules in the WO service", languageEnglish},
		{"german prose whose function words collide with acronyms", "Der Fehler ist noch offen", languageGerman},
		{"overlapping vocabulary entries over one german word", "review code prüfen", languageEnglish},
		{"one entry spelling two german words", "Bearbeite viele Dateien", languageGerman},
		{"empty task", "", languageEnglish},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := detectLanguage(tc.task); got != tc.want {
				t.Fatalf("detectLanguage(%q) = %s, want %s", tc.task, got, tc.want)
			}
		})
	}
}

// TestDetectLanguageIsCaseInsensitive pins the contract that capitalized German
// nouns, which is how they are actually written, count as markers.
func TestDetectLanguageIsCaseInsensitive(t *testing.T) {
	if got := detectLanguage("Fehler Beheben"); got != languageGerman {
		t.Fatalf("detectLanguage on capitalized German = %s, want German", got)
	}
}

// TestAcronymsAreNotGermanMarkers covers the acronym collision end to end: an
// English prompt whose acronyms happen to spell German function words must not
// switch the output language.
func TestAcronymsAreNotGermanMarkers(t *testing.T) {
	tasks := []string{
		"compare DER and DES encodings",
		"parse a DER certificate and document the DES fallback",
		"the MIT license IST field needs a test",
		"support AUS tax rules in the WO service",
	}

	for _, task := range tasks {
		rec := RecommendWithOptimization(task, OptimizeValue)
		if rec.Language != languageEnglish {
			t.Fatalf("expected the English prompt %q to stay English, got %s", task, rec.Language)
		}
		assertContainsAll(t, Format(rec), "Model: ", "Reason: ")
		assertNotContainsAny(t, Format(rec), "Modell:", "Begründung:")
	}
}

// TestDetectionIgnoresCapitalizationEntirely holds the case-insensitive
// contract against every spelling of one German request, including the mixed
// case an acronym rule would have read as an acronym followed by a German verb.
func TestDetectionIgnoresCapitalizationEntirely(t *testing.T) {
	for _, task := range []string{"fehler beheben", "Fehler beheben", "Fehler Beheben", "FEHLER beheben", "FEHLER BEHEBEN"} {
		if got := detectLanguage(task); got != languageGerman {
			t.Fatalf("detectLanguage(%q) = %s, want German", task, got)
		}
	}

	for _, task := range []string{"compare der and des encodings", "compare DER and DES encodings", "COMPARE DER AND DES ENCODINGS"} {
		if got := detectLanguage(task); got != languageEnglish {
			t.Fatalf("detectLanguage(%q) = %s, want English", task, got)
		}
	}
}

// TestAcronymCollisionsAreNotMarkers covers the curation that replaced reading
// capitals: a German function word that English technical prose spells as an
// acronym is not a marker at all, so two of them never reach the threshold and
// no term counts less than any other.
func TestAcronymCollisionsAreNotMarkers(t *testing.T) {
	collisions := []string{"der", "des", "das", "ist", "mit", "aus", "wo"}

	markers := markerSet()
	for _, term := range collisions {
		if markers[term] {
			t.Errorf("%q is also an acronym of English technical prose and must not be a marker", term)
		}
	}

	for _, task := range []string{strings.Join(collisions, " "), "compare DER and DES encodings", "the MIT license IST field", "support AUS tax rules in the WO service"} {
		if got := detectLanguage(task); got != languageEnglish {
			t.Fatalf("detectLanguage(%q) = %s, want English", task, got)
		}
	}
}

// TestEvidenceIsCountedInGermanWords pins how the threshold is counted: one
// German word is one piece of evidence wherever the vocabulary happens to spell
// it, so a phrase neither hides the German words inside it nor counts a borrowed
// English one.
func TestEvidenceIsCountedInGermanWords(t *testing.T) {
	cases := []struct {
		name string
		task string
		want language
	}{
		{"one word matched by a phrase and its part", "review code prüfen", languageEnglish},
		{"one word matched by a phrase and its part, repeated", "review code prüfen, code prüfen", languageEnglish},
		{"a second german word is further evidence", "bitte code prüfen", languageGerman},
		{"a phrase spelling two german words", "Bearbeite viele Dateien", languageGerman},
		{"the same two german words written separately", "viele geänderte Dateien", languageGerman},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := detectLanguage(tc.task); got != tc.want {
				t.Fatalf("detectLanguage(%q) = %s, want %s", tc.task, got, tc.want)
			}
		})
	}
}

// TestEveryMarkerCarriesGermanEvidence holds the invariant the weighing relies
// on: a term written entirely from English signal words is filtered out of the
// vocabulary rather than registered as a marker worth nothing.
func TestEveryMarkerCarriesGermanEvidence(t *testing.T) {
	for _, marker := range germanMarkers {
		if marker.evidence < 1 {
			t.Errorf("marker %q carries no German-only word", marker.term)
		}
		if got := germanOnlyWordCount(marker.term); got != marker.evidence {
			t.Errorf("marker %q carries evidence %d, want %d", marker.term, marker.evidence, got)
		}
	}

	for term, want := range map[string]int{"viele dateien": 2, "code prüfen": 1, "prüfen": 1, "pull request prüfen": 1, "softwaredesign prüfen": 1} {
		if got := germanOnlyWordCount(term); got != want {
			t.Errorf("germanOnlyWordCount(%q) = %d, want %d", term, got, want)
		}
	}
}

func TestGermanMarkersAreOrderedLongestFirst(t *testing.T) {
	for i := 1; i < len(germanMarkers); i++ {
		if len(germanMarkers[i-1].term) < len(germanMarkers[i].term) {
			t.Fatalf("marker %q precedes the longer marker %q; claiming the covering text needs the longest match first", germanMarkers[i-1].term, germanMarkers[i].term)
		}
	}
}

// TestGermanMarkersExcludeCrossLanguageHomographs proves the exclusions are
// applied to the marker set itself rather than only to the worked examples.
func TestGermanMarkersExcludeCrossLanguageHomographs(t *testing.T) {
	excluded := []string{
		"die", "man", "was", "so", "in", "hat", "bin", "mich",
		"der", "des", "das", "ist", "mit", "aus", "wo",
		"code-audit", "code-review", "code-reviews", "codereview",
		"design-system", "design-tokens", "designsystem", "module", "pull-request",
		"pull-request-review", "refactoring", "software-design", "softwaredesign", "standard", "systemdesign",
	}

	markers := markerSet()
	for _, term := range excluded {
		if markers[term] {
			t.Fatalf("%q also reads as ordinary English and must not be a German marker", term)
		}
	}

	// The classifier's genuinely German vocabulary must still survive the
	// borrowed-English and code-identifier filters.
	for _, term := range []string{"fehler", "implementiere", "benchmark-daten", "modellauswahl"} {
		if !markers[term] {
			t.Fatalf("expected the German classifier signal term %q to be a marker", term)
		}
	}
}

// TestGermanFunctionWordsSurviveTheEnglishFilter guards the one way the
// structural filter could silently weaken detection: readsAsEnglish is applied
// to every marker source, so a function word colliding with the English signal
// vocabulary would disappear without any call site changing.
func TestGermanFunctionWordsSurviveTheEnglishFilter(t *testing.T) {
	markers := markerSet()
	for _, word := range germanFunctionWordMarkers {
		if !markers[word] {
			t.Fatalf("German function word %q was filtered out of the marker set", word)
		}
	}
}

// TestReadsAsEnglishRecognizesBorrowedCompounds pins the rule that a German
// signal term spelled entirely from English signal words describes the task but
// says nothing about the prompt's language.
func TestReadsAsEnglishRecognizesBorrowedCompounds(t *testing.T) {
	cases := []struct {
		term string
		want bool
	}{
		{"pull-request-review", true},
		{"software-design", true},
		{"design-tokens", true},
		{"module", true},
		{"softwaredesign", true},
		{"systemdesign", true},
		{"designsystem", true},
		{"designsysteme", false},
		{"code prüfen", false},
		{"benchmark-daten", false},
		{"modellauswahl", false},
		{"", false},
	}

	for _, tc := range cases {
		if got := readsAsEnglish(tc.term); got != tc.want {
			t.Fatalf("readsAsEnglish(%q) = %t, want %t", tc.term, got, tc.want)
		}
	}
}

// TestEnglishTechnicalPromptsStayEnglish is the contract test behind the English
// bias: prompts written in English must never switch the output language, no
// matter how much borrowed technical vocabulary they pile up. It also bounds the
// cost of every marker added to the vocabulary, since each one is a new chance
// to misread an English prompt.
func TestEnglishTechnicalPromptsStayEnglish(t *testing.T) {
	tasks := []string{
		"update the pull-request-review workflow for the software-design module",
		"refactor the design-system and design-tokens module after the code-review",
		"run a code-audit on the pull-request and fix the standard refactoring gaps",
		"document the software-design decisions for each module in the codereview",
		"rename SystemDesign and DesignSystem in the documentation",
		"support AUS tax rules in the WO service",
		"list the plane geometry helpers and rate limit the request handler",
		"translate the report into a table and write a short summary of each step",
		"create an example page layout and describe the design decision",
		"find the bug in the search index and explain the answer format",
	}

	for _, task := range tasks {
		rec := RecommendWithOptimization(task, OptimizeValue)
		if rec.Language != languageEnglish {
			t.Fatalf("expected the English prompt %q to stay English, got %s", task, rec.Language)
		}
		assertContainsAll(t, Format(rec), "Model: ", "Reason: ")
		assertNotContainsAny(t, Format(rec), "Modell:", "Begründung:")
	}
}

// TestOrdinaryGermanRequestsAreDetected covers the prompts the marker vocabulary
// used to miss: short, everyday requests carrying no classifier signal term at
// all, where the articles alone were one marker short of the threshold.
func TestOrdinaryGermanRequestsAreDetected(t *testing.T) {
	tasks := []string{
		"Übersetze einen Text",
		"Erstelle eine Tabelle",
		"Schreibe eine kurze Antwort",
		"Erkläre mir das Beispiel",
		"Gib mir eine Liste mit Vorschlägen",
		"Zeig mir ein Diagramm",
		"Beschreibe die Aufgabe in drei Schritten",
		"Vereinfache diesen Absatz",
	}

	for _, task := range tasks {
		rec := RecommendWithOptimization(task, OptimizeValue)
		if rec.Language != languageGerman {
			t.Fatalf("expected the German prompt %q to be detected as German, got %s", task, rec.Language)
		}
		assertHumanOnlyOutput(t, Format(rec), languageGerman)
	}
}

// TestGermanRequestMarkersAreGermanOnly holds the curation rule for the request
// vocabulary. Terms here are not filtered by readsAsEnglish, because the words
// they collide with are ordinary English rather than classifier signals, so the
// only guard is that each one is genuinely German-only.
func TestGermanRequestMarkersAreGermanOnly(t *testing.T) {
	forbidden := []string{"plane", "listen", "lade", "laden", "gib", "wort", "text", "tag", "kind", "art", "rate", "band", "brief", "boot", "fast", "not", "hell"}

	registered := make(map[string]bool, len(germanRequestMarkers))
	for _, marker := range germanRequestMarkers {
		if registered[marker] {
			t.Errorf("request marker %q is listed twice", marker)
		}
		registered[marker] = true
	}
	for _, word := range forbidden {
		if registered[word] {
			t.Errorf("%q reads as ordinary English and must not be a request marker", word)
		}
	}

	markers := markerSet()
	for _, marker := range germanRequestMarkers {
		if !markers[marker] {
			t.Errorf("request marker %q never reached the marker set", marker)
		}
	}
}

func markerSet() map[string]bool {
	markers := make(map[string]bool, len(germanMarkers))
	for _, marker := range germanMarkers {
		markers[marker.term] = true
	}
	return markers
}

func TestGermanExplanationIncludesTheGermanCreditLine(t *testing.T) {
	rec := RecommendWithOptimization(germanCodingTask, OptimizeValue)
	entry, ok := benchmarkForRecommendation(rec)
	if !ok {
		t.Fatalf("expected %q to match a bundled benchmark row exactly, got %+v", germanCodingTask, rec)
	}

	out := FormatWithExplanation(rec)
	assertContainsAll(t, out,
		"Benchmark: Pass@1 "+entry.passAt1+"; durchschnittliche Kosten "+entry.averageCost+".",
		"Geschätzte Copilot-AI-Credits: "+strconv.FormatFloat(entry.credits, 'f', 1, 64)+" (Eingabe- und Ausgabe-Tokens, Schätzwert).",
		"Abwägung: "+entry.tradeoff.german,
	)
	assertNotContainsAny(t, out, "average cost", "Estimated Copilot AI credits", "Tradeoff:", "$")
}

func TestEnglishExplanationStaysEnglish(t *testing.T) {
	rec := RecommendWithOptimization(englishCodingTask, OptimizeValue)
	entry, ok := benchmarkForRecommendation(rec)
	if !ok {
		t.Fatalf("expected %q to match a bundled benchmark row exactly, got %+v", englishCodingTask, rec)
	}

	out := FormatWithExplanation(rec)
	assertContainsAll(t, out,
		"Benchmark: Pass@1 "+entry.passAt1+"; average cost "+entry.averageCost+".",
		"Estimated Copilot AI credits: "+strconv.FormatFloat(entry.credits, 'f', 1, 64)+" (input and output tokens, estimate).",
		"Tradeoff: "+entry.tradeoff.english,
	)
	assertNotContainsAny(t, out, "durchschnittliche Kosten", "Geschätzte Copilot-AI-Credits", "Abwägung:")
}

func TestGermanJSONKeepsEnglishIdentifiersAndLocalizesText(t *testing.T) {
	rec := RecommendWithOptimization(germanCodingTask, OptimizeValue)

	document, err := FormatJSON(rec, OptimizeQuality, true)
	if err != nil {
		t.Fatalf("expected JSON format to succeed for a German recommendation: %v", err)
	}

	var doc map[string]any
	if err := json.Unmarshal([]byte(document), &doc); err != nil {
		t.Fatalf("expected valid JSON, got %q: %v", document, err)
	}
	for _, field := range []string{"model", "reasoning", "profile", "reason", "benchmark"} {
		if _, ok := doc[field]; !ok {
			t.Fatalf("expected English field name %q in %q", field, document)
		}
	}
	model, _ := benchmarkModelID(rec.Model)
	if doc["model"] != model || doc["reasoning"] != rec.ReasoningSetting || doc["profile"] != "quality" {
		t.Fatalf("normalized identifiers must stay English and unchanged: %v", doc)
	}

	benchmark := benchmarkObject(t, document)
	entry, ok := benchmarkForRecommendation(rec)
	if !ok {
		t.Fatalf("expected a bundled benchmark row for %+v", rec)
	}
	for key := range benchmark {
		switch key {
		case "pass_at_1", "average_cost", "credits_estimate", "tradeoff":
		default:
			t.Fatalf("unexpected benchmark key %q in %q", key, document)
		}
	}
	if benchmark["credits_estimate"] != entry.credits {
		t.Fatalf("credits_estimate = %v, want %v in %q", benchmark["credits_estimate"], entry.credits, document)
	}
	if doc["reason"] != rec.Reason {
		t.Fatalf("expected the German reason in JSON, got %v", doc["reason"])
	}
	assertReasonLanguage(t, doc["reason"].(string), languageGerman)
	if benchmark["tradeoff"] != entry.tradeoff.german {
		t.Fatalf("expected the German tradeoff, got %v", benchmark["tradeoff"])
	}
}

// TestGermanRecommendationStillResolvesItsBenchmarkData is the regression guard
// for the coupling this feature had to break: while the normalized effort level
// was recovered by trimming an English label off the displayed setting,
// localizing that label would have failed JSON rendering and dropped the
// benchmark block from every German recommendation.
func TestGermanRecommendationStillResolvesItsBenchmarkData(t *testing.T) {
	for _, optimization := range allOptimizations {
		rec := RecommendWithOptimization(germanCodingTask, optimization)

		document, err := FormatJSON(rec, optimization, true)
		if err != nil {
			t.Fatalf("optimization %q: expected JSON format to succeed, got %v", optimization, err)
		}
		if _, ok := benchmarkObject(t, document)["pass_at_1"]; !ok {
			t.Fatalf("optimization %q: expected a benchmark block in %q", optimization, document)
		}
	}
}

// TestLanguageDoesNotChangeTheRecommendation proves detection is one-way: it
// selects the output language and never feeds back into classification or
// selection.
func TestLanguageDoesNotChangeTheRecommendation(t *testing.T) {
	for _, pair := range germanEnglishPairs {
		t.Run(pair.family, func(t *testing.T) {
			for _, optimization := range allOptimizations {
				english := RecommendWithOptimization(pair.english, optimization)
				german := RecommendWithOptimization(pair.german, optimization)

				if english.Language != languageEnglish || german.Language != languageGerman {
					t.Fatalf("optimization %q: expected English/German detection, got %s/%s", optimization, english.Language, german.Language)
				}
				if german.Model != english.Model || german.ReasoningSetting != english.ReasoningSetting {
					t.Fatalf("optimization %q: German %q got %s[%s], English %q got %s[%s]",
						optimization, pair.german, german.Model, german.ReasoningSetting,
						pair.english, english.Model, english.ReasoningSetting)
				}
				if germanDoc, englishDoc := identifiersOnly(t, german, optimization), identifiersOnly(t, english, optimization); !reflect.DeepEqual(germanDoc, englishDoc) {
					t.Fatalf("optimization %q: normalized JSON identifiers differ: German %v, English %v", optimization, germanDoc, englishDoc)
				}
				if german.Reason == english.Reason {
					t.Fatalf("optimization %q: expected a translated reason for %q", optimization, pair.german)
				}
			}
		})
	}

	t.Logf("compared %d German and English prompt pairs across %d optimization modes", len(germanEnglishPairs), len(allOptimizations))
}

// identifiersOnly renders the JSON document and drops the two human-readable
// values, leaving exactly the part of the contract that must not localize.
func identifiersOnly(t *testing.T, rec Recommendation, optimization Optimization) map[string]any {
	t.Helper()

	document, err := FormatJSON(rec, optimization, true)
	if err != nil {
		t.Fatalf("expected JSON format to succeed: %v", err)
	}
	var doc map[string]any
	if err := json.Unmarshal([]byte(document), &doc); err != nil {
		t.Fatalf("expected valid JSON, got %q: %v", document, err)
	}
	delete(doc, "reason")
	if benchmark, ok := doc["benchmark"].(map[string]any); ok {
		delete(benchmark, "tradeoff")
	}
	return doc
}

func TestGermanRecommendationsStayWithinHumanOnlyOutputGuardrails(t *testing.T) {
	tasks := []string{
		"behebe einen Tippfehler in der Readme-Datei",
		"schreibe diese Support-Antwort bestimmt aber freundlich um",
		germanCodingTask,
		"diagnostiziere eine sporadische Wettlaufsituation im Produktivbetrieb",
		"fasse ein langes Dokument zu einem Forschungsbericht zusammen",
		"entwirf ein Drahtmodell mit Farbpalette und Typografie",
		"hilf mir bitte bei dieser Sache",
	}

	for _, task := range tasks {
		for _, optimization := range allOptimizations {
			rec := RecommendWithOptimization(task, optimization)
			if rec.Language != languageGerman {
				t.Fatalf("expected %q to be detected as German", task)
			}
			assertHumanOnlyOutput(t, Format(rec), languageGerman)
		}
	}
}

func TestEveryReasonIsTranslatedIntoEveryLanguage(t *testing.T) {
	registered := make(map[reasonID]bool, len(allReasonIDs))
	for _, id := range allReasonIDs {
		registered[id] = true

		text, ok := reasonCopy[id]
		if !ok {
			t.Fatalf("reason %q has no copy registered", id)
		}
		if text.english == "" {
			t.Fatalf("reason %q has no English text", id)
		}
		if text.german == "" {
			t.Fatalf("reason %q has no German text", id)
		}
		if text.english == text.german {
			t.Fatalf("reason %q carries the same text in both languages", id)
		}
	}

	for id := range reasonCopy {
		if !registered[id] {
			t.Fatalf("reason %q has copy but is not registered in allReasonIDs", id)
		}
	}
}

func TestEveryOutputLabelIsDefinedInEveryLanguage(t *testing.T) {
	for _, lang := range allLanguages {
		labels, ok := labelsByLanguage[lang]
		if !ok {
			t.Fatalf("no output labels registered for %s", lang)
		}
		value := reflect.ValueOf(labels)
		for i := 0; i < value.NumField(); i++ {
			if value.Field(i).String() == "" {
				t.Fatalf("%s output label %q is missing", lang, value.Type().Field(i).Name)
			}
		}
	}
	if len(labelsByLanguage) != len(allLanguages) {
		t.Fatalf("%d label sets are registered for %d languages", len(labelsByLanguage), len(allLanguages))
	}
}

func TestEveryBundledTradeoffIsTranslatedIntoEveryLanguage(t *testing.T) {
	for key, entry := range bundledBenchmarks {
		for _, lang := range allLanguages {
			if text := entry.tradeoff.in(lang); strings.TrimSpace(text) == "" {
				t.Fatalf("bundled benchmark %v has no %s tradeoff", key, lang)
			}
		}
		if entry.tradeoff.english == entry.tradeoff.german {
			t.Fatalf("bundled benchmark %v carries the same tradeoff text in both languages", key)
		}
	}
}

// TestMissingTranslationFailsInsteadOfFallingBackToEnglish pins the behavior
// that makes drift visible: an untranslated sentence must fail loudly rather
// than serve English inside otherwise German output.
func TestMissingTranslationFailsInsteadOfFallingBackToEnglish(t *testing.T) {
	defer func() {
		recovered := recover()
		if recovered == nil {
			t.Fatal("expected a missing German translation to panic instead of returning English")
		}
		if message, ok := recovered.(string); !ok || !strings.Contains(message, "untranslated sentence") {
			t.Fatalf("expected the panic to name the untranslated text, got %v", recovered)
		}
	}()

	_ = localizedText{english: "untranslated sentence"}.in(languageGerman)
}

// TestEverySignalFamilyIsRegistered guards the marker vocabulary against the one
// failure it cannot detect itself: a new signal family whose terms never reach
// language detection because it was not added to allSignalFamilies.
//
// It compares family identities rather than counts. Counting occurrences of a
// source pattern would break on any harmless change to how the families are
// written, and would still pass if one family were dropped while another was
// registered twice.
func TestEverySignalFamilyIsRegistered(t *testing.T) {
	declared := declaredSignalFamilies(t)

	for name := range declared {
		if _, ok := allSignalFamilies[name]; !ok {
			t.Errorf("classifier.go declares signal family %q, but it is not registered in allSignalFamilies", name)
		}
	}
	for name := range allSignalFamilies {
		if !declared[name] {
			t.Errorf("allSignalFamilies registers %q, which is not a signal family declared in classifier.go", name)
		}
	}

	t.Logf("verified %d declared signal families against %d registry entries", len(declared), len(allSignalFamilies))
}

// TestSignalFamilyRegistryKeysNameTheirFamily closes the gap a name-keyed
// registry would otherwise leave: keys and values are independent, so without
// this the same family could be registered under two names while another went
// missing, and the set comparison above would still balance.
func TestSignalFamilyRegistryKeysNameTheirFamily(t *testing.T) {
	registry := signalFamilyRegistryEntries(t)

	if len(registry) != len(allSignalFamilies) {
		t.Fatalf("parsed %d registry entries but allSignalFamilies holds %d", len(registry), len(allSignalFamilies))
	}
	for key, value := range registry {
		if key != value {
			t.Errorf("allSignalFamilies maps key %q to family %q; the key must name its family", key, value)
		}
	}
}

// declaredSignalFamilies returns the name of every package-level variable in
// classifier.go initialized with a signalTerms literal.
func declaredSignalFamilies(t *testing.T) map[string]bool {
	t.Helper()

	declared := make(map[string]bool)
	for _, spec := range classifierValueSpecs(t) {
		for i, name := range spec.Names {
			if i >= len(spec.Values) {
				continue
			}
			literal, ok := spec.Values[i].(*ast.CompositeLit)
			if !ok {
				continue
			}
			if identifier, ok := literal.Type.(*ast.Ident); ok && identifier.Name == "signalTerms" {
				if declared[name.Name] {
					t.Fatalf("signal family %q is declared twice in classifier.go", name.Name)
				}
				declared[name.Name] = true
			}
		}
	}
	if len(declared) == 0 {
		t.Fatal("found no signal family declarations in classifier.go; the parser no longer matches the source")
	}
	return declared
}

// signalFamilyRegistryEntries returns the allSignalFamilies literal as a mapping
// from each key string to the name of the variable it points at.
func signalFamilyRegistryEntries(t *testing.T) map[string]string {
	t.Helper()

	for _, spec := range classifierValueSpecs(t) {
		for i, name := range spec.Names {
			if name.Name != "allSignalFamilies" || i >= len(spec.Values) {
				continue
			}
			literal, ok := spec.Values[i].(*ast.CompositeLit)
			if !ok {
				t.Fatalf("allSignalFamilies is not a composite literal")
			}

			entries := make(map[string]string, len(literal.Elts))
			for _, element := range literal.Elts {
				pair, ok := element.(*ast.KeyValueExpr)
				if !ok {
					t.Fatalf("allSignalFamilies holds a non key-value element")
				}
				key, ok := pair.Key.(*ast.BasicLit)
				if !ok || key.Kind != token.STRING {
					t.Fatalf("allSignalFamilies has a non-string key %v", pair.Key)
				}
				value, ok := pair.Value.(*ast.Ident)
				if !ok {
					t.Fatalf("allSignalFamilies maps %s to something other than a named family", key.Value)
				}
				name, err := strconv.Unquote(key.Value)
				if err != nil {
					t.Fatalf("cannot unquote registry key %s: %v", key.Value, err)
				}
				entries[name] = value.Name
			}
			return entries
		}
	}

	t.Fatal("allSignalFamilies is not declared in classifier.go")
	return nil
}

func classifierValueSpecs(t *testing.T) []*ast.ValueSpec {
	t.Helper()

	file, err := parser.ParseFile(token.NewFileSet(), "classifier.go", nil, 0)
	if err != nil {
		t.Fatalf("cannot parse the classifier source: %v", err)
	}

	var specs []*ast.ValueSpec
	for _, declaration := range file.Decls {
		general, ok := declaration.(*ast.GenDecl)
		if !ok || general.Tok != token.VAR {
			continue
		}
		for _, spec := range general.Specs {
			if value, ok := spec.(*ast.ValueSpec); ok {
				specs = append(specs, value)
			}
		}
	}
	return specs
}

func TestBundledBenchmarkLevelsAreKnownEffortLevels(t *testing.T) {
	for key := range bundledBenchmarks {
		if _, ok := normalizeEffortLevel(key.level); !ok {
			t.Fatalf("bundled benchmark level %q is not a known effort level", key.level)
		}
	}
}

// assertReasonLanguage checks that a reason sentence came from the expected
// language column of the reason table, without needing to know which reason the
// bands selected.
func assertReasonLanguage(t *testing.T, reason string, lang language) {
	t.Helper()

	for _, text := range reasonCopy {
		if reason == text.in(lang) {
			return
		}
		if reason == text.in(otherLanguage(lang)) {
			t.Fatalf("expected a %s reason, got the %s one: %q", lang, otherLanguage(lang), reason)
		}
	}
	t.Fatalf("reason %q is not registered in the reason table", reason)
}

func otherLanguage(lang language) language {
	if lang == languageGerman {
		return languageEnglish
	}
	return languageGerman
}
