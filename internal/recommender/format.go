package recommender

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// Format renders the human-facing output contract: one model, one setting, one reason.
func Format(rec Recommendation) string {
	return fmt.Sprintf("Model: %s\nReasoning: %s\nReason: %s", rec.Model, rec.ReasoningSetting, rec.Reason)
}

// FormatWithExplanation renders the default output plus exact bundled benchmark evidence.
func FormatWithExplanation(rec Recommendation) string {
	out := Format(rec)
	entry, ok := benchmarkForRecommendation(rec)
	if !ok {
		return out
	}
	return fmt.Sprintf("%s\nBenchmark: Pass@1 %s; average cost %s.\nTradeoff: %s", out, entry.passAt1, entry.averageCost, entry.tradeoff)
}

// FormatJSON renders a stable machine-readable recommendation document.
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

	out := jsonRecommendation{Model: model, Reasoning: reasoning, Profile: profileValue, Reason: rec.Reason}
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
	PassAt1     float64 `json:"pass_at_1"`
	AverageCost float64 `json:"average_cost"`
	Tradeoff    string  `json:"tradeoff,omitempty"`
}

func (entry benchmarkEntry) jsonBenchmark(explain bool) (*jsonBenchmark, error) {
	passAt1, err := parsePassAt1(entry.passAt1)
	if err != nil {
		return nil, err
	}
	averageCost, err := parseBenchmarkFloat("average_cost", entry.averageCost)
	if err != nil {
		return nil, err
	}
	benchmark := &jsonBenchmark{PassAt1: passAt1, AverageCost: averageCost}
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
	return Recommendation{Model: model, ReasoningSetting: "GPT reasoning level: " + level, Reason: reason}
}

func anthropicRecommendation(model, level, reason string) Recommendation {
	return Recommendation{Model: model, ReasoningSetting: "Anthropic Effort Level: " + level, Reason: reason}
}

type benchmarkEntry struct {
	passAt1     string
	averageCost string
	tradeoff    string
}

type benchmarkKey struct {
	model string
	level string
}

var bundledBenchmarks = map[benchmarkKey]benchmarkEntry{
	{model: "gpt-5.6-sol", level: "low"}:        {passAt1: "45%±2%", averageCost: "1.07", tradeoff: "Lowest-cost GPT 5.6 Sol setting, with lower Pass@1 than medium or high."},
	{model: "gpt-5.6-sol", level: "medium"}:     {passAt1: "61%±2%", averageCost: "1.86", tradeoff: "Large quality gain over low for a modest cost increase."},
	{model: "gpt-5.6-sol", level: "high"}:       {passAt1: "69%±1%", averageCost: "3.47", tradeoff: "Strong quality-cost balance, close to the model's maximum Pass@1."},
	{model: "gpt-5.6-sol", level: "xhigh"}:      {passAt1: "71%±1%", averageCost: "4.70", tradeoff: "Near-maximum GPT quality at substantially less cost than max."},
	{model: "gpt-5.6-sol", level: "max"}:        {passAt1: "73%±3%", averageCost: "8.39", tradeoff: "Highest Pass@1 in the bundled benchmark, at the highest GPT 5.6 Sol cost."},
	{model: "claude-opus-4.8", level: "low"}:    {passAt1: "41%±1%", averageCost: "2.29", tradeoff: "Lowest-cost Opus setting, with lower Pass@1 than higher effort settings."},
	{model: "claude-opus-4.8", level: "medium"}: {passAt1: "49%±2%", averageCost: "3.44", tradeoff: "Moderate cost with a meaningful quality gain over low."},
	{model: "claude-opus-4.8", level: "high"}:   {passAt1: "52%±5%", averageCost: "4.28", tradeoff: "Best Opus quality-cost balance."},
	{model: "claude-opus-4.8", level: "xhigh"}:  {passAt1: "54%±4%", averageCost: "8.01", tradeoff: "Slight quality gain over high at substantially higher cost."},
	{model: "claude-opus-4.8", level: "max"}:    {passAt1: "59%±2%", averageCost: "13.22", tradeoff: "Highest Opus Pass@1 at the highest Opus cost."},
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
	case GPT56Sol:
		return "gpt-5.6-sol", true
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
