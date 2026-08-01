package recommender

import "fmt"

// reasonID addresses one recommendation reason. Reasons are looked up by
// identifier rather than written inline at their call site so that the English
// text lives in one table and can be translated without touching selection
// logic. Identifiers are internal: they never appear in text or JSON output and
// are never localized.
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

// englishReasons is the only place recommendation reason copy is written.
//
// Each sentence explains why its band chose the option it did. Copy must stay
// inside the human-only output guardrails: it may describe credit cost
// qualitatively but must never print a price, and speed copy describes step
// counts rather than claiming a measured latency result, because steps are the
// bundled proxy for latency and not a latency measurement.
var englishReasons = map[reasonID]string{
	reasonSubstantiveQuality: "Highest pass rate available for substantive work, chosen for quality despite a high credit estimate.",
	reasonSubstantiveValue:   "Best value for substantive work: within a few points of the top pass rate at a fraction of the credits.",
	reasonSubstantiveCost:    "Cheapest choice that still clears the pass rate substantive work needs.",
	reasonSubstantiveSpeed:   "Fewest steps for substantive work while staying within a few points of the top pass rate.",

	reasonAnthropicQuality: "Highest pass rate Claude reaches on nuanced work, chosen for quality even though it uses the most credits.",
	reasonAnthropicValue:   "Best Claude value for nuanced work: close to its top pass rate for a third of the credits.",
	reasonAnthropicCost:    "Cheapest Claude setting that still clears the pass rate nuanced work needs.",
	reasonAnthropicSpeed:   "Fewest steps Claude needs on nuanced work while staying close to its top pass rate.",

	reasonRoutineQuality: "Highest pass rate that stays inside the credit and step limits kept for routine work.",
	reasonRoutineValue:   "Best value for routine work: the top pass rate available inside its credit and step limits.",
	reasonRoutineCost:    "Cheapest choice inside the routine limits that still clears the pass rate routine work needs.",
	reasonRoutineSpeed:   "Fewest steps inside the routine limits while staying within a few points of the top pass rate.",

	reasonSimpleQuality: "Highest pass rate that keeps a simple, low-risk task cheap and short.",
	reasonSimpleValue:   "Best value for a simple task: the cheapest choice within a few points of the top pass rate.",
	reasonSimpleCost:    "Cheapest choice that still clears the pass rate a simple, low-risk task needs.",
	reasonSimpleSpeed:   "Fewest steps for a simple task while staying close to the top pass rate.",

	reasonAmbiguousDefault: "Conservative default for an ambiguous task: strong value with enough reasoning for unclear work.",
}

func reasonText(id reasonID) string {
	text, ok := englishReasons[id]
	if !ok {
		panic(fmt.Sprintf("recommender: no reason text registered for %q", id))
	}
	return text
}
