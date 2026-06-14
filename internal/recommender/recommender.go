package recommender

import "strings"

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

// Service evaluates task descriptions against configured recommendation rules.
// It is immutable after construction and safe to reuse across requests.
type Service struct {
	rules                 []recommendationRule
	defaultRecommendation Recommendation
}

// NewService returns a recommender using Wayfinder's bundled offline rules.
func NewService() Service {
	return Service{
		rules:                 append([]recommendationRule(nil), defaultRules...),
		defaultRecommendation: defaultRecommendation(),
	}
}

// Recommend returns one offline, rules-based recommendation for a natural-language task.
func Recommend(task string) Recommendation {
	return NewService().Recommend(task)
}

// RecommendWithPreference returns one recommendation, optionally biased toward quality, cost, or speed.
func RecommendWithPreference(task string, preference Preference) Recommendation {
	return NewService().RecommendWithPreference(task, preference)
}

// Recommend returns one offline, rules-based recommendation for a natural-language task.
func (s Service) Recommend(task string) Recommendation {
	return s.RecommendWithPreference(task, PreferNone)
}

// RecommendWithPreference returns one recommendation, optionally biased toward quality, cost, or speed.
// Preferences are soft hints: they influence the choice only when the task traits
// make that bias appropriate, and they never downgrade high-risk or complex work.
func (s Service) RecommendWithPreference(task string, preference Preference) Recommendation {
	traits := classify(task)
	rules := s.rules
	if rules == nil {
		rules = defaultRules
	}

	for _, rule := range rules {
		if rule.matches(traits) {
			return rule.recommend(preference)
		}
	}

	if s.defaultRecommendation == (Recommendation{}) {
		return defaultRecommendation()
	}
	return s.defaultRecommendation
}

func defaultRecommendation() Recommendation {
	return gptRecommendation(GPT55, "medium", "Conservative default for an ambiguous task: enough reasoning for unclear work.")
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
