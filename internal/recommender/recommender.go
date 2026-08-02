package recommender

import "strings"

const (
	// Older models remain supported for compatibility, but bundled rules select
	// from the current benchmark data.
	GPT54      = "GPT 5.4"
	GPT55      = "GPT 5.5"
	GPT56Sol   = "GPT 5.6 Sol"
	GPT56Luna  = "GPT 5.6 Luna"
	GPT56Terra = "GPT 5.6 Terra"
	Opus48     = "Opus 4.8"
	Opus5      = "Claude Opus 5"
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
//
// Model and ReasoningSetting are identifiers and never localize: ReasoningSetting
// carries the bare provider effort level ("high"), and the provider terminology
// that labels it is added when the recommendation is formatted. Reason is
// human-readable text already written in Language, which is detected from the
// task text and defaults to English.
type Recommendation struct {
	Model            string
	ReasoningSetting string
	Reason           string
	Language         language
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
	rules []recommendationRule
}

// NewService returns a recommender using Wayfinder's bundled offline rules.
func NewService() Service {
	return Service{rules: append([]recommendationRule(nil), defaultRules...)}
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
//
// The task is answered in two steps that cannot see each other: the rules pick a
// model and a reason identifier from the task traits alone, and the detected
// language is applied to that selection afterwards. A language therefore cannot
// change which model or effort level a task is answered with, which is also why
// it is never part of taskTraits.
func (s Service) RecommendWithOptimizationAgainst(task string, optimization Optimization, against AgainstFamily) Recommendation {
	traits := classify(task)
	traits.against = against
	return s.selectionFor(traits, optimization).localize(detectLanguage(task))
}

func (s Service) selectionFor(traits taskTraits, optimization Optimization) selection {
	rules := s.rules
	if rules == nil {
		rules = defaultRules
	}

	for _, rule := range rules {
		if rule.matches(traits) {
			return rule.selects(traits).selectionFor(optimization)
		}
	}
	return ambiguousDefaultSelection()
}

// ambiguousDefaultSelection answers a task that matched no rule. Unlike the
// rules it ignores the optimization mode, staying deliberately conservative when
// the request gives too little signal to optimize for anything in particular.
// Its model and effort are the routine set's cost anchor.
func ambiguousDefaultSelection() selection {
	return selection{row: routineSet.selectFor(OptimizeCost), reason: reasonAmbiguousDefault}
}

type modelInfo struct {
	displayName string
	benchmarkID string
	provider    providerFamily
}

// bundledModels is the single registry mapping display names to the normalized
// IDs used in the bundled benchmark data and in JSON output.
var bundledModels = []modelInfo{
	{displayName: GPT54, benchmarkID: "gpt-5.4", provider: providerGPT},
	{displayName: GPT55, benchmarkID: "gpt-5.5", provider: providerGPT},
	{displayName: GPT56Sol, benchmarkID: "gpt-5.6-sol", provider: providerGPT},
	{displayName: GPT56Luna, benchmarkID: "gpt-5.6-luna", provider: providerGPT},
	{displayName: GPT56Terra, benchmarkID: "gpt-5.6-terra", provider: providerGPT},
	{displayName: Opus48, benchmarkID: "claude-opus-4.8", provider: providerAnthropic},
	{displayName: Opus5, benchmarkID: "claude-opus-5", provider: providerAnthropic},
	{displayName: Fable5, benchmarkID: "claude-fable-5", provider: providerAnthropic},
	{displayName: Sonnet46, benchmarkID: "claude-sonnet-4.6", provider: providerAnthropic},
	{displayName: Sonnet5, benchmarkID: "claude-sonnet-5", provider: providerAnthropic},
}

func modelInfoFor(model string) (modelInfo, bool) {
	for _, info := range bundledModels {
		if info.displayName == model {
			return info, true
		}
	}
	return modelInfo{}, false
}

func displayModelForBenchmarkID(benchmarkID string) (string, bool) {
	for _, info := range bundledModels {
		if info.benchmarkID == benchmarkID {
			return info.displayName, true
		}
	}
	return "", false
}

func providerForModel(model string) providerFamily {
	info, ok := modelInfoFor(model)
	if !ok {
		return providerUnknown
	}
	return info.provider
}
