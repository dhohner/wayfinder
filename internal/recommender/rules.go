package recommender

type recommendationRule struct {
	matches   func(taskTraits) bool
	recommend func(Preference) Recommendation
}

var defaultRules = []recommendationRule{
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
