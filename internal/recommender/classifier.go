package recommender

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

type taskTraits struct {
	simple           bool
	coding           bool
	codingIntent     bool
	codeReview       bool
	largeContext     bool
	anthropicFit     bool
	visualDesign     bool
	technicalDesign  bool
	nuancedRoutine   bool
	deepReasoning    bool
	highRisk         bool
	correctnessHeavy bool
	routineCoding    bool
	modelSelection   bool
	against          AgainstFamily
}

// signalTerms is one signal family's vocabulary, held as an English and a German
// term list that are always matched together. A prompt in either language, or a
// prompt mixing both, reaches the same traits and therefore the same
// recommendation.
//
// Maintenance rule: English and German terms are maintained side by side, and
// every term added to a family must be added in both languages. Parity between
// the two prompt languages is a durable property of the classifier, not a
// one-time migration.
//
// German terms are derived from the family's intent rather than translated word
// for word, and they list the inflected and compound forms a developer actually
// writes ("implementieren", "implementiere", "implementierung"), because terms
// are matched as whole words and a single dictionary form would miss most real
// prompts. Terms are stored lower case because classify lowercases the task
// before matching. A term whose German spelling is identical to the English one
// stays in the English list only; repeating it would broaden nothing.
type signalTerms struct {
	english []string
	german  []string
}

// matches reports whether text contains any term of the family, in either
// language, as a whole word.
func (s signalTerms) matches(text string) bool {
	return hasAny(text, s.english...) || hasAny(text, s.german...)
}

var simpleSignals = signalTerms{
	english: []string{
		"summarize", "summary", "rewrite", "proofread", "copy edit", "format", "extract", "release notes", "short email", "typo", "spelling", "grammar", "readme", "changelog", "rename", "one-line", "small", "minor", "quick", "lint", "comment", "documentation",
	},
	german: []string{
		"zusammenfassen", "zusammenfassung", "zusammengefasst", "kurzfassung", "überblick", "übersicht",
		"umschreiben", "umschreibe", "umformulieren", "umformulierung",
		"korrekturlesen", "korrektur", "lektorat", "lektorieren", "redigieren",
		"formatieren", "formatiere", "formatierung",
		"extrahieren", "extrahiere", "auslesen",
		"versionshinweise", "änderungshinweise", "änderungsprotokoll",
		"kurze e-mail", "kurze mail", "kurzer text",
		"tippfehler", "schreibfehler", "rechtschreibung", "rechtschreibfehler", "grammatik", "grammatikfehler",
		"umbenennen", "umbenennung", "benenne um",
		"einzeilig", "einzeiler",
		"klein", "kleine", "kleinen", "kleiner", "kleines",
		"geringfügig", "geringfügige", "unwesentlich",
		"schnell", "schnelle", "schnellen", "kurz", "kurze", "kurzen",
		"kommentar", "kommentare", "kommentieren", "kommentiere",
		"dokumentation", "dokumentieren", "dokumentiere",
	},
}

var codingSignals = signalTerms{
	english: []string{
		"code", "coding", "implement", "implementation", "refactor", "debug", "test", "typescript", "javascript", "react", "vue", "angular", "css", "html", "jsx", "tsx", "golang", "go", "python", "rust", "java", "sql", "api", "sdk", "cli", "json", "serialization", "serialize", "flag", "command-line", "classifier", "recommender", "module", "bug", "endpoint", "function", "class", "component", "frontend", "backend", "database", "schema", "query", "build", "deploy", "parser", "parse",
	},
	german: []string{
		"programmieren", "programmiere", "programmierung", "quellcode", "quelltext",
		"implementieren", "implementiere", "implementiert", "implementierung", "umsetzen", "umsetze", "umsetzung",
		"refaktorieren", "refaktoriere", "refaktorisieren", "refactoring", "überarbeiten", "überarbeite", "umbauen",
		"debuggen", "fehlersuche", "fehler", "fehlerhaft", "fehlerbehebung", "beheben", "behebe",
		"testen", "teste", "testfall", "testfälle",
		"endpunkt", "endpunkte", "schnittstelle", "schnittstellen",
		"funktion", "funktionen", "klasse", "klassen", "komponente", "komponenten", "modul", "module",
		"datenbank", "datenbanken", "datenbankschema", "abfrage", "abfragen",
		"bauen", "baue", "kompilieren", "kompiliere",
		"deployen", "ausrollen", "bereitstellen", "bereitstellung",
		"parsen", "zerlegen",
		"serialisieren", "serialisierung",
		"kommandozeile", "kommandozeilen", "kommandozeilenargument", "schalter",
		"klassifizierer", "klassifikator",
	},
}

