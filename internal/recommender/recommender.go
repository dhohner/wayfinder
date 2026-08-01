package recommender

import "strings"

const (
	// Older models remain supported for compatibility, but bundled rules select
	// the current benchmark leaders.
	GPT54      = "GPT 5.4"
	GPT55      = "GPT 5.5"
	GPT56Sol   = "GPT 5.6 Sol"
	GPT56Luna  = "GPT 5.6 Luna"
	GPT56Terra = "GPT 5.6 Terra"
	Opus48     = "Opus 4.8"
	Opus5      = "Opus 5"
	Fable5     = "Fable 5"
	Sonnet46   = "Sonnet 4.6"
	Sonnet5    = "Sonnet 5"
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

// AgainstFamily identifies the model family that authored code under review.
// It is only used for tasks classified as code review.
type AgainstFamily string

const (
	AgainstUnspecified AgainstFamily = ""
	AgainstGPT         AgainstFamily = "gpt"
	AgainstClaude      AgainstFamily = "claude"
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

// ParseAgainstFamily validates a --against value.
func ParseAgainstFamily(value string) (AgainstFamily, bool) {
	switch AgainstFamily(strings.ToLower(strings.TrimSpace(value))) {
	case AgainstGPT:
		return AgainstGPT, true
	case AgainstClaude:
		return AgainstClaude, true
	default:
		return AgainstUnspecified, false
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

// RecommendWithOptimizationAgainst returns one recommendation, using --against only for code-review tasks.
func RecommendWithOptimizationAgainst(task string, optimization Optimization, against AgainstFamily) Recommendation {
	return NewService().RecommendWithOptimizationAgainst(task, optimization, against)
}

// Recommend returns one offline, rules-based recommendation for a natural-language task.
func (s Service) Recommend(task string) Recommendation {
	return s.RecommendWithOptimization(task, OptimizeValue)
}

// RecommendWithOptimization returns one recommendation for the requested optimization mode.
func (s Service) RecommendWithOptimization(task string, optimization Optimization) Recommendation {
	return s.RecommendWithOptimizationAgainst(task, optimization, AgainstUnspecified)
}

// RecommendWithOptimizationAgainst returns one recommendation for the requested optimization mode.
// The against family only affects tasks classified as code review.
func (s Service) RecommendWithOptimizationAgainst(task string, optimization Optimization, against AgainstFamily) Recommendation {
	traits := classify(task)
	traits.against = against
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
	return gptRecommendation(GPT56Sol, "medium", "Conservative default for an ambiguous task: strong value with enough reasoning for unclear work.")
}

type modelInfo struct {
	benchmarkID string
	provider    providerFamily
}

func modelInfoFor(model string) (modelInfo, bool) {
	switch model {
	case GPT54:
		return modelInfo{benchmarkID: "gpt-5.4", provider: providerGPT}, true
	case GPT55:
		return modelInfo{benchmarkID: "gpt-5.5", provider: providerGPT}, true
	case GPT56Sol:
		return modelInfo{benchmarkID: "gpt-5.6-sol", provider: providerGPT}, true
	case GPT56Luna:
		return modelInfo{benchmarkID: "gpt-5.6-luna", provider: providerGPT}, true
	case GPT56Terra:
		return modelInfo{benchmarkID: "gpt-5.6-terra", provider: providerGPT}, true
	case Opus48:
		return modelInfo{benchmarkID: "claude-opus-4.8", provider: providerAnthropic}, true
	case Opus5:
		return modelInfo{benchmarkID: "claude-opus-5", provider: providerAnthropic}, true
	case Fable5:
		return modelInfo{benchmarkID: "claude-fable-5", provider: providerAnthropic}, true
	case Sonnet46:
		return modelInfo{benchmarkID: "claude-sonnet-4.6", provider: providerAnthropic}, true
	case Sonnet5:
		return modelInfo{benchmarkID: "claude-sonnet-5", provider: providerAnthropic}, true
	default:
		return modelInfo{}, false
	}
}

func providerForModel(model string) providerFamily {
	info, ok := modelInfoFor(model)
	if !ok {
		return providerUnknown
	}
	return info.provider
}
