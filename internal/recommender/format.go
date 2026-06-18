package recommender

import (
	"encoding/json"
	"fmt"
	"strconv"
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

// FormatJSON renders a stable machine-readable recommendation document.
// Benchmark fields are included only for exact bundled benchmark matches.
func FormatJSON(rec Recommendation, profile Optimization, explain bool) (string, error) {
	model, ok := benchmarkModelID(rec.Model)
	if !ok {
		return "", fmt.Errorf("cannot render JSON recommendation for unsupported model %q", rec.Model)
	}
	reasoning, ok := reasoningLevel(rec.ReasoningSetting)
	if !ok {
		return "", fmt.Errorf("cannot render JSON recommendation for unsupported reasoning setting %q", rec.ReasoningSetting)
	}
	profileValue, ok := normalizeProfile(profile)
	if !ok {
		return "", fmt.Errorf("cannot render JSON recommendation for unsupported profile %q", profile)
	}

	out := jsonRecommendation{
		Model:     model,
		Reasoning: reasoning,
		Profile:   profileValue,
		Reason:    rec.Reason,
	}
	if entry, ok := benchmarkForRecommendation(rec); ok {
		benchmark, err := entry.jsonBenchmark(explain)
		if err != nil {
			return "", err
		}
		out.Benchmark = benchmark
	}

	data, err := json.Marshal(out)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func normalizeProfile(profile Optimization) (string, bool) {
	switch profile {
	case OptimizeValue, OptimizeQuality, OptimizeCost, OptimizeSpeed:
		return string(profile), true
	default:
		return "", false
	}
}

type jsonRecommendation struct {
	Model     string         `json:"model"`
	Reasoning string         `json:"reasoning"`
	Profile   string         `json:"profile"`
	Reason    string         `json:"reason"`
	Benchmark *jsonBenchmark `json:"benchmark,omitempty"`
}

type jsonBenchmark struct {
	PassAt1   float64 `json:"pass_at_1"`
	AIC       float64 `json:"aic"`
	AICFactor float64 `json:"aic_factor"`
	Tradeoff  string  `json:"tradeoff,omitempty"`
}

func (entry benchmarkEntry) jsonBenchmark(explain bool) (*jsonBenchmark, error) {
	passAt1, err := parsePassAt1(entry.passAt1)
	if err != nil {
		return nil, err
	}
	aic, err := parseBenchmarkFloat("aic", entry.aic)
	if err != nil {
		return nil, err
	}
	aicFactor, err := parseBenchmarkFloat("aic_factor", entry.aicFactor)
	if err != nil {
		return nil, err
	}

	benchmark := &jsonBenchmark{PassAt1: passAt1, AIC: aic, AICFactor: aicFactor}
	if explain {
		benchmark.Tradeoff = entry.tradeoff
	}
	return benchmark, nil
}

func parsePassAt1(value string) (float64, error) {
	percent := strings.TrimSpace(value)
	if i := strings.Index(percent, "%"); i >= 0 {
		percent = percent[:i]
	}
	parsed, err := strconv.ParseFloat(strings.TrimSpace(percent), 64)
	if err != nil {
		return 0, fmt.Errorf("cannot parse benchmark pass_at_1 %q: %w", value, err)
	}
	return parsed / 100, nil
}

func parseBenchmarkFloat(name, value string) (float64, error) {
	parsed, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	if err != nil {
		return 0, fmt.Errorf("cannot parse benchmark %s %q: %w", name, value, err)
	}
	return parsed, nil
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
		tradeoff:  "Lower Pass@1 than GPT 5.5 high with higher estimated output-credit use.",
	},
	{model: "gpt-5.5", level: "medium"}: {
		passAt1:   "48%±3%",
		aic:       "57.0",
		aicFactor: "1.00",
		tradeoff:  "Lowest estimated output-credit use in the bundled benchmark, with lower Pass@1 than higher-reasoning GPT 5.5 options.",
	},
	{model: "gpt-5.5", level: "high"}: {
		passAt1:   "62%±4%",
		aic:       "93.0",
		aicFactor: "1.63",
		tradeoff:  "Stronger benchmark quality than medium at 1.63× the estimated output credits of gpt-5.5[medium].",
	},
	{model: "gpt-5.5", level: "xhigh"}: {
		passAt1:   "70%±3%",
		aic:       "141.0",
		aicFactor: "2.47",
		tradeoff:  "Highest Pass@1 in the bundled benchmark, with about 2.47× the estimated output credits of gpt-5.5[medium].",
	},
	{model: "claude-opus-4.8", level: "medium"}: {
		passAt1:   "47%±4%",
		aic:       "100.0",
		aicFactor: "1.75",
		tradeoff:  "Claude medium has near-GPT-5.5-medium Pass@1 with 1.75× the estimated output credits of gpt-5.5[medium].",
	},
	{model: "claude-opus-4.8", level: "high"}: {
		passAt1:   "51%±3%",
		aic:       "120.0",
		aicFactor: "2.11",
		tradeoff:  "Best bundled Claude quality/credit balance, with higher Pass@1 and output-credit use than Claude medium.",
	},
	{model: "claude-opus-4.8", level: "xhigh"}: {
		passAt1:   "58%±4%",
		aic:       "215.0",
		aicFactor: "3.77",
		tradeoff:  "Higher Claude Pass@1 than high, with substantially higher estimated output-credit use.",
	},
	{model: "claude-opus-4.8", level: "max"}: {
		passAt1:   "58%±2%",
		aic:       "340.0",
		aicFactor: "5.96",
		tradeoff:  "Matches Claude xhigh Pass@1 in the bundled benchmark but uses the most estimated output credits among Claude Opus 4.8 entries.",
	},
	{model: "claude-sonnet-4.6", level: "high"}: {
		passAt1:   "32%±2%",
		aic:       "114.0",
		aicFactor: "2.00",
		tradeoff:  "Lower Pass@1 than Opus 4.8 high with similar estimated output-credit use.",
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
