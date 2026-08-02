package recommender

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// Format renders the human-facing output contract: one model, one setting, one
// reason, in the recommendation's language.
func Format(rec Recommendation) string {
	labels := labelsFor(rec.Language)
	return fmt.Sprintf("%s %s\n%s %s\n%s %s",
		labels.model, rec.Model,
		labels.reasoning, reasoningSettingText(rec),
		labels.reason, rec.Reason,
	)
}

// reasoningSettingText labels the effort level with the terminology of the
// selected model's provider. The level itself is a provider-defined identifier
// and stays untranslated; only the label around it follows the language. A
// model outside the bundled registry has no provider terminology to apply, so
// its setting is reported bare.
func reasoningSettingText(rec Recommendation) string {
	labels := labelsFor(rec.Language)
	switch providerForModel(rec.Model) {
	case providerGPT:
		return labels.gptReasoning + " " + rec.ReasoningSetting
	case providerAnthropic:
		return labels.anthropicEffort + " " + rec.ReasoningSetting
	default:
		return rec.ReasoningSetting
	}
}

// FormatWithExplanation renders the default output plus exact bundled benchmark
// evidence. The credit figure keeps its qualification in every language,
// because the figure models a cache-hit rate rather than measuring one.
func FormatWithExplanation(rec Recommendation) string {
	out := Format(rec)
	entry, ok := benchmarkForRecommendation(rec)
	if !ok {
		return out
	}
	labels := labelsFor(rec.Language)
	return fmt.Sprintf(
		"%s\n%s Pass@1 %s; %s %s.\n%s %.1f (%s).\n%s %s",
		out,
		labels.benchmark, entry.passAt1, labels.averageCost, entry.averageCost,
		labels.credits, entry.credits, labels.creditsQualification,
		labels.tradeoff, entry.tradeoff.in(rec.Language),
	)
}

