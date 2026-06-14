package recommender

import "fmt"

// Format renders the v1 human-facing output contract: one model, one setting, one reason.
func Format(rec Recommendation) string {
	return fmt.Sprintf("Model: %s\nReasoning: %s\nReason: %s", rec.Model, rec.ReasoningSetting, rec.Reason)
}

func gptRecommendation(model, level, reason string) Recommendation {
	return Recommendation{
		Model:            model,
		ReasoningSetting: "GPT reasoning level: " + level,
		Reason:           reason,
	}
}

func anthropicRecommendation(model, level, reason string) Recommendation {
	return Recommendation{
		Model:            model,
		ReasoningSetting: "Anthropic Effort Level: " + level,
		Reason:           reason,
	}
}
