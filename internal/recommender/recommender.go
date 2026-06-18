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

// Optimization is the recommendation mode requested by the developer.
type Optimization string

const (
	OptimizeValue   Optimization = "value"
	OptimizeQuality Optimization = "quality"
	OptimizeCost    Optimization = "cost"
	OptimizeSpeed   Optimization = "speed"
)

// ParseOptimization validates a --optimize value.
func ParseOptimization(value string) (Optimization, bool) {
	switch Optimization(strings.ToLower(strings.TrimSpace(value))) {
	case OptimizeValue:
		return OptimizeValue, true
	case OptimizeQuality:
		return OptimizeQuality, true
	case OptimizeCost:
		return OptimizeCost, true
	case OptimizeSpeed:
		return OptimizeSpeed, true
	default:
		return OptimizeValue, false
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

// RecommendWithOptimization returns one recommendation optimized for value, quality, cost, or speed.
func RecommendWithOptimization(task string, optimization Optimization) Recommendation {
	return NewService().RecommendWithOptimization(task, optimization)
}

// Recommend returns one offline, rules-based recommendation for a natural-language task.
func (s Service) Recommend(task string) Recommendation {
	return s.RecommendWithOptimization(task, OptimizeValue)
}

// RecommendWithOptimization returns one recommendation for the requested optimization mode.
func (s Service) RecommendWithOptimization(task string, optimization Optimization) Recommendation {
	traits := classify(task)
	rules := s.rules
	if rules == nil {
		rules = defaultRules
	}

	for _, rule := range rules {
		if rule.matches(traits) {
			return rule.recommend(optimization, traits)
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
