package recommender

type recommendationRule struct {
	matches   func(taskTraits) bool
	recommend func(Optimization, taskTraits) Recommendation
}

var defaultRules = []recommendationRule{
	{matches: func(traits taskTraits) bool { return traits.codeReview }, recommend: recommendCodeReview},
	{matches: isVisualDesignOnly, recommend: recommendVisualDesign},
	{matches: func(traits taskTraits) bool { return traits.coding }, recommend: recommendCoding},
	{matches: func(traits taskTraits) bool { return traits.anthropicFit }, recommend: recommendAnthropicFit},
	{matches: func(traits taskTraits) bool { return traits.highRisk }, recommend: recommendHighRisk},
	{matches: func(traits taskTraits) bool { return traits.deepReasoning }, recommend: recommendDeepReasoning},
	{matches: func(traits taskTraits) bool { return traits.nuancedRoutine }, recommend: recommendNuancedRoutine},
	{matches: func(traits taskTraits) bool { return traits.simple && !traits.largeContext }, recommend: recommendSimple},
	{matches: func(traits taskTraits) bool { return traits.largeContext }, recommend: recommendDevelopmentOrLargeContext},
}

func recommendHighRisk(optimization Optimization, _ taskTraits) Recommendation {
	if optimization == OptimizeQuality {
		return gptRecommendation(GPT56Sol, "xhigh", "Near-maximum reasoning quality for high-risk work at a more balanced cost.")
	}
	return gptRecommendation(GPT56Sol, "high", "Best value for high-risk work where stronger reasoning is worth the extra cost.")
}

func recommendDeepReasoning(optimization Optimization, _ taskTraits) Recommendation {
	if optimization == OptimizeQuality {
		return gptRecommendation(GPT56Sol, "xhigh", "Near-maximum reasoning quality for complex work at a more balanced cost.")
	}
	return gptRecommendation(GPT56Sol, "high", "Best value for complex work where stronger reasoning is worth the extra cost.")
}

func recommendAnthropicFit(optimization Optimization, _ taskTraits) Recommendation {
	switch optimization {
	case OptimizeQuality:
		return anthropicRecommendation(Opus48, "high", "Quality optimization selects the best Opus quality-cost balance for nuanced long-form or creative work.")
	case OptimizeCost, OptimizeSpeed:
		return gptRecommendation(GPT56Sol, "medium", "Cost or speed optimization favors the stronger default while keeping enough reasoning for nuanced work.")
	default:
		return anthropicRecommendation(Opus48, "low", "Good value for long-form or creative work where Claude is a better fit.")
	}
}

func isVisualDesignOnly(traits taskTraits) bool {
	return traits.visualDesign && !traits.codingIntent && !traits.technicalDesign
}

func recommendVisualDesign(optimization Optimization, _ taskTraits) Recommendation {
	if optimization == OptimizeQuality {
		return anthropicRecommendation(Opus48, "high", "Quality optimization raises effort for visual, UI, or UX design work where interface nuance matters.")
	}
	return anthropicRecommendation(Opus48, "low", "Good value for visual, UI, or UX design work while keeping starting effort low.")
}

func recommendNuancedRoutine(optimization Optimization, _ taskTraits) Recommendation {
	if optimization == OptimizeQuality {
		return gptRecommendation(GPT56Sol, "high", "Quality optimization adds reasoning depth for a nuanced routine task.")
	}
	return gptRecommendation(GPT56Sol, "medium", "Strong value for messy input, nuance, or multiple simple constraints.")
}

func recommendSimple(Optimization, taskTraits) Recommendation {
	return gptRecommendation(GPT56Sol, "low", "Low-reasoning GPT 5.6 Sol offers a strong low-cost result for a simple, low-risk task.")
}

func recommendCoding(optimization Optimization, traits taskTraits) Recommendation {
	if isSimpleCoding(traits) {
		if optimization == OptimizeQuality {
			return gptRecommendation(GPT56Sol, "medium", "Quality optimization adds moderate reasoning for a genuinely simple coding task.")
		}
		return recommendSimple(optimization, traits)
	}

	if isHighReasoningCoding(traits) {
		return recommendGPTReviewOrCoding(optimization, "coding")
	}

	return recommendRoutineCoding(optimization)
}

func recommendRoutineCoding(optimization Optimization) Recommendation {
	if optimization == OptimizeQuality {
		return gptRecommendation(GPT56Sol, "high", "Quality optimization adds stronger reasoning for routine coding work without using the maximum setting.")
	}
	return gptRecommendation(GPT56Sol, "medium", "Strong value for routine coding work without clear high-risk or deep-reasoning signals.")
}

func isHighReasoningCoding(traits taskTraits) bool {
	if traits.modelSelection {
		return false
	}
	return traits.highRisk || traits.deepReasoning || traits.correctnessHeavy || (traits.largeContext && !traits.routineCoding)
}

func recommendCodeReview(optimization Optimization, traits taskTraits) Recommendation {
	if traits.against == AgainstGPT {
		return recommendClaudeReview(optimization)
	}
	return recommendGPTReviewOrCoding(optimization, "review")
}

func recommendGPTReviewOrCoding(optimization Optimization, domain string) Recommendation {
	substantiveWork := "substantive coding work"
	if domain == "review" {
		substantiveWork = "adversarial code review"
	}

	switch optimization {
	case OptimizeCost:
		return gptRecommendation(GPT56Sol, "medium", "Cost optimization selects the lower-consumption GPT 5.6 Sol option for "+substantiveWork+".")
	case OptimizeSpeed:
		return gptRecommendation(GPT56Sol, "medium", "Speed optimization selects a lower-reasoning GPT 5.6 Sol option for "+substantiveWork+".")
	case OptimizeQuality:
		return gptRecommendation(GPT56Sol, "xhigh", "Quality optimization selects near-maximum GPT reasoning at a more balanced cost for "+substantiveWork+".")
	default:
		return gptRecommendation(GPT56Sol, "high", "Balanced value choice for "+substantiveWork+".")
	}
}

func recommendClaudeReview(optimization Optimization) Recommendation {
	switch optimization {
	case OptimizeCost, OptimizeSpeed:
		return anthropicRecommendation(Opus48, "low", "Lower-effort Opus provides the best review value against GPT-authored work.")
	case OptimizeQuality:
		return anthropicRecommendation(Opus48, "high", "Quality optimization selects Opus at its best quality-cost balance against GPT-authored work.")
	default:
		return anthropicRecommendation(Opus48, "high", "Cross-family review selects Opus to evaluate GPT-authored work.")
	}
}

func recommendDevelopmentOrLargeContext(optimization Optimization, _ taskTraits) Recommendation {
	if optimization == OptimizeQuality {
		return gptRecommendation(GPT56Sol, "high", "Quality optimization raises reasoning for larger-context work while avoiding the maximum-cost setting.")
	}
	return gptRecommendation(GPT56Sol, "medium", "Strong value for larger-context work.")
}

func isSimpleCoding(traits taskTraits) bool {
	return traits.simple && !traits.routineCoding && !traits.largeContext && !traits.deepReasoning && !traits.highRisk && !traits.nuancedRoutine && !traits.correctnessHeavy
}