var codingIntentSignals = signalTerms{
	english: []string{
		"code", "coding", "implement", "implementation", "refactor", "debug", "test", "typescript", "javascript", "react", "vue", "angular", "css", "html", "jsx", "tsx", "golang", "go", "python", "rust", "java", "sql", "api", "sdk", "cli", "json", "serialization", "serialize", "flag", "command-line", "classifier", "recommender", "module", "bug", "endpoint", "function", "class", "backend", "database", "schema", "query", "deploy", "parser", "parse",
	},
	german: []string{
		"programmieren", "programmiere", "programmierung", "quellcode", "quelltext",
		"implementieren", "implementiere", "implementiert", "implementierung", "umsetzen", "umsetze", "umsetzung",
		"refaktorieren", "refaktoriere", "refaktorisieren", "refactoring", "überarbeiten", "überarbeite", "umbauen",
		"debuggen", "fehlersuche", "fehler", "fehlerhaft", "fehlerbehebung", "beheben", "behebe",
		"testen", "teste", "testfall", "testfälle",
		"endpunkt", "endpunkte", "schnittstelle", "schnittstellen",
		"funktion", "funktionen", "klasse", "klassen", "modul", "module",
		"datenbank", "datenbanken", "datenbankschema", "abfrage", "abfragen",
		"deployen", "ausrollen", "bereitstellen", "bereitstellung",
		"parsen", "zerlegen",
		"serialisieren", "serialisierung",
		"kommandozeile", "kommandozeilen", "kommandozeilenargument", "schalter",
		"klassifizierer", "klassifikator",
	},
}

var largeContextSignals = signalTerms{
	english: []string{
		"many files", "multiple files", "whole repo", "entire repo", "repository", "repo", "codebase", "monorepo", "migration", "cross-service", "multi-service", "integration", "legacy", "large file", "large files", "10-page", "10 page", "thousands of lines",
	},
	german: []string{
		"viele dateien", "mehrere dateien", "ganzes repository", "gesamtes repository",
		"codebasis", "quellcodebasis", "gesamte codebasis",
		"umstellung", "portierung",
		"dienstübergreifend", "serviceübergreifend", "systemübergreifend", "modulübergreifend",
		"altsystem", "altsysteme", "altcode",
		"große datei", "große dateien", "tausende zeilen", "tausenden zeilen", "zehnseitig", "mehrere module",
	},
}

var codeReviewSignals = signalTerms{
	english: []string{
		"code review", "review code", "review the code", "review this code", "review my code", "adversarial code review", "pull request review", "review pull request", "review a pull request", "review this pull request", "review the pull request", "review pr", "review a pr", "review this pr", "review the pr", "review diff", "review a diff", "review this diff", "review the diff", "review implementation", "review an implementation", "review this implementation", "review the implementation", "implementation review", "review patch", "review a patch", "review the patch", "review this patch", "audit code", "audit the code", "audit this code", "audit pull request", "audit a pull request", "audit this pull request", "audit pr", "audit a pr", "audit this pr", "audit diff", "audit this diff", "audit patch", "audit this patch",
	},
	german: []string{
		"code-review", "code-reviews", "codereview", "code-durchsicht",
		"codeprüfung", "code-prüfung", "quellcodeprüfung", "quellcode-prüfung", "implementierungsprüfung",
		"code prüfen", "code überprüfen", "code begutachten", "quellcode prüfen", "quellcode überprüfen", "quelltext prüfen", "quelltext überprüfen",
		"implementierung prüfen", "implementierung überprüfen", "änderungen prüfen", "änderungen überprüfen",
		"pull request prüfen", "pull-request prüfen", "pull-request-review", "pr prüfen", "diff prüfen", "patch prüfen",
		"code auditieren", "code-audit",
	},
}

