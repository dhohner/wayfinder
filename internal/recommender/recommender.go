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

// Recommend returns one offline, rules-based recommendation for a natural-language task.
func Recommend(task string) Recommendation {
	traits := classify(task)

	switch {
	case traits.highRisk || traits.deepReasoning:
		return Recommendation{
			Model:            GPT55,
			ReasoningSetting: "GPT reasoning level: high",
			Reason:           "Best fit for complex or high-risk work where stronger reasoning is worth the extra cost.",
		}
	case traits.largeContext || traits.coding:
		return Recommendation{
			Model:            GPT55,
			ReasoningSetting: "GPT reasoning level: low",
			Reason:           "Good balance for development work where stronger model capability at low reasoning is likely cheaper than a medium-effort Sonnet run.",
		}
	case traits.simple:
		return Recommendation{
			Model:            GPT54,
			ReasoningSetting: "GPT reasoning level: low",
			Reason:           "A lower-cost choice is adequate for a simple, low-risk task.",
		}
	default:
		return Recommendation{
			Model:            GPT54,
			ReasoningSetting: "GPT reasoning level: medium",
			Reason:           "Conservative default for an ambiguous task: enough reasoning for unclear work while keeping total token cost below a medium-effort Sonnet run.",
		}
	}
}

// Format renders the v1 human-facing output contract: one model, one setting, one reason.
func Format(rec Recommendation) string {
	return fmt.Sprintf("Model: %s\nReasoning: %s\nReason: %s", rec.Model, rec.ReasoningSetting, rec.Reason)
}

type taskTraits struct {
	simple        bool
	coding        bool
	largeContext  bool
	deepReasoning bool
	highRisk      bool
}

func classify(task string) taskTraits {
	text := strings.ToLower(task)
	return taskTraits{
		simple:        hasAny(text, "summarize", "summary", "rewrite", "proofread", "format", "extract", "release notes", "short email"),
		coding:        hasAny(text, "code", "coding", "implement", "refactor", "debug", "test", "typescript", "golang", "go ", "python", "api", "module", "bug"),
		largeContext:  hasAny(text, "large", "many files", "repository", "repo", "codebase", "migration"),
		deepReasoning: hasAny(text, "architecture", "design", "distributed", "intermittent", "root cause", "plan", "analyze", "tradeoff", "complex"),
		highRisk:      hasAny(text, "security", "auth", "authentication", "payment", "billing", "production", "data loss", "incident", "compliance"),
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
