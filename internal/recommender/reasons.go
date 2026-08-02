package recommender

import "fmt"

// reasonID addresses one recommendation reason. Reasons are looked up by
// identifier rather than written inline at their call site so that the copy
// lives in one table and can be translated without touching selection logic.
// Identifiers are internal: they never appear in text or JSON output and are
// never localized.
type reasonID string

const (
	reasonSubstantiveQuality reasonID = "substantive.quality"
	reasonSubstantiveValue   reasonID = "substantive.value"
	reasonSubstantiveCost    reasonID = "substantive.cost"
	reasonSubstantiveSpeed   reasonID = "substantive.speed"

	reasonAnthropicQuality reasonID = "anthropic.quality"
	reasonAnthropicValue   reasonID = "anthropic.value"
	reasonAnthropicCost    reasonID = "anthropic.cost"
	reasonAnthropicSpeed   reasonID = "anthropic.speed"

	reasonRoutineQuality reasonID = "routine.quality"
	reasonRoutineValue   reasonID = "routine.value"
	reasonRoutineCost    reasonID = "routine.cost"
	reasonRoutineSpeed   reasonID = "routine.speed"

	reasonSimpleQuality reasonID = "simple.quality"
	reasonSimpleValue   reasonID = "simple.value"
	reasonSimpleCost    reasonID = "simple.cost"
	reasonSimpleSpeed   reasonID = "simple.speed"

	reasonAmbiguousDefault reasonID = "ambiguous.default"
)

// allReasonIDs is every registered reason identifier. It exists so that
// translation completeness is checked against the identifier set itself rather
// than against one language's table, which would make a missing entry
// undetectable in exactly the direction that matters.
var allReasonIDs = []reasonID{
	reasonSubstantiveQuality, reasonSubstantiveValue, reasonSubstantiveCost, reasonSubstantiveSpeed,
	reasonAnthropicQuality, reasonAnthropicValue, reasonAnthropicCost, reasonAnthropicSpeed,
	reasonRoutineQuality, reasonRoutineValue, reasonRoutineCost, reasonRoutineSpeed,
	reasonSimpleQuality, reasonSimpleValue, reasonSimpleCost, reasonSimpleSpeed,
	reasonAmbiguousDefault,
}

// reasonCopy is the only place recommendation reason copy is written, in every
// supported language.
//
// Each sentence explains why its band chose the option it did. Copy must stay
// inside the human-only output guardrails in both languages: it may describe
// credit cost qualitatively but must never print a price, it must avoid ranking
// and alternatives language, and speed copy describes step counts rather than
// claiming a measured latency result, because steps are the bundled proxy for
// latency and not a latency measurement. German copy additionally avoids
// "Effort", which is Anthropic's terminology and must not surface in a GPT
// recommendation.
var reasonCopy = map[reasonID]localizedText{
	reasonSubstantiveQuality: {
		english: "Highest pass rate available for substantive work, chosen for quality despite a high credit estimate.",
		german:  "Höchste verfügbare Trefferquote für anspruchsvolle Arbeit, wegen der Qualität gewählt trotz hoher geschätzter Credits.",
	},
	reasonSubstantiveValue: {
		english: "Best value for substantive work: within a few points of the top pass rate at a fraction of the credits.",
		german:  "Bester Kompromiss für anspruchsvolle Arbeit: wenige Punkte unter der höchsten Trefferquote bei einem Bruchteil der Credits.",
	},
	reasonSubstantiveCost: {
		english: "Cheapest choice that still clears the pass rate substantive work needs.",
		german:  "Günstigste Wahl, die die für anspruchsvolle Arbeit nötige Trefferquote noch erreicht.",
	},
	reasonSubstantiveSpeed: {
		english: "Fewest steps for substantive work while staying within a few points of the top pass rate.",
		german:  "Wenigste Schritte für anspruchsvolle Arbeit und dabei nur wenige Punkte unter der höchsten Trefferquote.",
	},

	reasonAnthropicQuality: {
		english: "Highest pass rate Claude reaches on nuanced work, chosen for quality even though it uses the most credits.",
		german:  "Höchste Trefferquote, die Claude bei feinsinniger Arbeit erreicht, wegen der Qualität gewählt, obwohl sie die meisten Credits verbraucht.",
	},
	reasonAnthropicValue: {
		english: "Best Claude value for nuanced work: close to its top pass rate for a third of the credits.",
		german:  "Bester Claude-Kompromiss für feinsinnige Arbeit: nahe an seiner höchsten Trefferquote für ein Drittel der Credits.",
	},
	reasonAnthropicCost: {
		english: "Cheapest Claude setting that still clears the pass rate nuanced work needs.",
		german:  "Günstigste Claude-Einstellung, die die für feinsinnige Arbeit nötige Trefferquote noch erreicht.",
	},
	reasonAnthropicSpeed: {
		english: "Fewest steps Claude needs on nuanced work while staying close to its top pass rate.",
		german:  "Wenigste Schritte, die Claude für feinsinnige Arbeit braucht, und dabei nahe an seiner höchsten Trefferquote.",
	},

	reasonRoutineQuality: {
		english: "Highest pass rate that stays inside the credit and step limits kept for routine work.",
		german:  "Höchste Trefferquote innerhalb der Credit- und Schrittgrenzen, die für Routinearbeit gelten.",
	},
	reasonRoutineValue: {
		english: "Best value for routine work: the top pass rate available inside its credit and step limits.",
		german:  "Bester Kompromiss für Routinearbeit: die höchste Trefferquote innerhalb ihrer Credit- und Schrittgrenzen.",
	},
	reasonRoutineCost: {
		english: "Cheapest choice inside the routine limits that still clears the pass rate routine work needs.",
		german:  "Günstigste Wahl innerhalb der Routinegrenzen, die die für Routinearbeit nötige Trefferquote noch erreicht.",
	},
	reasonRoutineSpeed: {
		english: "Fewest steps inside the routine limits while staying within a few points of the top pass rate.",
		german:  "Wenigste Schritte innerhalb der Routinegrenzen und dabei nur wenige Punkte unter der höchsten Trefferquote.",
	},

	reasonSimpleQuality: {
		english: "Highest pass rate that keeps a simple, low-risk task cheap and short.",
		german:  "Höchste Trefferquote, die eine einfache, risikoarme Aufgabe günstig und kurz hält.",
	},
	reasonSimpleValue: {
		english: "Best value for a simple task: the cheapest choice within a few points of the top pass rate.",
		german:  "Bester Kompromiss für eine einfache Aufgabe: die günstigste Wahl wenige Punkte unter der höchsten Trefferquote.",
	},
	reasonSimpleCost: {
		english: "Cheapest choice that still clears the pass rate a simple, low-risk task needs.",
		german:  "Günstigste Wahl, die die für eine einfache, risikoarme Aufgabe nötige Trefferquote noch erreicht.",
	},
	reasonSimpleSpeed: {
		english: "Fewest steps for a simple task while staying close to the top pass rate.",
		german:  "Wenigste Schritte für eine einfache Aufgabe und dabei nahe an der höchsten Trefferquote.",
	},

	reasonAmbiguousDefault: {
		english: "Conservative default for an ambiguous task: strong value with enough reasoning for unclear work.",
		german:  "Konservative Voreinstellung für eine unklare Aufgabe: guter Kompromiss mit genug Reasoning für unklare Arbeit.",
	},
}

func reasonText(id reasonID, lang language) string {
	text, ok := reasonCopy[id]
	if !ok {
		panic(fmt.Sprintf("recommender: no reason copy registered for %q", id))
	}
	return text.in(lang)
}
