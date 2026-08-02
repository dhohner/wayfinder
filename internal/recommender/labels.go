package recommender

import "fmt"

// outputLabels holds every fixed label of the human-readable output for one
// language. The labels are part of the user-visible contract, so they are
// written in one table per language rather than inline at their format string.
//
// Three values stay untranslated in both languages because a user types them
// into a provider UI rather than reading them as prose: the effort levels
// (low, medium, high, xhigh, max), the "Pass@1" metric name, and the model
// display names. The German "Reasoning:" label is likewise unchanged, because
// the provider terminology it introduces is itself English.
type outputLabels struct {
	model                string
	reasoning            string
	reason               string
	benchmark            string
	averageCost          string
	credits              string
	creditsQualification string
	tradeoff             string
	gptReasoning         string
	anthropicEffort      string
}

var labelsByLanguage = map[language]outputLabels{
	languageEnglish: {
		model:                "Model:",
		reasoning:            "Reasoning:",
		reason:               "Reason:",
		benchmark:            "Benchmark:",
		averageCost:          "average cost",
		credits:              "Estimated Copilot AI credits:",
		creditsQualification: "input and output tokens, estimate",
		tradeoff:             "Tradeoff:",
		gptReasoning:         "GPT reasoning level:",
		anthropicEffort:      "Anthropic Effort Level:",
	},
	languageGerman: {
		model:                "Modell:",
		reasoning:            "Reasoning:",
		reason:               "Begründung:",
		benchmark:            "Benchmark:",
		averageCost:          "durchschnittliche Kosten",
		credits:              "Geschätzte Copilot-AI-Credits:",
		creditsQualification: "Eingabe- und Ausgabe-Tokens, Schätzwert",
		tradeoff:             "Abwägung:",
		gptReasoning:         "GPT-Reasoning-Stufe:",
		anthropicEffort:      "Anthropic-Effort-Stufe:",
	},
}

func labelsFor(lang language) outputLabels {
	labels, ok := labelsByLanguage[lang]
	if !ok {
		panic(fmt.Sprintf("recommender: no output labels registered for %s", lang))
	}
	return labels
}
