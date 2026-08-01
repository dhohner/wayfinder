package recommender

type recommendationRule struct {
	matches   func(taskTraits) bool
	recommend func(Optimization, taskTraits) Recommendation
}

// defaultRules are evaluated in order; the first match wins. Each rule maps its
// task category to a candidate set and lets that set's bands choose the model
// and effort level for the requested optimization mode.
var defaultRules = []recommendationRule{
	{matches: func(traits taskTraits) bool { return traits.codeReview }, recommend: recommendCodeReview},
	{matches: isVisualDesignOnly, recommend: recommendUsing(&anthropicSet)},
	{matches: func(traits taskTraits) bool { return traits.coding }, recommend: recommendCoding},
	{matches: func(traits taskTraits) bool { return traits.anthropicFit }, recommend: recommendUsing(&anthropicSet)},
	{matches: func(traits taskTraits) bool { return traits.highRisk }, recommend: recommendUsing(&substantiveSet)},
	{matches: func(traits taskTraits) bool { return traits.deepReasoning }, recommend: recommendUsing(&substantiveSet)},
	{matches: func(traits taskTraits) bool { return traits.nuancedRoutine }, recommend: recommendUsing(&routineSet)},
	{matches: func(traits taskTraits) bool { return traits.simple && !traits.largeContext }, recommend: recommendUsing(&simpleSet)},
	{matches: func(traits taskTraits) bool { return traits.largeContext }, recommend: recommendUsing(&routineSet)},
}

func recommendUsing(set *candidateSet) func(Optimization, taskTraits) Recommendation {
	return func(optimization Optimization, _ taskTraits) Recommendation {
		return recommendFromSet(*set, optimization)
	}
}

func isVisualDesignOnly(traits taskTraits) bool {
	return traits.visualDesign && !traits.codingIntent && !traits.technicalDesign
}

func recommendCoding(optimization Optimization, traits taskTraits) Recommendation {
	if isSimpleCoding(traits) {
		return recommendFromSet(simpleSet, optimization)
	}
	if isHighReasoningCoding(traits) {
		return recommendFromSet(substantiveSet, optimization)
	}
	return recommendFromSet(routineSet, optimization)
}

func isHighReasoningCoding(traits taskTraits) bool {
	if traits.modelSelection {
		return false
	}
	return traits.highRisk || traits.deepReasoning || traits.correctnessHeavy || (traits.largeContext && !traits.routineCoding)
}

func recommendCodeReview(optimization Optimization, traits taskTraits) Recommendation {
	if traits.against == AgainstGPT {
		return recommendFromSet(anthropicSet, optimization)
	}
	return recommendFromSet(substantiveSet, optimization)
}

func isSimpleCoding(traits taskTraits) bool {
	return traits.simple && !traits.routineCoding && !traits.largeContext && !traits.deepReasoning && !traits.highRisk && !traits.nuancedRoutine && !traits.correctnessHeavy
}
