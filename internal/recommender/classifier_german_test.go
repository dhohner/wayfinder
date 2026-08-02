package recommender

import (
	"strings"
	"testing"
)

// germanEnglishPairs covers every classifier signal family, including the three
// term lists that only gate isCodeReview and isCorrectnessHeavyCoding, with a
// German prompt and its English equivalent. Both prompts of a pair must reach
// the same recommendation in every optimization mode.
var germanEnglishPairs = []struct {
	family  string
	english string
	german  string
}{
	{
		family:  "simpleSignals",
		english: "summarize these release notes",
		german:  "eine kurze zusammenfassung dieser versionshinweise",
	},
	{
		family:  "codingSignals",
		english: "implement a small Go API endpoint",
		german:  "implementiere einen kleinen Go-API-Endpunkt",
	},
	{
		family:  "codingIntentSignals",
		english: "refactor the parser module",
		german:  "refaktoriere das Parser-Modul",
	},
	{
		family:  "largeContextSignals",
		english: "plan a legacy monorepo migration across multiple files",
		german:  "plane eine Migration im Altsystem über mehrere Dateien",
	},
	{
		family:  "codeReviewSignals",
		english: "do a code review of this pull request",
		german:  "führe ein Code-Review für diesen Pull Request durch",
	},
	{
		family:  "anthropicFitSignals",
		english: "write a long essay with a clear narrative",
		german:  "schreibe einen Aufsatz mit einer klaren Erzählung",
	},
	{
		family:  "visualDesignSignals",
		english: "design a wireframe with a color palette and typography",
		german:  "entwirf ein Drahtmodell mit Farbpalette und Typografie",
	},
	{
		family:  "technicalDesignSignals",
		english: "propose a system architecture for this service",
		german:  "schlage eine Systemarchitektur für diesen Dienst vor",
	},
	{
		family:  "nuancedRoutineSignals",
		english: "extract requirements from a messy product request",
		german:  "leite die Anforderungen aus einer unstrukturierten Produktanfrage ab",
	},
	{
		family:  "deepReasoningSignals",
		english: "diagnose a memory leak and optimize the state machine",
		german:  "diagnostiziere ein Speicherleck und optimiere den Zustandsautomaten",
	},
	{
		family:  "highRiskSignals",
		english: "check the access control and secret handling for security",
		german:  "prüfe die Zugriffskontrolle und die Zugangsdaten auf Sicherheit",
	},
	{
		family:  "correctnessHeavySignals",
		english: "compare arbitrarily large values with lossless precision",
		german:  "vergleiche beliebig große Werte verlustfrei und präzise",
	},
	{
		family:  "routineCodingSignals",
		english: "add a CLI flag and update the usage text",
		german:  "füge ein Kommandozeilen-Flag hinzu und aktualisiere den Hilfetext",
	},
	{
		family:  "modelSelectionSignals",
		english: "tune the classifier signals for model selection",
		german:  "aktualisiere die Klassifizierer-Signale für die Modellauswahl",
	},
	{
		family:  "modelSelectionCodingActionSignals",
		english: "broaden the recommendation rules",
		german:  "erweitere die Regeln für die Modellauswahl",
	},
	{
		family:  "reviewActionSignals and reviewObjectSignals",
		english: "review the auth module implementation",
		german:  "überprüfe die Implementierung des Moduls für die Authentifizierung",
	},
	{
		family:  "correctnessHeavyActionSignals",
		english: "preserve the current behavior for rounding and overflow",
		german:  "bewahre das aktuelle Verhalten bei Rundung und Überlauf",
	},
	{
		family:  "metaCodeReviewFeatureSignals",
		english: "update the model selection for code-review tasks",
		german:  "aktualisiere die Modellauswahl für Code-Reviews",
	},
}

func TestGermanPromptsGetTheSameRecommendationAsTheirEnglishEquivalent(t *testing.T) {
	for _, pair := range germanEnglishPairs {
		t.Run(pair.family, func(t *testing.T) {
			for _, optimization := range allOptimizations {
				english := RecommendWithOptimization(pair.english, optimization)
				german := RecommendWithOptimization(pair.german, optimization)

				if german.Model != english.Model || german.ReasoningSetting != english.ReasoningSetting {
					t.Fatalf("optimization %q: German %q got %s / %s, English %q got %s / %s",
						optimization, pair.german, german.Model, german.ReasoningSetting,
						pair.english, english.Model, english.ReasoningSetting)
				}
				if german.Reason == defaultRecommendation().Reason {
					t.Fatalf("optimization %q: German prompt %q fell through to the ambiguous-task default", optimization, pair.german)
				}
				if english.Reason == defaultRecommendation().Reason {
					t.Fatalf("optimization %q: English prompt %q fell through to the ambiguous-task default", optimization, pair.english)
				}
			}
		})
	}

	t.Logf("compared %d German and English prompt pairs across %d optimization modes", len(germanEnglishPairs), len(allOptimizations))
}

