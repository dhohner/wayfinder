package recommender

import (
	"fmt"
	"strings"
)

// Format renders the v1 human-facing output contract: one model, one setting, one reason.
func Format(rec Recommendation) string {
	return fmt.Sprintf("Model: %s\nReasoning: %s\nReason: %s", rec.Model, rec.ReasoningSetting, rec.Reason)
}

// FormatWithExplanation renders the default human output plus exact bundled benchmark evidence when available.
// Recommendations without an exact benchmark match keep the compact output and do not show benchmark details.
func FormatWithExplanation(rec Recommendation) string {
	out := Format(rec)
	entry, ok := benchmarkForRecommendation(rec)
	if !ok {
		return out
	}

	return fmt.Sprintf(
		"%s\nBenchmark: Pass@1 %s; AIC %s; AIC factor %s.\nTradeoff: %s",
		out,
		entry.passAt1,
		entry.aic,
		entry.aicFactor,
		entry.tradeoff,
	)
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

type benchmarkEntry struct {
	passAt1   string
	aic       string
	aicFactor string
	tradeoff  string
}

type benchmarkKey struct {
	model string
	level string
}

var bundledBenchmarks = map[benchmarkKey]benchmarkEntry{
	{model: "gpt-5.4", level: "xhigh"}: {
		passAt1:   "56%±2%",
		aic:       "106.5",
		aicFactor: "1.87",
		tradeoff:  "Lower Pass@1 than GPT 5.5 high with higher estimated output-credit use; the data contains no latency measurements.",
	},
	{model: "gpt-5.5", level: "medium"}: {
		passAt1:   "48%±3%",
		aic:       "57.0",
		aicFactor: "1.00",
		tradeoff:  "Lowest estimated output-credit use in the bundled benchmark, with lower Pass@1 than higher-reasoning GPT 5.5 options; the data contains no latency measurements.",
	},
	{model: "gpt-5.5", level: "high"}: {
		passAt1:   "62%±4%",
		aic:       "93.0",
		aicFactor: "1.63",
		tradeoff:  "Stronger benchmark quality than medium at 1.63× the estimated output credits of gpt-5.5[medium]; the data contains no latency measurements.",
	},
	{model: "gpt-5.5", level: "xhigh"}: {
		passAt1:   "70%±3%",
		aic:       "141.0",
		aicFactor: "2.47",
		tradeoff:  "Highest Pass@1 in the bundled benchmark, with about 2.47× the estimated output credits of gpt-5.5[medium]; the data contains no latency measurements.",
	},
	{model: "claude-opus-4.8", level: "medium"}: {
		passAt1:   "47%±4%",
		aic:       "100.0",
		aicFactor: "1.75",
		tradeoff:  "Claude medium has near-GPT-5.5-medium Pass@1 with 1.75× the estimated output credits of gpt-5.5[medium]; the data contains no latency measurements.",
	},
	{model: "claude-opus-4.8", level: "high"}: {
		passAt1:   "51%±3%",
		aic:       "120.0",
		aicFactor: "2.11",
		tradeoff:  "Best bundled Claude quality/credit balance, with higher Pass@1 and output-credit use than Claude medium; the data contains no latency measurements.",
	},
	{model: "claude-opus-4.8", level: "xhigh"}: {
		passAt1:   "58%±4%",
		aic:       "215.0",
		aicFactor: "3.77",
		tradeoff:  "Higher Claude Pass@1 than high, with substantially higher estimated output-credit use; the data contains no latency measurements.",
	},
	{model: "claude-opus-4.8", level: "max"}: {
		passAt1:   "58%±2%",
		aic:       "340.0",
		aicFactor: "5.96",
		tradeoff:  "Matches Claude xhigh Pass@1 in the bundled benchmark but uses the most estimated output credits among Claude Opus 4.8 entries; the data contains no latency measurements.",
	},
	{model: "claude-sonnet-4.6", level: "high"}: {
		passAt1:   "32%±2%",
		aic:       "114.0",
		aicFactor: "2.00",
		tradeoff:  "Lower Pass@1 than Opus 4.8 high with similar estimated output-credit use; the data contains no latency measurements.",
	},
}

func benchmarkForRecommendation(rec Recommendation) (benchmarkEntry, bool) {
	model, ok := benchmarkModelID(rec.Model)
	if !ok {
		return benchmarkEntry{}, false
	}
	level, ok := reasoningLevel(rec.ReasoningSetting)
	if !ok {
		return benchmarkEntry{}, false
	}
	entry, ok := bundledBenchmarks[benchmarkKey{model: model, level: level}]
	return entry, ok
}

func benchmarkModelID(model string) (string, bool) {
	switch model {
	case GPT54:
		return "gpt-5.4", true
	case GPT55:
		return "gpt-5.5", true
	case Opus48:
		return "claude-opus-4.8", true
	case Sonnet46:
		return "claude-sonnet-4.6", true
	default:
		return "", false
	}
}

func reasoningLevel(setting string) (string, bool) {
	for _, prefix := range []string{"GPT reasoning level:", "Anthropic Effort Level:"} {
		if strings.HasPrefix(setting, prefix) {
			level := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(setting, prefix)))
			return level, level != ""
		}
	}
	return "", false
}
