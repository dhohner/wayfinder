package recommender

type recommendationRule struct {
	matches   func(taskTraits) bool
	recommend func(Optimization, taskTraits) Recommendation
}

var defaultRules = []recommendationRule{
	{matches: func(traits taskTraits) bool { return traits.anthropicFit || traits.visualDesign }, recommend: recommendAnthropicFit},
	{matches: func(traits taskTraits) bool { return traits.coding }, recommend: recommendCoding},
	{matches: func(traits taskTraits) bool { return traits.highRisk }, recommend: recommendHighRisk},
	{matches: func(traits taskTraits) bool { return traits.deepReasoning }, recommend: recommendDeepReasoning},
	{matches: func(traits taskTraits) bool { return traits.nuancedRoutine }, recommend: recommendNuancedRoutine},
	{matches: func(traits taskTraits) bool { return traits.simple && !traits.largeContext }, recommend: recommendSimple},
	{matches: func(traits taskTraits) bool { return traits.largeContext }, recommend: recommendDevelopmentOrLargeContext},
}

func recommendHighRisk(optimization Optimization, _ taskTraits) Recommendation {
	if optimization == OptimizeQuality {
		return gptRecommendation(GPT55, "xhigh", "Best fit for high-risk work where maximum reasoning quality is worth the extra cost.")
	}
	return gptRecommendation(GPT55, "high", "Best fit for high-risk work where stronger reasoning is worth the extra cost.")
}

func recommendDeepReasoning(optimization Optimization, _ taskTraits) Recommendation {
	if optimization == OptimizeQuality {
		return gptRecommendation(GPT55, "xhigh", "Best fit for complex work where maximum reasoning quality is worth the extra cost.")
	}
	return gptRecommendation(GPT55, "high", "Best fit for complex work where stronger reasoning is worth the extra cost.")
}

func recommendAnthropicFit(optimization Optimization, _ taskTraits) Recommendation {
	switch optimization {
	case OptimizeQuality:
		return anthropicRecommendation(Opus48, "high", "Quality optimization chooses Opus for long-form, creative, or visual design work where nuance matters.")
	case OptimizeCost, OptimizeSpeed:
		return gptRecommendation(GPT55, "medium", "Cost or speed optimization favors the stronger default while keeping enough reasoning for nuanced work.")
	default:
		return anthropicRecommendation(Opus48, "medium", "Good fit for long-form, creative, or visual design work where Opus capability is useful.")
	}
}

func recommendNuancedRoutine(optimization Optimization, _ taskTraits) Recommendation {
	if optimization == OptimizeQuality {
		return gptRecommendation(GPT55, "high", "Quality optimization adds reasoning depth for a nuanced routine task.")
	}
	return gptRecommendation(GPT55, "medium", "Good default for messy input, nuance, or multiple simple constraints.")
}

func recommendSimple(Optimization, taskTraits) Recommendation {
	return gptRecommendation(GPT55, "low", "Lower-reasoning GPT 5.5 is adequate for a simple, low-risk task.")
}

func recommendCoding(optimization Optimization, traits taskTraits) Recommendation {
	if isSimpleCoding(traits) {
		if optimization == OptimizeQuality {
			return gptRecommendation(GPT55, "medium", "Quality optimization adds moderate reasoning for a genuinely simple coding task.")
		}
		return recommendSimple(optimization, traits)
	}

	switch optimization {
	case OptimizeCost:
		return gptRecommendation(GPT55, "medium", "Cost optimization selects the lower-consumption coding frontier option while preserving GPT 5.5 capability.")
	case OptimizeSpeed:
		return gptRecommendation(GPT55, "medium", "Speed optimization selects the lower-consumption coding frontier option without relying on latency measurements.")
	case OptimizeQuality:
		return gptRecommendation(GPT55, "xhigh", "Quality optimization selects the strongest GPT 5.5 reasoning for substantive coding work.")
	default:
		return gptRecommendation(GPT55, "high", "Balanced value choice for substantive coding work.")
	}
}

func recommendDevelopmentOrLargeContext(optimization Optimization, _ taskTraits) Recommendation {
	if optimization == OptimizeQuality {
		return gptRecommendation(GPT55, "high", "Quality optimization raises reasoning for larger-context work while avoiding the maximum-cost setting.")
	}
	return gptRecommendation(GPT55, "medium", "Good default for larger-context work.")
}

func isSimpleCoding(traits taskTraits) bool {
	return traits.simple && !traits.largeContext && !traits.deepReasoning && !traits.highRisk && !traits.nuancedRoutine && !traits.correctnessHeavy
}
