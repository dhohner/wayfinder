package recommender

import (
	"fmt"
	"strings"
)

const (
	GPT54    = "GPT 5.4"
	GPT55    = "GPT 5.5"
	Opus48   = "Opus 4.8"
	Sonnet46 = "Sonnet 4.6"
)

// Recommendation is the single model choice Wayfinder returns for a task.
type Recommendation struct {
	Model            string
	ReasoningSetting string
	Reason           string
}

// Preference is a simple recommendation bias requested by the developer.
type Preference string

const (
	PreferNone    Preference = ""
	PreferQuality Preference = "quality"
	PreferCost    Preference = "cost"
	PreferSpeed   Preference = "speed"
)

// ParsePreference validates a --prefer value.
func ParsePreference(value string) (Preference, bool) {
	switch Preference(strings.ToLower(strings.TrimSpace(value))) {
	case PreferQuality:
		return PreferQuality, true
	case PreferCost:
		return PreferCost, true
	case PreferSpeed:
		return PreferSpeed, true
	default:
		return PreferNone, false
	}
}

// Recommend returns one offline, rules-based recommendation for a natural-language task.
func Recommend(task string) Recommendation {
	return RecommendWithPreference(task, PreferNone)
}

// RecommendWithPreference returns one recommendation, optionally biased toward quality, cost, or speed.
// Preferences influence the choice only when the task traits make that bias appropriate.
func RecommendWithPreference(task string, preference Preference) Recommendation {
	traits := classify(task)

	switch {
	case traits.highRisk || traits.deepReasoning:
		return Recommendation{
			Model:            GPT55,
			ReasoningSetting: "GPT reasoning level: high",
			Reason:           "Best fit for complex or high-risk work where stronger reasoning is worth the extra cost; preference does not override the task's risk.",
		}
	case traits.nuancedRoutine && !traits.coding:
		switch preference {
		case PreferQuality:
			return Recommendation{Model: GPT55, ReasoningSetting: "GPT reasoning level: medium", Reason: "Quality preference adds reasoning depth for a nuanced routine task while avoiding the highest-cost setting."}
		case PreferCost:
			return Recommendation{Model: GPT54, ReasoningSetting: "GPT reasoning level: medium", Reason: "Cost preference uses a lower-cost model with enough reasoning for messy but low-risk work."}
		default:
			return Recommendation{Model: GPT55, ReasoningSetting: "GPT reasoning level: low", Reason: "Good fit for lightweight work with messy input, nuance, or multiple simple constraints."}
		}
	case traits.simple && !traits.largeContext:
		switch preference {
		case PreferQuality:
			return Recommendation{Model: GPT54, ReasoningSetting: "GPT reasoning level: medium", Reason: "Quality preference adds reasoning depth, but the simple, low-risk task does not justify a highest-cost model."}
		default:
			return Recommendation{Model: GPT54, ReasoningSetting: "GPT reasoning level: low", Reason: "A lower-cost, fast choice is adequate for a simple, low-risk task."}
		}
	case traits.largeContext || traits.coding:
		switch preference {
		case PreferQuality:
			return Recommendation{Model: GPT55, ReasoningSetting: "GPT reasoning level: medium", Reason: "Quality preference raises reasoning for development work while avoiding the highest-cost setting."}
		case PreferCost:
			return Recommendation{Model: GPT54, ReasoningSetting: "GPT reasoning level: medium", Reason: "Cost preference chooses a lower-cost model because the task looks moderate rather than high-risk."}
		case PreferSpeed:
			return Recommendation{Model: GPT55, ReasoningSetting: "GPT reasoning level: low", Reason: "Speed preference keeps stronger coding capability while avoiding higher reasoning because the task is not high-risk or deeply complex."}
		default:
			return Recommendation{Model: GPT55, ReasoningSetting: "GPT reasoning level: low", Reason: "Good balance for development work where stronger model capability at low reasoning is likely cheaper than a medium-effort run."}
		}
	default:
		switch preference {
		case PreferSpeed, PreferCost:
			return Recommendation{Model: GPT54, ReasoningSetting: "GPT reasoning level: low", Reason: "The task is ambiguous but not complex, so the preference favors a lower-cost, faster setting."}
		default:
			return Recommendation{Model: GPT54, ReasoningSetting: "GPT reasoning level: medium", Reason: "Conservative default for an ambiguous task: enough reasoning for unclear work."}
		}
	}
}

// Format renders the v1 human-facing output contract: one model, one setting, one reason.
func Format(rec Recommendation) string {
	return fmt.Sprintf("Model: %s\nReasoning: %s\nReason: %s", rec.Model, rec.ReasoningSetting, rec.Reason)
}

type taskTraits struct {
	simple         bool
	coding         bool
	largeContext   bool
	nuancedRoutine bool
	deepReasoning  bool
	highRisk       bool
}

func classify(task string) taskTraits {
	text := strings.ToLower(task)
	return taskTraits{
		simple:        hasAny(text, "summarize", "summary", "rewrite", "proofread", "format", "extract", "release notes", "short email", "typo", "readme", "rename", "one-line", "small", "minor", "lint", "comment", "documentation"),
		coding:        hasAny(text, "code", "coding", "implement", "refactor", "debug", "test", "typescript", "golang", "go ", "python", "api", "module", "bug", "endpoint", "function"),
		largeContext:   hasAny(text, "large", "long", "many files", "repository", "repo", "codebase", "migration", "cross-service", "multi-service", "10-page", "10 page"),
		nuancedRoutine: hasAny(text, "messy", "inconsistent", "nuanced", "firm but empathetic", "preserve intent", "multiple constraints", "overlap", "overlapping", "edge case", "requirements", "product request", "project plan", "meeting notes", "support reply", "policy"),
		deepReasoning:  hasAny(text, "architecture", "system design", "distributed", "intermittent", "root cause", "tradeoff", "complex", "race condition", "concurrency", "performance", "scalability"),
		highRisk:      hasAny(text, "security", "auth", "authentication", "payment", "billing", "production", "data loss", "incident", "compliance", "privacy", "encryption", "permissions"),
	}
}

func hasAny(text string, needles ...string) bool {
	for _, needle := range needles {
		if strings.Contains(text, needle) {
			return true
		}
	}
	return false
}