var anthropicFitSignals = signalTerms{
	english: []string{
		"long document", "long-form", "longform", "essay", "narrative", "manuscript", "policy brief", "research brief", "research report", "market analysis", "literature review", "creative writing", "story", "tone", "voice", "brand voice", "editorial", "script", "speech",
	},
	german: []string{
		"langes dokument", "lange dokumente", "langform",
		"aufsatz", "erzählung", "erzählend", "narrativ", "manuskript",
		"positionspapier", "grundsatzpapier", "forschungsbericht", "marktanalyse", "literaturübersicht", "literaturrecherche",
		"kreatives schreiben", "geschichte",
		"tonfall", "tonalität", "markenstimme", "redaktionell", "leitartikel",
		"drehbuch", "rede", "ansprache",
	},
}

var visualDesignSignals = signalTerms{
	english: []string{
		"visual design", "ui", "ux", "ui design", "ux design", "user interface", "user experience", "user interface design", "user experience design", "interface design", "interaction design", "design system", "design tokens", "mockup", "wireframe", "prototype", "layout", "page layout", "screen design", "typography", "color palette", "visual identity", "brand design", "brand identity", "branding", "accessibility review", "a11y", "figma",
	},
	german: []string{
		"visuelles design", "gestaltung", "oberflächendesign",
		"benutzeroberfläche", "benutzeroberflächen", "benutzererlebnis", "nutzererlebnis", "benutzerführung", "benutzerfreundlichkeit",
		"interaktionsdesign", "designsystem", "design-system", "designsysteme", "design-tokens",
		"drahtmodell", "prototyp", "seitenlayout", "bildschirmdesign", "bildschirmentwurf",
		"typografie", "farbpalette", "farbschema",
		"visuelle identität", "markenidentität", "markendesign",
		"barrierefreiheit", "barrierefrei", "zugänglichkeit",
	},
}

var technicalDesignSignals = signalTerms{
	english: []string{
		"architecture", "software architecture", "system architecture", "system design", "technical design", "software design", "engineering design",
	},
	german: []string{
		"architektur", "softwarearchitektur", "software-architektur", "systemarchitektur", "architekturentwurf",
		"systementwurf", "systemdesign", "technischer entwurf", "technisches design", "softwaredesign", "software-design",
	},
}

var nuancedRoutineSignals = signalTerms{
	english: []string{
		"messy", "inconsistent", "nuanced", "firm but empathetic", "preserve intent", "multiple constraints", "overlap", "overlapping", "edge case", "edge cases", "requirements", "product request", "project plan", "meeting notes", "support reply", "policy", "stakeholder", "prioritize", "triage",
	},
	german: []string{
		"unstrukturiert", "unstrukturierte", "unstrukturierten", "unsortiert", "chaotisch", "durcheinander",
		"inkonsistent", "inkonsistente", "widersprüchlich", "uneinheitlich",
		"nuanciert", "differenziert", "feinfühlig", "empathisch", "bestimmt aber freundlich",
		"absicht bewahren", "intention bewahren",
		"randbedingungen", "mehrere einschränkungen",
		"überschneidung", "überschneidungen", "überschneiden",
		"grenzfall", "grenzfälle", "sonderfall", "sonderfälle",
		"anforderung", "anforderungen",
		"produktanfrage", "produktwunsch", "projektplan",
		"besprechungsnotizen", "sitzungsprotokoll", "meeting-notizen",
		"support-antwort", "supportanfrage", "kundenantwort",
		"richtlinie", "richtlinien",
		"beteiligte", "interessengruppen",
		"priorisieren", "priorisiere", "priorisierung", "einordnen", "sichten",
	},
}

var deepReasoningSignals = signalTerms{
	english: []string{
		"architecture", "system design", "technical design", "software design", "engineering design", "distributed", "intermittent", "root cause", "complex", "race condition", "concurrency", "deadlock", "performance", "scalability", "optimize", "profiling", "memory leak", "algorithm", "state machine", "data model", "investigate", "diagnose",
	},
	german: []string{
		"architektur", "softwarearchitektur", "systemarchitektur", "systementwurf", "systemdesign", "technischer entwurf", "technisches design", "softwaredesign",
		"verteilt", "verteilte", "verteilten", "verteiltes",
		"sporadisch", "sporadische", "zeitweise",
		"ursache", "grundursache", "ursachenanalyse", "fehleranalyse",
		"komplex", "komplexe", "komplexen", "komplexes",
		"wettlaufsituation", "nebenläufigkeit", "parallelität", "verklemmung",
		"leistung", "leistungsproblem", "performanz", "skalierbarkeit", "skalierung",
		"optimieren", "optimiere", "optimierung", "profilierung",
		"speicherleck", "speicherlecks",
		"algorithmus", "algorithmen",
		"zustandsautomat", "zustandsautomaten", "zustandsmaschine", "datenmodell",
		"untersuchen", "untersuche", "analysieren", "analysiere",
		"diagnostizieren", "diagnostiziere",
	},
}

