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

// benchmarkEntry is one published benchmark row.
//
// passAt1 keeps the published error margin, which is display-only; comparisons use
// the parsed point estimate. steps is the bundled proxy for wall-clock latency.
// credits is the estimated GitHub Copilot AI credit cost (1 credit = $0.01) of the
// run's output tokens only, priced at the GA short-context output rate; input,
// cached-input, and cache-write tokens are excluded, making it a lower bound that
// must never be presented as an exact price.
type benchmarkEntry struct {
	passAt1     string
	averageCost string
	steps       int
	credits     float64
	tradeoff    string
}

type benchmarkKey struct {
	model string
	level string
}

// bundledBenchmarks transcribes every row of the results table in bench_v2.md and is
// read-only after construction. bench.md is historical reference only and must not
// inform any value here.
var bundledBenchmarks = map[benchmarkKey]benchmarkEntry{
	{model: "claude-opus-5", level: "low"}:      {passAt1: "58%±2%", averageCost: "1.66", steps: 36, credits: 50.0, tradeoff: "Lowest-cost Opus 5 setting, well below its higher-effort Pass@1."},
	{model: "claude-opus-5", level: "medium"}:   {passAt1: "69%±1%", averageCost: "3.29", steps: 52, credits: 92.5, tradeoff: "Large quality gain over low at about half the cost of high."},
	{model: "claude-opus-5", level: "high"}:     {passAt1: "73%±2%", averageCost: "6.08", steps: 73, credits: 160.0, tradeoff: "Best Opus 5 quality-cost balance, within a point of its maximum Pass@1."},
	{model: "claude-opus-5", level: "xhigh"}:    {passAt1: "73%±3%", averageCost: "9.07", steps: 89, credits: 230.0, tradeoff: "Matches high on Pass@1 at appreciably higher cost and more steps."},
	{model: "claude-opus-5", level: "max"}:      {passAt1: "74%±4%", averageCost: "11.84", steps: 99, credits: 295.0, tradeoff: "Highest Pass@1 in the bundled benchmark, at the highest Opus 5 cost."},
	{model: "claude-opus-4.8", level: "low"}:    {passAt1: "41%±1%", averageCost: "2.29", steps: 54, credits: 72.5, tradeoff: "Lowest-cost Opus setting, with lower Pass@1 than higher effort settings."},
	{model: "claude-opus-4.8", level: "medium"}: {passAt1: "49%±2%", averageCost: "3.44", steps: 66, credits: 102.5, tradeoff: "Moderate cost with a meaningful quality gain over low."},
	{model: "claude-opus-4.8", level: "high"}:   {passAt1: "52%±5%", averageCost: "4.28", steps: 73, credits: 125.0, tradeoff: "Best Opus quality-cost balance."},
	{model: "claude-opus-4.8", level: "xhigh"}:  {passAt1: "54%±4%", averageCost: "8.01", steps: 95, credits: 215.0, tradeoff: "Slight quality gain over high at substantially higher cost."},
	{model: "claude-opus-4.8", level: "max"}:    {passAt1: "59%±2%", averageCost: "13.22", steps: 120, credits: 337.5, tradeoff: "Highest Opus 4.8 Pass@1 at the highest Opus 4.8 cost."},
	{model: "claude-fable-5", level: "low"}:     {passAt1: "60%±3%", averageCost: "3.76", steps: 38, credits: 125.0, tradeoff: "Lowest-cost Fable 5 setting, still well short of its higher-effort Pass@1."},
	{model: "claude-fable-5", level: "medium"}:  {passAt1: "65%±4%", averageCost: "6.09", steps: 48, credits: 200.0, tradeoff: "Moderate quality gain over low at a substantially higher cost."},
	{model: "claude-fable-5", level: "high"}:    {passAt1: "69%±1%", averageCost: "9.18", steps: 59, credits: 285.0, tradeoff: "Best Fable 5 quality-cost balance, close to its maximum Pass@1."},
	{model: "claude-fable-5", level: "xhigh"}:   {passAt1: "70%±3%", averageCost: "13.41", steps: 68, credits: 400.0, tradeoff: "Peak Fable 5 Pass@1, at a much higher cost than high."},
	{model: "claude-fable-5", level: "max"}:     {passAt1: "70%±4%", averageCost: "21.63", steps: 88, credits: 595.0, tradeoff: "Matches xhigh on Pass@1 at the highest cost in the bundled benchmark."},
	{model: "claude-sonnet-5", level: "low"}:    {passAt1: "31%±1%", averageCost: "2.19", steps: 77, credits: 36.0, tradeoff: "Lowest-cost Sonnet 5 setting, with the weakest Sonnet 5 Pass@1."},
	{model: "claude-sonnet-5", level: "medium"}: {passAt1: "40%±3%", averageCost: "4.08", steps: 108, credits: 57.0, tradeoff: "Clear quality gain over low at roughly double the cost."},
	{model: "claude-sonnet-5", level: "high"}:   {passAt1: "48%±5%", averageCost: "7.43", steps: 147, credits: 87.0, tradeoff: "Best Sonnet 5 quality-cost balance."},
	{model: "claude-sonnet-5", level: "xhigh"}:  {passAt1: "50%±3%", averageCost: "11.89", steps: 186, credits: 121.0, tradeoff: "Small quality gain over high at substantially higher cost and more steps."},
	{model: "claude-sonnet-5", level: "max"}:    {passAt1: "54%±4%", averageCost: "26.40", steps: 268, credits: 214.0, tradeoff: "Highest Sonnet 5 Pass@1, at more than double the cost of xhigh and the most benchmark steps of any bundled run."},
	{model: "claude-sonnet-4.6", level: "high"}: {passAt1: "30%±4%", averageCost: "5.52", steps: 134, credits: 114.0, tradeoff: "Only benchmarked Sonnet 4.6 setting, and behind Sonnet 5 on both quality and cost."},
	{model: "gpt-5.6-sol", level: "low"}:        {passAt1: "45%±2%", averageCost: "1.07", steps: 23, credits: 33.0, tradeoff: "Lowest-cost GPT 5.6 Sol setting, with lower Pass@1 than medium or high."},
	{model: "gpt-5.6-sol", level: "medium"}:     {passAt1: "61%±2%", averageCost: "1.86", steps: 31, credits: 54.0, tradeoff: "Large quality gain over low for a modest cost increase."},
	{model: "gpt-5.6-sol", level: "high"}:       {passAt1: "69%±1%", averageCost: "3.47", steps: 37, credits: 84.0, tradeoff: "Strong quality-cost balance, close to the model's maximum Pass@1."},
	{model: "gpt-5.6-sol", level: "xhigh"}:      {passAt1: "71%±1%", averageCost: "4.70", steps: 44, credits: 123.0, tradeoff: "Near-maximum GPT quality at substantially less cost than max."},
	{model: "gpt-5.6-sol", level: "max"}:        {passAt1: "73%±3%", averageCost: "8.39", steps: 61, credits: 180.0, tradeoff: "Highest GPT 5.6 Sol Pass@1, at the highest GPT 5.6 Sol cost."},
	{model: "gpt-5.6-luna", level: "low"}:       {passAt1: "2%±1%", averageCost: "0.07", steps: 12, credits: 1.9, tradeoff: "Cheapest and fastest bundled setting, but its Pass@1 is near zero."},
	{model: "gpt-5.6-luna", level: "medium"}:    {passAt1: "11%±1%", averageCost: "0.22", steps: 24, credits: 4.9, tradeoff: "Still far too low a Pass@1 to be a viable setting for real work."},
	{model: "gpt-5.6-luna", level: "high"}:      {passAt1: "44%±3%", averageCost: "0.78", steps: 49, credits: 15.6, tradeoff: "First Luna setting with usable Pass@1, at a very low cost."},
	{model: "gpt-5.6-luna", level: "xhigh"}:     {passAt1: "57%±2%", averageCost: "1.54", steps: 71, credits: 27.0, tradeoff: "Solid Pass@1 for the price, well above high."},
	{model: "gpt-5.6-luna", level: "max"}:       {passAt1: "67%±4%", averageCost: "3.03", steps: 102, credits: 43.8, tradeoff: "Highest Luna Pass@1 at a low cost, but its slowest setting by a wide margin."},
	{model: "gpt-5.6-terra", level: "low"}:      {passAt1: "24%±1%", averageCost: "0.43", steps: 21, credits: 12.9, tradeoff: "Lowest-cost Terra setting, with low Pass@1."},
	{model: "gpt-5.6-terra", level: "medium"}:   {passAt1: "35%±3%", averageCost: "0.58", steps: 25, credits: 18.0, tradeoff: "Modest quality gain over low for a small cost increase."},
	{model: "gpt-5.6-terra", level: "high"}:     {passAt1: "54%±4%", averageCost: "1.13", steps: 34, credits: 33.0, tradeoff: "Strong Pass@1 for a very low cost and few steps."},
	{model: "gpt-5.6-terra", level: "xhigh"}:    {passAt1: "60%±2%", averageCost: "2.13", steps: 43, credits: 60.0, tradeoff: "Better Pass@1 than high at roughly double the cost."},
	{model: "gpt-5.6-terra", level: "max"}:      {passAt1: "70%±3%", averageCost: "4.95", steps: 76, credits: 108.0, tradeoff: "Highest Terra Pass@1, competitive with far more expensive models."},
	{model: "gpt-5.5", level: "low"}:            {passAt1: "27%±2%", averageCost: "1.20", steps: 28, credits: 28.2, tradeoff: "Lowest-cost GPT 5.5 setting, with low Pass@1."},
	{model: "gpt-5.5", level: "medium"}:         {passAt1: "54%±3%", averageCost: "2.75", steps: 46, credits: 60.0, tradeoff: "Large quality gain over low for a modest cost increase."},
	{model: "gpt-5.5", level: "high"}:           {passAt1: "64%±3%", averageCost: "5.10", steps: 62, credits: 93.0, tradeoff: "Best GPT 5.5 quality-cost balance."},
	{model: "gpt-5.5", level: "xhigh"}:          {passAt1: "67%±6%", averageCost: "7.23", steps: 82, credits: 138.0, tradeoff: "Highest benchmarked GPT 5.5 Pass@1, at its highest cost."},
	{model: "gpt-5.4", level: "xhigh"}:          {passAt1: "52%±2%", averageCost: "5.65", steps: 70, credits: 106.5, tradeoff: "Only benchmarked GPT 5.4 setting, and behind GPT 5.6 on both quality and cost."},
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
	info, ok := modelInfoFor(model)
	return info.benchmarkID, ok
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
