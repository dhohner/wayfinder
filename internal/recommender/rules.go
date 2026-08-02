package recommender

// recommendationRule matches a task category and names the candidate set that
// answers it. Both functions see only the task traits: a rule cannot observe the
// output language, so a German prompt and its English equivalent reach the same
// candidate set and therefore the same model and effort level. Language enters
// once, where the resulting selection is localized.
type recommendationRule struct {
	matches func(taskTraits) bool
	selects func(taskTraits) *candidateSet
}

// defaultRules are evaluated in order; the first match wins. Each rule maps its
// task category to a candidate set and lets that set's bands choose the model
// and effort level for the requested optimization mode.
var defaultRules = []recommendationRule{
	{matches: func(traits taskTraits) bool { return traits.codeReview }, selects: selectCodeReviewSet},
	{matches: isVisualDesignOnly, selects: alwaysSelect(&anthropicSet)},
	{matches: func(traits taskTraits) bool { return traits.coding }, selects: selectCodingSet},
	{matches: func(traits taskTraits) bool { return traits.anthropicFit }, selects: alwaysSelect(&anthropicSet)},
	{matches: func(traits taskTraits) bool { return traits.highRisk || traits.deepReasoning }, selects: alwaysSelect(&substantiveSet)},
	{matches: func(traits taskTraits) bool { return traits.nuancedRoutine }, selects: alwaysSelect(&routineSet)},
	{matches: func(traits taskTraits) bool { return traits.simple && !traits.largeContext }, selects: alwaysSelect(&simpleSet)},
	{matches: func(traits taskTraits) bool { return traits.largeContext }, selects: alwaysSelect(&routineSet)},
}

func alwaysSelect(set *candidateSet) func(taskTraits) *candidateSet {
	return func(taskTraits) *candidateSet { return set }
}

func isVisualDesignOnly(traits taskTraits) bool {
	return traits.visualDesign && !traits.codingIntent && !traits.technicalDesign
}

func selectCodingSet(traits taskTraits) *candidateSet {
	if isSimpleCoding(traits) {
		return &simpleSet
	}
	if isHighReasoningCoding(traits) {
		return &substantiveSet
	}
	return &routineSet
}

func isHighReasoningCoding(traits taskTraits) bool {
	if traits.modelSelection {
		return false
	}
	return traits.highRisk || traits.deepReasoning || traits.correctnessHeavy || (traits.largeContext && !traits.routineCoding)
}

func selectCodeReviewSet(traits taskTraits) *candidateSet {
	if traits.against == AgainstGPT {
		return &anthropicSet
	}
	return &substantiveSet
}

func isSimpleCoding(traits taskTraits) bool {
	return traits.simple && !traits.routineCoding && !traits.largeContext && !traits.deepReasoning && !traits.highRisk && !traits.nuancedRoutine && !traits.correctnessHeavy
}