var highRiskSignals = signalTerms{
	english: []string{
		"security", "auth", "authentication", "authorization", "oauth", "sso", "rbac", "permission", "permissions", "secret", "token", "payment", "billing", "invoice", "production", "data loss", "incident", "compliance", "privacy", "pii", "gdpr", "hipaa", "pci", "encryption", "legal", "medical", "financial", "finance", "access control",
	},
	german: []string{
		"sicherheit", "sicherheitslücke", "sicherheitslücken", "sicherheitsüberprüfung", "sicherheitsrelevant", "sicherheitskritisch", "sicherheitsvorfall",
		"authentifizierung", "authentisierung", "autorisierung", "anmeldung", "anmeldedaten",
		"berechtigung", "berechtigungen", "zugriffskontrolle", "zugriffsrechte", "zugangsdaten",
		"geheimnis", "geheimnisse", "passwort", "passwörter",
		"zahlung", "zahlungen", "bezahlung", "abrechnung", "rechnung", "rechnungsstellung",
		"produktion", "produktiv", "produktivsystem", "produktivbetrieb", "produktivumgebung",
		"datenverlust", "vorfall", "störfall",
		"datenschutz", "privatsphäre", "personenbezogene daten", "dsgvo",
		"verschlüsselung", "verschlüsseln",
		"rechtlich", "juristisch", "medizinisch", "finanziell", "finanzen", "haftung",
	},
}

var correctnessHeavySignals = signalTerms{
	english: []string{
		"typed comparison", "type comparison", "stable ordering", "stable sort", "edge case", "edge cases", "arbitrarily large", "large values", "big integer", "bignum", "precision", "precise", "lossless", "lossless precision", "required behavior", "current behavior", "preserve behavior", "guarantee", "guarantees", "rounding", "overflow",
	},
	german: []string{
		"typisierter vergleich", "typvergleich", "typsicherer vergleich",
		"stabile sortierung", "stabile reihenfolge", "sortierstabilität",
		"grenzfall", "grenzfälle", "sonderfall", "sonderfälle",
		"beliebig große", "beliebig großen", "große werte", "großen werten", "große zahlen", "ganzzahl", "ganzzahlen",
		"genauigkeit", "genauigkeitsverlust", "präzision", "präzise", "verlustfrei", "verlustfreie",
		"gefordertes verhalten", "aktuelles verhalten", "aktuelle verhalten", "verhalten bewahren", "verhalten beibehalten",
		"garantie", "garantien", "garantieren", "garantiere",
		"rundung", "runden", "überlauf", "überläufe",
	},
}

var routineCodingSignals = signalTerms{
	english: []string{
		"cli", "command-line", "flag", "argument", "usage text", "json", "format", "formatting", "serialization", "serialize", "test", "tests", "coverage", "helper", "helpers", "wrapper", "runner", "service", "split", "extract", "reusable", "wire", "plumbing",
	},
	german: []string{
		"kommandozeile", "kommandozeilen", "kommandozeilenargument", "schalter", "argumente",
		"nutzungshinweis", "hilfetext", "hilfeausgabe",
		"formatierung", "serialisierung", "serialisieren",
		"testen", "teste", "testabdeckung", "testüberdeckung", "abdeckung",
		"hilfsfunktion", "hilfsfunktionen", "hilfsklasse", "hilfsmethode",
		"dienst", "dienste",
		"aufteilen", "aufspalten", "auslagern", "extrahieren", "extrahiere",
		"wiederverwendbar", "wiederverwendung", "verdrahten", "anbinden", "anbindung",
	},
}

