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

type providerFamily string

const (
	providerUnknown   providerFamily = ""
	providerGPT       providerFamily = "gpt"
	providerAnthropic providerFamily = "anthropic"
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
	case traits.highRisk:
		return gptRecommendation(GPT55, "high", "Best fit for high-risk work where stronger reasoning is worth the extra cost; preference does not override the task's risk.")
	case traits.anthropicFit && traits.deepReasoning:
		return anthropicRecommendation(Opus48, "high", "Best fit for demanding long-form analysis where Opus capability and a higher Effort Level are worth the extra cost.")
	case traits.deepReasoning:
		return gptRecommendation(GPT55, "high", "Best fit for complex work where stronger reasoning is worth the extra cost; preference does not override the task's complexity.")
	case traits.anthropicFit:
		switch preference {
		case PreferQuality:
			return anthropicRecommendation(Opus48, "medium", "Quality preference chooses Opus for long-form work with enough depth for nuance.")
		case PreferCost, PreferSpeed:
			return anthropicRecommendation(Sonnet46, "low", "The task looks long-form but low-risk, so Sonnet favors cost and speed.")
		default:
			return anthropicRecommendation(Sonnet46, "medium", "Good fit for long-form, low-risk work where Sonnet balances quality and cost.")
		}
	case traits.nuancedRoutine && !traits.coding:
		switch preference {
		case PreferQuality:
			return gptRecommendation(GPT55, "medium", "Quality preference adds reasoning depth for a nuanced routine task while avoiding the highest-cost setting.")
		case PreferCost:
			return gptRecommendation(GPT54, "medium", "Cost preference uses a lower-cost model with enough reasoning for messy but low-risk work.")
		default:
			return gptRecommendation(GPT55, "low", "Good fit for lightweight work with messy input, nuance, or multiple simple constraints.")
		}
	case traits.simple && !traits.largeContext:
		switch preference {
		case PreferQuality:
			return gptRecommendation(GPT54, "medium", "Quality preference adds reasoning depth, but the simple, low-risk task does not justify a highest-cost model.")
		default:
			return gptRecommendation(GPT54, "low", "A lower-cost, fast choice is adequate for a simple, low-risk task.")
		}
	case traits.largeContext || traits.coding:
		switch preference {
		case PreferQuality:
			return gptRecommendation(GPT55, "medium", "Quality preference raises reasoning for development work while avoiding the highest-cost setting.")
		case PreferCost:
			return gptRecommendation(GPT54, "medium", "Cost preference chooses a lower-cost model because the task looks moderate rather than high-risk.")
		case PreferSpeed:
			return gptRecommendation(GPT55, "low", "Speed preference keeps stronger coding capability while avoiding higher reasoning because the task is not high-risk or deeply complex.")
		default:
			return gptRecommendation(GPT55, "low", "Good balance for development work where stronger model capability at low reasoning is likely cheaper than a medium-reasoning run.")
		}
	default:
		switch preference {
		case PreferSpeed, PreferCost:
			return gptRecommendation(GPT54, "low", "The task is ambiguous but not complex, so the preference favors a lower-cost, faster setting.")
		default:
			return gptRecommendation(GPT54, "medium", "Conservative default for an ambiguous task: enough reasoning for unclear work.")
		}
	}
}

// Format renders the v1 human-facing output contract: one model, one setting, one reason.
func Format(rec Recommendation) string {
	return fmt.Sprintf("Model: %s\nReasoning: %s\nReason: %s", rec.Model, rec.ReasoningSetting, rec.Reason)
}

func gptRecommendation(model, level, reason string) Recommendation {
	return Recommendation{
		Model:            model,
		ReasoningSetting: "GPT reasoning level: " + level,
		Reason:           reason,
	}
}

func anthropicRecommendation(model, level, reason string) Recommendation {
	return Recommendation{
		Model:            model,
		ReasoningSetting: "Anthropic Effort Level: " + level,
		Reason:           reason,
	}
}

func providerForModel(model string) providerFamily {
	switch model {
	case GPT54, GPT55:
		return providerGPT
	case Opus48, Sonnet46:
		return providerAnthropic
	default:
		return providerUnknown
	}
}

type taskTraits struct {
	simple         bool
	coding         bool
	largeContext   bool
	anthropicFit   bool
	nuancedRoutine bool
	deepReasoning  bool
	highRisk       bool
}

func classify(task string) taskTraits {
	text := strings.ToLower(task)
	return taskTraits{
		simple:         hasAny(text, "summarize", "summary", "rewrite", "proofread", "format", "extract", "release notes", "short email", "typo", "readme", "rename", "one-line", "small", "minor", "lint", "comment", "documentation"),
		coding:         hasAny(text, "code", "coding", "implement", "refactor", "debug", "test", "typescript", "golang", "go ", "python", "api", "module", "bug", "endpoint", "function"),
		largeContext:   hasAny(text, "large", "long", "many files", "repository", "repo", "codebase", "migration", "cross-service", "multi-service", "10-page", "10 page"),
		anthropicFit:   hasAny(text, "long document", "long-form", "longform", "essay", "narrative", "manuscript", "policy brief", "research brief", "research report", "market analysis", "literature review", "creative writing", "story", "tone", "voice"),
		nuancedRoutine: hasAny(text, "messy", "inconsistent", "nuanced", "firm but empathetic", "preserve intent", "multiple constraints", "overlap", "overlapping", "edge case", "requirements", "product request", "project plan", "meeting notes", "support reply", "policy"),
		deepReasoning:  hasAny(text, "architecture", "system design", "distributed", "intermittent", "root cause", "tradeoff", "complex", "race condition", "concurrency", "performance", "scalability"),
		highRisk:       hasAny(text, "security", "auth", "authentication", "payment", "billing", "production", "data loss", "incident", "compliance", "privacy", "encryption", "permissions"),
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