func TestClassifierAppliesWordBoundariesAcrossGermanCharacters(t *testing.T) {
	cases := []struct {
		name  string
		task  string
		trait func(taskTraits) bool
		want  bool
	}{
		{
			name:  "term followed by ä stays inside the longer word",
			task:  "kurzärmelige Hemden bestellen",
			trait: func(traits taskTraits) bool { return traits.simple },
			want:  false,
		},
		{
			name:  "term followed by ö stays inside the longer word",
			task:  "ein Datenbankökosystem bewerten",
			trait: func(traits taskTraits) bool { return traits.coding },
			want:  false,
		},
		{
			name:  "term followed by ü stays inside the longer word",
			task:  "eine Regelüberwachung im Sportverein",
			trait: func(traits taskTraits) bool { return traits.modelSelection },
			want:  false,
		},
		{
			name:  "term preceded by ß stays inside the longer word",
			task:  "ein Plakat im Großformat drucken",
			trait: func(traits taskTraits) bool { return traits.simple },
			want:  false,
		},
		{
			name:  "term containing ä matches as a whole word",
			task:  "dokumentiere die Grenzfälle",
			trait: func(traits taskTraits) bool { return traits.correctnessHeavy },
			want:  true,
		},
		{
			name:  "term containing ö matches as a whole word",
			task:  "melde den Störfall",
			trait: func(traits taskTraits) bool { return traits.highRisk },
			want:  true,
		},
		{
			name:  "term containing ü matches as a whole word",
			task:  "plane eine Sicherheitsüberprüfung",
			trait: func(traits taskTraits) bool { return traits.highRisk },
			want:  true,
		},
		{
			name:  "term containing ß matches as a whole word",
			task:  "vergleiche große Werte",
			trait: func(traits taskTraits) bool { return traits.correctnessHeavy },
			want:  true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// The same text is classified in the spelling a developer would
			// write it and in an all-caps spelling, because German nouns are
			// capitalized and matching must not depend on case.
			for _, spelling := range []string{strings.ToLower(tc.task), tc.task, strings.ToUpper(tc.task)} {
				if got := tc.trait(classify(spelling)); got != tc.want {
					t.Fatalf("classify(%q) gave trait %t, want %t", spelling, got, tc.want)
				}
			}
		})
	}
}

func TestClassifierMatchesGermanAndEnglishSignalsInTheSamePrompt(t *testing.T) {
	const (
		englishPart = "implement a Go endpoint"
		germanPart  = "beachte dabei die Barrierefreiheit"
	)
	mixed := englishPart + " und " + germanPart

	if classify(englishPart).visualDesign {
		t.Fatalf("expected the English part alone not to carry the visual-design trait")
	}
	if classify(germanPart).coding {
		t.Fatalf("expected the German part alone not to carry the coding trait")
	}

	traits := classify(mixed)
	if !traits.coding {
		t.Fatalf("expected %q to carry the coding trait from its English terms, got %+v", mixed, traits)
	}
	if !traits.visualDesign {
		t.Fatalf("expected %q to carry the visual-design trait from its German terms, got %+v", mixed, traits)
	}
}

func TestClassifierMatchesCapitalizedGermanNounsLikeLowerCaseOnes(t *testing.T) {
	cases := []struct{ capitalized, lower string }{
		{"prüfe die Sicherheit der Anmeldung", "prüfe die sicherheit der anmeldung"},
		{"entwirf die Architektur des Dienstes", "entwirf die architektur des dienstes"},
		{"schreibe eine Zusammenfassung des Berichts", "schreibe eine zusammenfassung des berichts"},
		{"prüfe die Barrierefreiheit der Benutzeroberfläche", "prüfe die barrierefreiheit der benutzeroberfläche"},
	}

	for _, tc := range cases {
		t.Run(tc.lower, func(t *testing.T) {
			if got, want := classify(tc.capitalized), classify(tc.lower); got != want {
				t.Fatalf("classify(%q) = %+v, classify(%q) = %+v", tc.capitalized, got, tc.lower, want)
			}
			if (classify(tc.lower) == taskTraits{}) {
				t.Fatalf("expected %q to carry at least one trait", tc.lower)
			}
		})
	}
}