var modelSelectionSignals = signalTerms{
	english: []string{
		"model selection", "recommendation", "recommendations", "recommender", "classifier", "classification", "signal", "signals", "rule", "rules", "benchmark", "reasoning level", "effort level", "optimization mode", "optimization modes", "model family", "provider terminology",
	},
	german: []string{
		"modellauswahl", "modell-auswahl", "modellwahl", "modellempfehlung",
		"empfehlung", "empfehlungen", "empfehlungslogik",
		"klassifizierer", "klassifikator", "klassifizierung",
		"signale", "signalliste",
		"regel", "regeln", "regelwerk",
		"benchmark-daten", "benchmarkwerte",
		"denkstufe", "denkaufwand", "aufwandsstufe",
		"optimierungsmodus", "optimierungsmodi",
		"modellfamilie", "modellfamilien", "anbieterbegriffe",
	},
}

var modelSelectionCodingActionSignals = signalTerms{
	english: []string{
		"add", "tune", "update", "change", "modify", "broaden", "narrow", "route", "select", "prefer", "default", "support", "classify", "implement", "refactor", "extract", "replace", "remove",
	},
	german: []string{
		"hinzufügen", "füge hinzu", "ergänzen", "ergänze",
		"anpassen", "passe an", "justieren", "abstimmen",
		"aktualisieren", "aktualisiere",
		"ändern", "ändere", "verändern", "abändern", "modifizieren",
		"erweitern", "erweitere", "ausweiten",
		"einschränken", "eingrenzen", "verengen",
		"routen", "weiterleiten", "leite weiter",
		"auswählen", "wähle", "wählen",
		"bevorzugen", "bevorzuge", "standard", "voreinstellung",
		"unterstützen", "unterstütze", "unterstützung",
		"klassifizieren", "klassifiziere", "einordnen",
		"implementieren", "implementiere", "umsetzen", "umsetze",
		"refaktorieren", "überarbeiten", "überarbeite", "umbauen",
		"extrahieren", "extrahiere", "ersetzen", "ersetze",
		"entfernen", "entferne", "löschen", "lösche",
	},
}

// reviewActionSignals and reviewObjectSignals gate the fallback branch of
// isCodeReview: a review verb applied to a code object. They carry German verbs
// separately from codeReviewSignals because German puts the verb and its object
// far apart ("überprüfe die Implementierung des Moduls"), which no fixed phrase
// in codeReviewSignals can capture.
var reviewActionSignals = signalTerms{
	english: []string{"review", "audit"},
	german: []string{
		"prüfe", "prüfen", "prüfung", "überprüfe", "überprüfen", "überprüfung",
		"begutachte", "begutachten", "begutachtung", "durchsehen", "durchsicht", "gegenlesen",
		"kontrolliere", "kontrollieren", "auditiere", "auditieren",
	},
}

var reviewObjectSignals = signalTerms{
	english: []string{
		"code", "implementation", "pull request", "pr", "diff", "patch", "bug", "bugs", "module", "function", "class", "component", "endpoint", "repository", "repo", "codebase",
		"typescript", "javascript", "golang", "go", "python", "rust", "java", "sql",
	},
	german: []string{
		"quellcode", "quelltext", "programmcode", "implementierung", "änderung", "änderungen",
		"pull-request", "modul", "module", "funktion", "funktionen", "klasse", "klassen",
		"komponente", "komponenten", "endpunkt", "endpunkte", "codebasis", "fehler", "datei", "dateien",
	},
}

// metaCodeReviewFeatureSignals describe building Wayfinder's own code-review
// support rather than asking for a code review, so they suppress the codeReview
// trait for model-selection work.
var metaCodeReviewFeatureSignals = signalTerms{
	english: []string{
		"code review model selection", "code-review model selection", "code review recommendation", "code-review recommendation",
		"code review task", "code-review task", "code review tasks", "code-review tasks",
		"classify code review", "classify code-review", "route code review", "route code-review",
		"review model family", "review frontier option", "--against", "against gpt", "against claude",
	},
	german: []string{
		"modellauswahl für code-reviews", "modellauswahl für code reviews", "modellauswahl für code-review",
		"code-review-empfehlung", "empfehlung für code-reviews", "empfehlungen für code-reviews",
		"code-review-aufgabe", "code-review-aufgaben",
		"code-reviews klassifizieren", "code-reviews routen", "review-modellfamilie",
		"gegen gpt", "gegen claude",
	},
}