// FormatJSON renders a stable machine-readable recommendation document.
//
// Field names, the normalized model ID, the normalized reasoning ID, the
// profile, and the benchmark keys are identifiers and stay English whatever the
// prompt language. Only `reason` and `tradeoff` carry human-readable text and
// therefore follow the recommendation's language.
func FormatJSON(rec Recommendation, profile Optimization, explain bool) (string, error) {
	model, ok := benchmarkModelID(rec.Model)
	if !ok {
		return "", fmt.Errorf("cannot render JSON recommendation for unsupported model %q", rec.Model)
	}
	reasoning, ok := normalizeEffortLevel(rec.ReasoningSetting)
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
		// The tradeoff sentence is the block's only human-readable field, so it
		// is the only one re-rendered in the recommendation's language when the
		// document carries it.
		if benchmark.Tradeoff != "" {
			benchmark.Tradeoff = entry.tradeoff.in(rec.Language)
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
	PassAt1         float64 `json:"pass_at_1"`
	AverageCost     float64 `json:"average_cost"`
	CreditsEstimate float64 `json:"credits_estimate"`
	Tradeoff        string  `json:"tradeoff,omitempty"`
}

// jsonBenchmark renders the machine-readable evidence block. Every field but
// the tradeoff sentence is numeric and language-independent; the sentence is
// rendered in the default language and localized by FormatJSON.
func (entry benchmarkEntry) jsonBenchmark(explain bool) (*jsonBenchmark, error) {
	passAt1, err := parsePassAt1(entry.passAt1)
	if err != nil {
		return nil, err
	}
	averageCost, err := parseBenchmarkFloat("average_cost", entry.averageCost)
	if err != nil {
		return nil, err
	}
	benchmark := &jsonBenchmark{PassAt1: passAt1, AverageCost: averageCost, CreditsEstimate: entry.credits}
	if explain {
		benchmark.Tradeoff = entry.tradeoff.in(languageEnglish)
	}
	return benchmark, nil
}

func passAt1NumericText(value string) string {
	percent := strings.TrimSpace(value)
	if i := strings.Index(percent, "%"); i >= 0 {
		percent = percent[:i]
	}
	return strings.TrimSpace(percent)
}

func parsePassAt1(value string) (float64, error) {
	parsed, err := strconv.ParseFloat(passAt1NumericText(value), 64)
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

// effortLevels are the reasoning settings a developer types into a provider UI.
// They are provider-defined identifiers: they are never translated, and they
// are carried on the recommendation rather than recovered from a displayed
// label, so that localizing a label cannot break the JSON document or the
// benchmark lookup.
var effortLevels = []string{"low", "medium", "high", "xhigh", "max"}

func normalizeEffortLevel(setting string) (string, bool) {
	level := strings.ToLower(strings.TrimSpace(setting))
	for _, known := range effortLevels {
		if level == known {
			return level, true
		}
	}
	return "", false
}

// benchmarkEntry is one published benchmark row.
//
// passAt1 keeps the published error margin, which is display-only; comparisons use
// the parsed point estimate. steps is the bundled proxy for wall-clock latency.
// credits is the estimated GitHub Copilot AI credit cost (1 credit = $0.01) of the
// whole run, counting output tokens at the GA short-context output rate and input
// tokens at a blended rate assuming a 95% cache-hit ratio. That ratio is fitted
// from the benchmark's own measured costs, not measured per run, so the figure is
// a two-sided estimate carrying roughly a quarter error bar and must never be
// presented as an exact price. publishedRow is the row's 1-based position in the
// bench_v2.md results table and exists only so selection has a deterministic final
// tie-break.
type benchmarkEntry struct {
	publishedRow int
	passAt1      string
	averageCost  string
	steps        int
	credits      float64
	tradeoff     localizedText
}

type benchmarkKey struct {
	model string
	level string
}

// bundledBenchmarks transcribes every row of the results table in bench_v2.md and is
// read-only after construction. bench.md is historical reference only and must not
// inform any value here.
//
// The credits values derive from Copilot list prices verified on 2026-08-02 applied to
// the DeepSWE v1.1 token counts; a provider price change invalidates them even when the
// benchmark itself is unchanged, so refresh them from bench_v2.md rather than recomputing
// from an older price table. Because they now include input tokens, they are roughly
// three times the output-only figures published before 2026-08-02 and are not comparable
// to them; the credit ceilings in selection.go are denominated in these units.
//
// Every tradeoff sentence is written in both output languages. Effort level names
// and Pass@1 stay untranslated inside the German copy, because they are the
// identifiers a developer types into a provider UI.
var bundledBenchmarks = map[benchmarkKey]benchmarkEntry{
	{model: "claude-opus-5", level: "low"}:      {publishedRow: 20, passAt1: "58%±2%", averageCost: "1.66", steps: 36, credits: 163.1, tradeoff: localizedText{english: "Lowest-cost Opus 5 setting, well below its higher-effort Pass@1.", german: "Günstigste Opus-5-Einstellung, deutlich unter dem Pass@1 der höheren Stufen."}},
	{model: "claude-opus-5", level: "medium"}:   {publishedRow: 10, passAt1: "69%±1%", averageCost: "3.29", steps: 52, credits: 352.0, tradeoff: localizedText{english: "Large quality gain over low at about half the cost of high.", german: "Großer Qualitätsgewinn gegenüber low bei etwa der Hälfte der Kosten von high."}},
	{model: "claude-opus-5", level: "high"}:     {publishedRow: 3, passAt1: "73%±2%", averageCost: "6.08", steps: 73, credits: 685.0, tradeoff: localizedText{english: "Best Opus 5 quality-cost balance, within a point of its maximum Pass@1.", german: "Bestes Verhältnis von Qualität und Kosten bei Opus 5, einen Punkt unter seinem maximalen Pass@1."}},
	{model: "claude-opus-5", level: "xhigh"}:    {publishedRow: 2, passAt1: "73%±3%", averageCost: "9.07", steps: 89, credits: 1047.9, tradeoff: localizedText{english: "Matches high on Pass@1 at appreciably higher cost and more steps.", german: "Erreicht das Pass@1 von high bei spürbar höheren Kosten und mehr Schritten."}},
	{model: "claude-opus-5", level: "max"}:      {publishedRow: 1, passAt1: "74%±4%", averageCost: "11.84", steps: 99, credits: 1383.3, tradeoff: localizedText{english: "Highest Pass@1 in the bundled benchmark, at the highest Opus 5 cost.", german: "Höchstes Pass@1 im mitgelieferten Benchmark, zu den höchsten Opus-5-Kosten."}},
	{model: "claude-opus-4.8", level: "low"}:    {publishedRow: 33, passAt1: "41%±1%", averageCost: "2.29", steps: 54, credits: 247.6, tradeoff: localizedText{english: "Lowest-cost Opus setting, with lower Pass@1 than higher effort settings.", german: "Günstigste Opus-Einstellung, mit geringerem Pass@1 als die höheren Stufen."}},
	{model: "claude-opus-4.8", level: "medium"}: {publishedRow: 29, passAt1: "49%±2%", averageCost: "3.44", steps: 66, credits: 381.9, tradeoff: localizedText{english: "Moderate cost with a meaningful quality gain over low.", german: "Moderate Kosten mit einem spürbaren Qualitätsgewinn gegenüber low."}},
	{model: "claude-opus-4.8", level: "high"}:   {publishedRow: 27, passAt1: "52%±5%", averageCost: "4.28", steps: 73, credits: 481.0, tradeoff: localizedText{english: "Best Opus quality-cost balance.", german: "Bestes Verhältnis von Qualität und Kosten bei Opus."}},
	{model: "claude-opus-4.8", level: "xhigh"}:  {publishedRow: 22, passAt1: "54%±4%", averageCost: "8.01", steps: 95, credits: 929.7, tradeoff: localizedText{english: "Slight quality gain over high at substantially higher cost.", german: "Geringer Qualitätsgewinn gegenüber high bei deutlich höheren Kosten."}},
	{model: "claude-opus-4.8", level: "max"}:    {publishedRow: 19, passAt1: "59%±2%", averageCost: "13.22", steps: 120, credits: 1573.6, tradeoff: localizedText{english: "Highest Opus 4.8 Pass@1 at the highest Opus 4.8 cost.", german: "Höchstes Opus-4.8-Pass@1 zu den höchsten Opus-4.8-Kosten."}},
	{model: "claude-fable-5", level: "low"}:     {publishedRow: 18, passAt1: "60%±3%", averageCost: "3.76", steps: 38, credits: 371.4, tradeoff: localizedText{english: "Lowest-cost Fable 5 setting, still well short of its higher-effort Pass@1.", german: "Günstigste Fable-5-Einstellung, weiterhin deutlich unter dem Pass@1 der höheren Stufen."}},
	{model: "claude-fable-5", level: "medium"}:  {publishedRow: 14, passAt1: "65%±4%", averageCost: "6.09", steps: 48, credits: 627.6, tradeoff: localizedText{english: "Moderate quality gain over low at a substantially higher cost.", german: "Moderater Qualitätsgewinn gegenüber low bei deutlich höheren Kosten."}},
	{model: "claude-fable-5", level: "high"}:    {publishedRow: 11, passAt1: "69%±1%", averageCost: "9.18", steps: 59, credits: 987.2, tradeoff: localizedText{english: "Best Fable 5 quality-cost balance, close to its maximum Pass@1.", german: "Bestes Verhältnis von Qualität und Kosten bei Fable 5, nahe an seinem maximalen Pass@1."}},
	{model: "claude-fable-5", level: "xhigh"}:   {publishedRow: 6, passAt1: "70%±3%", averageCost: "13.41", steps: 68, credits: 1464.5, tradeoff: localizedText{english: "Peak Fable 5 Pass@1, at a much higher cost than high.", german: "Höchstes Fable-5-Pass@1, zu deutlich höheren Kosten als high."}},
	{model: "claude-fable-5", level: "max"}:     {publishedRow: 7, passAt1: "70%±4%", averageCost: "21.63", steps: 88, credits: 2425.6, tradeoff: localizedText{english: "Matches xhigh on Pass@1 at the highest cost in the bundled benchmark.", german: "Erreicht das Pass@1 von xhigh zu den höchsten Kosten im mitgelieferten Benchmark."}},
	{model: "claude-sonnet-5", level: "low"}:    {publishedRow: 36, passAt1: "31%±1%", averageCost: "2.19", steps: 77, credits: 167.1, tradeoff: localizedText{english: "Lowest-cost Sonnet 5 setting, with the weakest Sonnet 5 Pass@1.", german: "Günstigste Sonnet-5-Einstellung, mit dem schwächsten Sonnet-5-Pass@1."}},
	{model: "claude-sonnet-5", level: "medium"}: {publishedRow: 34, passAt1: "40%±3%", averageCost: "4.08", steps: 108, credits: 326.1, tradeoff: localizedText{english: "Clear quality gain over low at roughly double the cost.", german: "Deutlicher Qualitätsgewinn gegenüber low bei etwa doppelten Kosten."}},
	{model: "claude-sonnet-5", level: "high"}:   {publishedRow: 30, passAt1: "48%±5%", averageCost: "7.43", steps: 147, credits: 616.7, tradeoff: localizedText{english: "Best Sonnet 5 quality-cost balance.", german: "Bestes Verhältnis von Qualität und Kosten bei Sonnet 5."}},
	{model: "claude-sonnet-5", level: "xhigh"}:  {publishedRow: 28, passAt1: "50%±3%", averageCost: "11.89", steps: 186, credits: 1010.9, tradeoff: localizedText{english: "Small quality gain over high at substantially higher cost and more steps.", german: "Kleiner Qualitätsgewinn gegenüber high bei deutlich höheren Kosten und mehr Schritten."}},
	{model: "claude-sonnet-5", level: "max"}:    {publishedRow: 24, passAt1: "54%±4%", averageCost: "26.40", steps: 268, credits: 2314.2, tradeoff: localizedText{english: "Highest Sonnet 5 Pass@1, at more than double the cost of xhigh and the most benchmark steps of any bundled run.", german: "Höchstes Sonnet-5-Pass@1, zu mehr als den doppelten Kosten von xhigh und mit den meisten Schritten aller mitgelieferten Läufe."}},
	{model: "claude-sonnet-4.6", level: "high"}: {publishedRow: 37, passAt1: "30%±4%", averageCost: "5.52", steps: 134, credits: 674.1, tradeoff: localizedText{english: "Only benchmarked Sonnet 4.6 setting, and behind Sonnet 5 on both quality and cost.", german: "Einzige gemessene Sonnet-4.6-Einstellung und Sonnet 5 sowohl bei Qualität als auch bei Kosten unterlegen."}},
	{model: "gpt-5.6-sol", level: "low"}:        {publishedRow: 31, passAt1: "45%±2%", averageCost: "1.07", steps: 23, credits: 81.8, tradeoff: localizedText{english: "Lowest-cost GPT 5.6 Sol setting, with lower Pass@1 than medium or high.", german: "Günstigste GPT-5.6-Sol-Einstellung, mit geringerem Pass@1 als medium oder high."}},
	{model: "gpt-5.6-sol", level: "medium"}:     {publishedRow: 16, passAt1: "61%±2%", averageCost: "1.86", steps: 31, credits: 164.4, tradeoff: localizedText{english: "Large quality gain over low for a modest cost increase.", german: "Großer Qualitätsgewinn gegenüber low bei geringem Kostenanstieg."}},
	{model: "gpt-5.6-sol", level: "high"}:       {publishedRow: 9, passAt1: "69%±1%", averageCost: "3.47", steps: 37, credits: 282.1, tradeoff: localizedText{english: "Strong quality-cost balance, close to the model's maximum Pass@1.", german: "Starkes Verhältnis von Qualität und Kosten, nahe am maximalen Pass@1 des Modells."}},
	{model: "gpt-5.6-sol", level: "xhigh"}:      {publishedRow: 5, passAt1: "71%±1%", averageCost: "4.70", steps: 44, credits: 431.0, tradeoff: localizedText{english: "Near-maximum GPT quality at substantially less cost than max.", german: "Nahezu maximale GPT-Qualität zu deutlich geringeren Kosten als max."}},
	{model: "gpt-5.6-sol", level: "max"}:        {publishedRow: 4, passAt1: "73%±3%", averageCost: "8.39", steps: 61, credits: 753.3, tradeoff: localizedText{english: "Highest GPT 5.6 Sol Pass@1, at the highest GPT 5.6 Sol cost.", german: "Höchstes GPT-5.6-Sol-Pass@1, zu den höchsten GPT-5.6-Sol-Kosten."}},
	{model: "gpt-5.6-luna", level: "low"}:       {publishedRow: 41, passAt1: "2%±1%", averageCost: "0.07", steps: 12, credits: 0.8, tradeoff: localizedText{english: "Cheapest and fastest bundled setting, but its Pass@1 is near zero.", german: "Günstigste und schnellste mitgelieferte Einstellung, aber ihr Pass@1 liegt nahe null."}},
	{model: "gpt-5.6-luna", level: "medium"}:    {publishedRow: 40, passAt1: "11%±1%", averageCost: "0.22", steps: 24, credits: 2.8, tradeoff: localizedText{english: "Still far too low a Pass@1 to be a viable setting for real work.", german: "Pass@1 weiterhin viel zu niedrig, um für echte Arbeit brauchbar zu sein."}},
	{model: "gpt-5.6-luna", level: "high"}:      {publishedRow: 32, passAt1: "44%±3%", averageCost: "0.78", steps: 49, credits: 12.9, tradeoff: localizedText{english: "First Luna setting with usable Pass@1, at a very low cost.", german: "Erste Luna-Einstellung mit brauchbarem Pass@1, zu sehr geringen Kosten."}},
	{model: "gpt-5.6-luna", level: "xhigh"}:     {publishedRow: 21, passAt1: "57%±2%", averageCost: "1.54", steps: 71, credits: 27.4, tradeoff: localizedText{english: "Solid Pass@1 for the price, well above high.", german: "Solides Pass@1 für die Kosten, deutlich über high."}},
	{model: "gpt-5.6-luna", level: "max"}:       {publishedRow: 12, passAt1: "67%±4%", averageCost: "3.03", steps: 102, credits: 53.6, tradeoff: localizedText{english: "Highest Luna Pass@1 at a low cost, but its slowest setting by a wide margin.", german: "Höchstes Luna-Pass@1 bei geringen Kosten, aber mit deutlichem Abstand die langsamste Einstellung."}},
	{model: "gpt-5.6-terra", level: "low"}:      {publishedRow: 39, passAt1: "24%±1%", averageCost: "0.43", steps: 21, credits: 24.2, tradeoff: localizedText{english: "Lowest-cost Terra setting, with low Pass@1.", german: "Günstigste Terra-Einstellung, mit niedrigem Pass@1."}},
	{model: "gpt-5.6-terra", level: "medium"}:   {publishedRow: 35, passAt1: "35%±3%", averageCost: "0.58", steps: 25, credits: 35.1, tradeoff: localizedText{english: "Modest quality gain over low for a small cost increase.", german: "Geringer Qualitätsgewinn gegenüber low bei kleinem Kostenanstieg."}},
	{model: "gpt-5.6-terra", level: "high"}:     {publishedRow: 25, passAt1: "54%±4%", averageCost: "1.13", steps: 34, credits: 71.0, tradeoff: localizedText{english: "Strong Pass@1 for a very low cost and few steps.", german: "Starkes Pass@1 bei sehr geringen Kosten und wenigen Schritten."}},
	{model: "gpt-5.6-terra", level: "xhigh"}:    {publishedRow: 17, passAt1: "60%±2%", averageCost: "2.13", steps: 43, credits: 141.7, tradeoff: localizedText{english: "Better Pass@1 than high at roughly double the cost.", german: "Besseres Pass@1 als high bei etwa doppelten Kosten."}},
	{model: "gpt-5.6-terra", level: "max"}:      {publishedRow: 8, passAt1: "70%±3%", averageCost: "4.95", steps: 76, credits: 354.0, tradeoff: localizedText{english: "Highest Terra Pass@1, competitive with far more expensive models.", german: "Höchstes Terra-Pass@1, konkurrenzfähig mit deutlich teureren Modellen."}},
	{model: "gpt-5.5", level: "low"}:            {publishedRow: 38, passAt1: "27%±2%", averageCost: "1.20", steps: 28, credits: 84.6, tradeoff: localizedText{english: "Lowest-cost GPT 5.5 setting, with low Pass@1.", german: "Günstigste GPT-5.5-Einstellung, mit niedrigem Pass@1."}},
	{model: "gpt-5.5", level: "medium"}:         {publishedRow: 23, passAt1: "54%±3%", averageCost: "2.75", steps: 46, credits: 235.7, tradeoff: localizedText{english: "Large quality gain over low for a modest cost increase.", german: "Großer Qualitätsgewinn gegenüber low bei geringem Kostenanstieg."}},
	{model: "gpt-5.5", level: "high"}:           {publishedRow: 15, passAt1: "64%±3%", averageCost: "5.10", steps: 62, credits: 428.3, tradeoff: localizedText{english: "Best GPT 5.5 quality-cost balance.", german: "Bestes Verhältnis von Qualität und Kosten bei GPT 5.5."}},
	{model: "gpt-5.5", level: "xhigh"}:          {publishedRow: 13, passAt1: "67%±6%", averageCost: "7.23", steps: 82, credits: 745.9, tradeoff: localizedText{english: "Highest benchmarked GPT 5.5 Pass@1, at its highest cost.", german: "Höchstes gemessenes GPT-5.5-Pass@1, zu seinen höchsten Kosten."}},
	{model: "gpt-5.4", level: "xhigh"}:          {publishedRow: 26, passAt1: "52%±2%", averageCost: "5.65", steps: 70, credits: 448.5, tradeoff: localizedText{english: "Only benchmarked GPT 5.4 setting, and behind GPT 5.6 on both quality and cost.", german: "Einzige gemessene GPT-5.4-Einstellung und GPT 5.6 sowohl bei Qualität als auch bei Kosten unterlegen."}},
}

func benchmarkForRecommendation(rec Recommendation) (benchmarkEntry, bool) {
	model, ok := benchmarkModelID(rec.Model)
	if !ok {
		return benchmarkEntry{}, false
	}
	level, ok := normalizeEffortLevel(rec.ReasoningSetting)
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
