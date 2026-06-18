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
		return anthropicRecommendation(Opus48, "high", "Quality optimization chooses Opus for long-form or creative work where nuance matters.")
	case OptimizeCost, OptimizeSpeed:
		return gptRecommendation(GPT55, "medium", "Cost or speed optimization favors the stronger default while keeping enough reasoning for nuanced work.")
	default:
		return anthropicRecommendation(Opus48, "medium", "Good fit for long-form or creative work where Opus capability is useful.")
	}
}

func isVisualDesignOnly(traits taskTraits) bool {
	return traits.visualDesign && !traits.codingIntent && !traits.technicalDesign
}

func recommendVisualDesign(optimization Optimization, _ taskTraits) Recommendation {
	if optimization == OptimizeQuality {
		return anthropicRecommendation(Opus48, "medium", "Quality optimization raises effort for visual, UI, or UX design work where interface nuance matters.")
	}
	return anthropicRecommendation(Opus48, "low", "Good fit for visual, UI, or UX design work while keeping starting effort low.")
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

	return recommendGPTReviewOrCoding(optimization, "coding")
}

func recommendCodeReview(optimization Optimization, traits taskTraits) Recommendation {
	switch traits.against {
	case AgainstGPT:
		return recommendClaudeReview(optimization)
	case AgainstClaude, AgainstUnspecified:
		return recommendGPTReviewOrCoding(optimization, "review")
	default:
		return recommendGPTReviewOrCoding(optimization, "review")
	}
}

func recommendGPTReviewOrCoding(optimization Optimization, domain string) Recommendation {
	substantiveWork := "substantive coding work"
	costReason := "Cost optimization selects the lower-consumption coding frontier option while preserving GPT 5.5 capability."
	speedReason := "Speed optimization selects the lower-consumption coding frontier option without relying on latency measurements."
	if domain == "review" {
		substantiveWork = "adversarial code review"
		costReason = "Cost optimization selects the lower-consumption GPT review frontier option while preserving GPT 5.5 capability."
		speedReason = "Speed optimization selects the lower-consumption GPT review frontier option without relying on latency measurements."
	}

	switch optimization {
	case OptimizeCost:
		return gptRecommendation(GPT55, "medium", costReason)
	case OptimizeSpeed:
		return gptRecommendation(GPT55, "medium", speedReason)
	case OptimizeQuality:
		return gptRecommendation(GPT55, "xhigh", "Quality optimization selects the strongest GPT 5.5 reasoning for "+substantiveWork+".")
	default:
		return gptRecommendation(GPT55, "high", "Balanced value choice for "+substantiveWork+".")
	}
}

func recommendClaudeReview(optimization Optimization) Recommendation {
	switch optimization {
	case OptimizeCost:
		return anthropicRecommendation(Opus48, "medium", "Cost optimization selects Claude Opus for adversarial review while keeping effort moderate.")
	case OptimizeSpeed:
		return anthropicRecommendation(Opus48, "medium", "Speed optimization selects Claude Opus for adversarial review without relying on latency measurements.")
	case OptimizeQuality:
		return anthropicRecommendation(Opus48, "xhigh", "Quality optimization selects Claude Opus with maximum review effort against GPT-authored work.")
	default:
		return anthropicRecommendation(Opus48, "high", "Cross-family review selects Claude Opus to evaluate GPT-authored work.")
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