var correctnessHeavyActionSignals = signalTerms{
	english: []string{"compare", "comparison", "ensure", "fix", "handle", "implement", "make", "preserve", "sort", "support", "validate", "behavior"},
	german: []string{
		"vergleiche", "vergleichen", "vergleich",
		"sicherstellen", "gewährleisten",
		"beheben", "behebe", "korrigieren", "korrigiere",
		"behandeln", "behandle",
		"implementieren", "implementiere", "umsetzen",
		"bewahren", "bewahre", "beibehalten",
		"sortieren", "sortiere",
		"unterstützen", "unterstütze",
		"validieren", "validiere", "prüfen", "verhalten",
	},
}

func classify(task string) taskTraits {
	text := strings.ToLower(task)
	correctnessHeavy := correctnessHeavySignals.matches(text)
	modelSelection := modelSelectionSignals.matches(text)
	modelSelectionCoding := modelSelection && modelSelectionCodingActionSignals.matches(text)
	codingIntent := codingIntentSignals.matches(text) || isCorrectnessHeavyCoding(text, correctnessHeavy) || modelSelectionCoding
	coding := codingIntent || codingSignals.matches(text)
	routineCoding := coding && (modelSelection || routineCodingSignals.matches(text))
	return taskTraits{
		simple:           simpleSignals.matches(text),
		coding:           coding,
		codingIntent:     codingIntent,
		codeReview:       isCodeReview(text, coding, modelSelection),
		largeContext:     largeContextSignals.matches(text),
		anthropicFit:     anthropicFitSignals.matches(text),
		visualDesign:     visualDesignSignals.matches(text),
		technicalDesign:  technicalDesignSignals.matches(text),
		nuancedRoutine:   nuancedRoutineSignals.matches(text),
		deepReasoning:    deepReasoningSignals.matches(text),
		highRisk:         highRiskSignals.matches(text),
		correctnessHeavy: correctnessHeavy,
		routineCoding:    routineCoding,
		modelSelection:   modelSelection,
	}
}

func isCodeReview(text string, coding bool, modelSelection bool) bool {
	if modelSelection && metaCodeReviewFeatureSignals.matches(text) {
		return false
	}
	if codeReviewSignals.matches(text) {
		return true
	}
	return coding && reviewActionSignals.matches(text) && reviewObjectSignals.matches(text)
}

func isCorrectnessHeavyCoding(text string, correctnessHeavy bool) bool {
	return correctnessHeavy && correctnessHeavyActionSignals.matches(text)
}

func hasAny(text string, needles ...string) bool {
	for _, needle := range needles {
		if containsTerm(text, needle) {
			return true
		}
	}
	return false
}

// containsTerm reports whether needle occurs in text as a whole word. Matching a
// term inside a longer word would be wrong in both languages ("auth" in
// "author", "regel" in "Regelüberwachung"), so both surrounding characters must
// be boundaries.
func containsTerm(text, needle string) bool {
	needle = strings.TrimSpace(needle)
	if needle == "" {
		return false
	}

	// A rejected occurrence does not rule out a later standalone one, so the
	// scan continues past every match that fails the boundary test.
	for offset := 0; offset <= len(text)-len(needle); {
		index := strings.Index(text[offset:], needle)
		if index < 0 {
			return false
		}

		start := offset + index
		if isBoundaryBefore(text, start) && isBoundaryAfter(text, start+len(needle)) {
			return true
		}

		_, width := utf8.DecodeRuneInString(text[start:])
		offset = start + width
	}
	return false
}

// isBoundaryBefore and isBoundaryAfter decode whole runes rather than single
// bytes, because every continuation byte of a multi-byte character such as ä,
// ö, ü, or ß falls outside any ASCII range and would otherwise be read as a
// word boundary.
func isBoundaryBefore(text string, index int) bool {
	if index <= 0 {
		return true
	}
	r, _ := utf8.DecodeLastRuneInString(text[:index])
	return !isWordRune(r)
}

func isBoundaryAfter(text string, index int) bool {
	if index >= len(text) {
		return true
	}
	r, _ := utf8.DecodeRuneInString(text[index:])
	return !isWordRune(r)
}

// isWordRune treats any Unicode letter or digit as part of a word, so that word
// boundaries are the same for German text as for English.
func isWordRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r)
}
