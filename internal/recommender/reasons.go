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
//
// Where a band's answer may run long, its sentence says so. The simple category
// declares no latency budget and cost mode is never narrowed by one, so simple
// quality, value, and cost, routine cost, and the ambiguous default can answer
// with a row costing several times the steps of the shortest setting in their
// set. The reason line is always shown while the row's tradeoff line needs
// --explain, and the row tradeoff is shared by every category selecting that
// row, so the disclosure belongs here rather than there. It is written in steps,
// never as a claim about measured wall-clock.
//
// Substantive quality and cost also answer with long-running rows and carry no
// such disclosure: substantive declares no latency budget and never promised a
// short answer, so there is no shorter answer for the copy to correct.
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
		english: "Best pass rate among the lowest-credit settings that clear the floor substantive work needs.",
		german:  "Beste Trefferquote unter den Einstellungen mit den niedrigsten Credits, die die für anspruchsvolle Arbeit nötige Untergrenze erreichen.",
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
		english: "Best Claude value for nuanced work: close to its top pass rate for a quarter of the credits.",
		german:  "Bester Claude-Kompromiss für feinsinnige Arbeit: nahe an seiner höchsten Trefferquote für ein Viertel der Credits.",
	},
	reasonAnthropicCost: {
		english: "Best Claude pass rate among the lowest-credit settings that clear the floor nuanced work needs.",
		german:  "Beste Claude-Trefferquote unter den Einstellungen mit den niedrigsten Credits, die die für feinsinnige Arbeit nötige Untergrenze erreichen.",
	},
	reasonAnthropicSpeed: {
		english: "Fewest steps Claude needs on nuanced work while staying close to its top pass rate.",
		german:  "Wenigste Schritte, die Claude für feinsinnige Arbeit braucht, und dabei nahe an seiner höchsten Trefferquote.",
	},

	reasonRoutineQuality: {
		english: "Highest pass rate compatible with the credit budget and quality safeguards for routine work.",
		german:  "Höchste Trefferquote im Einklang mit dem Credit-Budget und den Qualitätsvorgaben für Routinearbeit.",
	},
	reasonRoutineValue: {
		english: "Best value within the credit budget and safeguards for routine work.",
		german:  "Bester Kompromiss innerhalb des Credit-Budgets und der Vorgaben für Routinearbeit.",
	},
	reasonRoutineCost: {
		english: "Best pass rate among the lowest-credit choices inside the routine credit limit, and it runs well past the step count routine work usually keeps to.",
		german:  "Beste Trefferquote unter den günstigsten Optionen innerhalb der Credit-Grenze für Routinearbeit, und sie läuft deutlich über die Schrittzahl hinaus, die Routinearbeit sonst einhält.",
	},
	reasonRoutineSpeed: {
		english: "Fewest steps inside the routine limits while staying within a few points of the top pass rate.",
		german:  "Wenigste Schritte innerhalb der Routinegrenzen und dabei nur wenige Punkte unter der höchsten Trefferquote.",
	},

	reasonSimpleQuality: {
		english: "Highest pass rate that keeps a simple, low-risk task cheap, and it takes noticeably more steps than a simple task usually needs.",
		german:  "Höchste Trefferquote, die eine einfache, risikoarme Aufgabe günstig hält, und sie braucht deutlich mehr Schritte, als eine einfache Aufgabe sonst benötigt.",
	},
	reasonSimpleValue: {
		english: "Best value for a simple task: the best pass rate among the lowest-credit choices near the top score, and it takes noticeably more steps than a simple task usually needs.",
		german:  "Bester Kompromiss für eine einfache Aufgabe: die beste Trefferquote unter den günstigsten Optionen nahe der höchsten Trefferquote, und sie braucht deutlich mehr Schritte, als eine einfache Aufgabe sonst benötigt.",
	},
	reasonSimpleCost: {
		english: "Best pass rate among the lowest-credit choices that clear the floor for a simple, low-risk task, and it trades a longer run for the lower credit cost.",
		german:  "Beste Trefferquote unter den günstigsten Optionen, die die Untergrenze für eine einfache, risikoarme Aufgabe erreichen, und sie erkauft die geringeren Credit-Kosten mit einem längeren Lauf.",
	},
	reasonSimpleSpeed: {
		english: "Fewest steps for a simple task while staying close to the top pass rate.",
		german:  "Wenigste Schritte für eine einfache Aufgabe und dabei nahe an der höchsten Trefferquote.",
	},

	reasonAmbiguousDefault: {
		english: "Conservative default for an ambiguous task: the best pass rate among the lowest-credit routine choices, trading a longer run for the lower credit cost.",
		german:  "Konservative Voreinstellung für eine unklare Aufgabe: Sie bietet die beste Trefferquote unter den Routineoptionen mit den niedrigsten Credits und nimmt für die geringeren Credit-Kosten einen längeren Lauf in Kauf.",
	},
}

func reasonText(id reasonID, lang language) string {
	text, ok := reasonCopy[id]
	if !ok {
		panic(fmt.Sprintf("recommender: no reason copy registered for %q", id))
	}
	return text.in(lang)
}
