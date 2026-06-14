package recommender

import (
	"fmt"
	"strings"
)

const (
	// GPT54 and Sonnet46 remain supported for provider classification, but built-in
	// recommendations prefer newer defaults unless a future rule explicitly opts in.
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
// Preferences are soft hints: they influence the choice only when the task traits
// make that bias appropriate, and they never downgrade high-risk or complex work.
func RecommendWithPreference(task string, preference Preference) Recommendation {
	traits := classify(task)

	for _, rule := range recommendationRules {
		if rule.matches(traits) {
			return rule.recommend(preference)
		}
	}

	return gptRecommendation(GPT55, "medium", "Conservative default for an ambiguous task: enough reasoning for unclear work.")
}

type recommendationRule struct {
	matches   func(taskTraits) bool
	recommend func(Preference) Recommendation
}

var recommendationRules = []recommendationRule{
	{matches: func(traits taskTraits) bool { return traits.highRisk }, recommend: recommendHighRisk},
	{matches: func(traits taskTraits) bool { return traits.deepReasoning }, recommend: recommendDeepReasoning},
	{matches: func(traits taskTraits) bool { return traits.anthropicFit || traits.visualDesign }, recommend: recommendAnthropicFit},
	{matches: func(traits taskTraits) bool { return traits.nuancedRoutine && !traits.coding }, recommend: recommendNuancedRoutine},
	{matches: func(traits taskTraits) bool { return traits.simple && !traits.largeContext }, recommend: recommendSimple},
	{matches: func(traits taskTraits) bool { return traits.largeContext || traits.coding }, recommend: recommendDevelopmentOrLargeContext},
}

func recommendHighRisk(preference Preference) Recommendation {
	if preference == PreferQuality {
		return gptRecommendation(GPT55, "xhigh", "Best fit for high-risk work where maximum reasoning quality is worth the extra cost.")
	}
	return gptRecommendation(GPT55, "high", "Best fit for high-risk work where stronger reasoning is worth the extra cost; preference does not override the task's risk.")
}

func recommendDeepReasoning(preference Preference) Recommendation {
	if preference == PreferQuality {
		return gptRecommendation(GPT55, "xhigh", "Best fit for complex work where maximum reasoning quality is worth the extra cost.")
	}
	return gptRecommendation(GPT55, "high", "Best fit for complex work where stronger reasoning is worth the extra cost; preference does not override the task's complexity.")
}

func recommendAnthropicFit(preference Preference) Recommendation {
	switch preference {
	case PreferQuality:
		return anthropicRecommendation(Opus48, "high", "Quality preference chooses Opus for long-form, creative, or visual design work where nuance matters.")
	case PreferCost, PreferSpeed:
		return gptRecommendation(GPT55, "medium", "Cost or speed preference favors the stronger default while keeping enough reasoning for nuanced work.")
	default:
		return anthropicRecommendation(Opus48, "medium", "Good fit for long-form, creative, or visual design work where Opus capability is useful.")
	}
}

func recommendNuancedRoutine(preference Preference) Recommendation {
	if preference == PreferQuality {
		return gptRecommendation(GPT55, "high", "Quality preference adds reasoning depth for a nuanced routine task.")
	}
	return gptRecommendation(GPT55, "medium", "Good default for messy input, nuance, or multiple simple constraints.")
}

func recommendSimple(Preference) Recommendation {
	return gptRecommendation(GPT55, "low", "A fast, lower-reasoning GPT 5.5 choice is adequate for a simple, low-risk task.")
}

func recommendDevelopmentOrLargeContext(preference Preference) Recommendation {
	if preference == PreferQuality {
		return gptRecommendation(GPT55, "high", "Quality preference raises reasoning for development work while avoiding the maximum-cost setting.")
	}
	return gptRecommendation(GPT55, "medium", "Good default for moderate development or larger-context work.")
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
	visualDesign   bool
	nuancedRoutine bool
	deepReasoning  bool
	highRisk       bool
}

func classify(task string) taskTraits {
	text := strings.ToLower(task)
	return taskTraits{
		simple:         hasAny(text, "summarize", "summary", "rewrite", "proofread", "format", "extract", "release notes", "short email", "typo", "readme", "rename", "one-line", "small", "minor", "lint", "comment", "documentation"),
		coding:         hasAny(text, "code", "coding", "implement", "refactor", "debug", "test", "typescript", "golang", "go", "python", "api", "module", "bug", "endpoint", "function"),
		largeContext:   hasAny(text, "large", "long", "many files", "repository", "repo", "codebase", "migration", "cross-service", "multi-service", "10-page", "10 page"),
		anthropicFit:   hasAny(text, "long document", "long-form", "longform", "essay", "narrative", "manuscript", "policy brief", "research brief", "research report", "market analysis", "literature review", "creative writing", "story", "tone", "voice"),
		visualDesign:   hasAny(text, "visual design", "ui design", "ux design", "interface design", "interaction design", "design system", "mockup", "wireframe", "prototype", "layout", "typography", "color palette", "brand", "branding"),
		nuancedRoutine: hasAny(text, "messy", "inconsistent", "nuanced", "firm but empathetic", "preserve intent", "multiple constraints", "overlap", "overlapping", "edge case", "requirements", "product request", "project plan", "meeting notes", "support reply", "policy"),
		deepReasoning:  hasAny(text, "architecture", "system design", "distributed", "intermittent", "root cause", "tradeoff", "complex", "race condition", "concurrency", "performance", "scalability"),
		highRisk:       hasAny(text, "security", "auth", "authentication", "payment", "billing", "production", "data loss", "incident", "compliance", "privacy", "encryption", "permissions"),
	}
}

func hasAny(text string, needles ...string) bool {
	for _, needle := range needles {
		if containsTerm(text, needle) {
			return true
		}
	}
	return false
}

func containsTerm(text, needle string) bool {
	needle = strings.TrimSpace(needle)
	if needle == "" {
		return false
	}

	for start := strings.Index(text, needle); start >= 0; {
		end := start + len(needle)
		if hasBoundary(text, start-1) && hasBoundary(text, end) {
			return true
		}

		next := strings.Index(text[start+1:], needle)
		if next < 0 {
			return false
		}
		start += next + 1
	}
	return false
}

func hasBoundary(text string, index int) bool {
	if index < 0 || index >= len(text) {
		return true
	}
	c := text[index]
	return !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9'))
}
