package recommender

import (
	"strings"
	"testing"
)

func TestRecommendReturnsOneSupportedModelAndProviderSetting(t *testing.T) {
	rec := Recommend("refactor a TypeScript auth module and explain the risk")

	if rec.Model != GPT55 {
		t.Fatalf("expected %s, got %s", GPT55, rec.Model)
	}
	if rec.ReasoningSetting != "GPT reasoning level: high" {
		t.Fatalf("unexpected reasoning setting: %q", rec.ReasoningSetting)
	}
	if rec.Reason == "" {
		t.Fatal("expected a human-readable reason")
	}
}

func TestRecommendSimpleTaskCanUseLowReasoningGPT55(t *testing.T) {
	cases := []string{
		"summarize these release notes",
		"fix a typo in a README",
		"rename a variable in a small Go function",
	}

	for _, task := range cases {
		rec := Recommend(task)

		if rec.Model != GPT55 {
			t.Fatalf("expected %s for %q, got %s", GPT55, task, rec.Model)
		}
		if rec.ReasoningSetting != "GPT reasoning level: low" {
			t.Fatalf("expected low reasoning for %q, got %q", task, rec.ReasoningSetting)
		}
	}
}

func TestRecommendNuancedRoutineTaskUsesGPT55MediumReasoning(t *testing.T) {
	cases := []string{
		"rewrite this support reply to be firm but empathetic",
		"extract requirements from a messy product request",
		"convert inconsistent meeting notes into a clean project plan",
	}

	for _, task := range cases {
		rec := Recommend(task)

		if rec.Model != GPT55 || rec.ReasoningSetting != "GPT reasoning level: medium" {
			t.Fatalf("expected medium-reasoning GPT 5.5 for %q, got %+v", task, rec)
		}
	}
}

func TestRecommendAmbiguousTaskUsesConservativeOfflineDefault(t *testing.T) {
	rec := Recommend("help me with this task")

	if rec.Model != GPT55 {
		t.Fatalf("expected conservative default %s, got %s", GPT55, rec.Model)
	}
	if rec.ReasoningSetting != "GPT reasoning level: medium" {
		t.Fatalf("expected GPT medium reasoning, got %q", rec.ReasoningSetting)
	}
}

func TestZeroValueServiceUsesBundledDefaults(t *testing.T) {
	var svc Service

	rec := svc.Recommend("help me with this task")

	if rec.Model != GPT55 || rec.ReasoningSetting != "GPT reasoning level: medium" {
		t.Fatalf("expected zero-value service to use bundled defaults, got %+v", rec)
	}
}

func TestRecommendCodingTaskAvoidsMediumEffortSonnetDefault(t *testing.T) {
	rec := Recommend("implement a Go API endpoint")

	if rec.Model != GPT55 {
		t.Fatalf("expected %s, got %s", GPT55, rec.Model)
	}
	if rec.ReasoningSetting != "GPT reasoning level: medium" {
		t.Fatalf("expected GPT medium reasoning, got %q", rec.ReasoningSetting)
	}
}

func TestAnthropicRecommendationsUseEffortLevelTerminology(t *testing.T) {
	cases := []struct {
		name       string
		task       string
		preference Preference
		wantModel  string
		wantEffort string
	}{
		{
			name:       "opus default",
			task:       "summarize a long document into a research brief",
			preference: PreferNone,
			wantModel:  Opus48,
			wantEffort: "Anthropic Effort Level: medium",
		},
		{
			name:       "opus quality",
			task:       "summarize a long document into a research brief",
			preference: PreferQuality,
			wantModel:  Opus48,
			wantEffort: "Anthropic Effort Level: high",
		},
		{
			name:       "opus visual design",
			task:       "review this visual design mockup and improve typography",
			preference: PreferNone,
			wantModel:  Opus48,
			wantEffort: "Anthropic Effort Level: medium",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := RecommendWithPreference(tc.task, tc.preference)
			out := Format(rec)

			if rec.Model != tc.wantModel || rec.ReasoningSetting != tc.wantEffort {
				t.Fatalf("expected %s with %q, got %+v", tc.wantModel, tc.wantEffort, rec)
			}
			if strings.Contains(out, "GPT reasoning level") || strings.Contains(strings.ToLower(out), "equivalent terminology") || strings.Contains(strings.ToLower(out), "stronger reasoning") {
				t.Fatalf("Anthropic output used incorrect terminology: %q", out)
			}
		})
	}
}

func TestProviderTerminologyMatchesSelectedModelFamily(t *testing.T) {
	cases := []Recommendation{
		RecommendWithPreference("fix a typo in a README", PreferQuality),
		RecommendWithPreference("implement a Go API endpoint", PreferQuality),
		RecommendWithPreference("summarize a long document into a research brief", PreferCost),
		RecommendWithPreference("analyze a long document and explain complex research tradeoffs", PreferQuality),
		RecommendWithPreference("debug an intermittent distributed race condition", PreferSpeed),
	}

	for _, rec := range cases {
		out := Format(rec)
		switch providerForModel(rec.Model) {
		case providerGPT:
			if !strings.Contains(out, "Reasoning: GPT reasoning level:") || strings.Contains(out, "Anthropic Effort Level") || strings.Contains(strings.ToLower(out), "effort") {
				t.Fatalf("GPT output used incorrect terminology: %+v\n%s", rec, out)
			}
		case providerAnthropic:
			if !strings.Contains(out, "Reasoning: Anthropic Effort Level:") || strings.Contains(out, "GPT reasoning level") {
				t.Fatalf("Anthropic output used incorrect terminology: %+v\n%s", rec, out)
			}
		default:
			t.Fatalf("unsupported model: %s", rec.Model)
		}
	}
}

func TestProviderForModelClassifiesSupportedFamilies(t *testing.T) {
	cases := map[string]providerFamily{
		GPT54:    providerGPT,
		GPT55:    providerGPT,
		Opus48:   providerAnthropic,
		Sonnet46: providerAnthropic,
	}

	for model, want := range cases {
		if got := providerForModel(model); got != want {
			t.Fatalf("providerForModel(%q) = %q, want %q", model, got, want)
		}
	}
}

func TestRecommendComplexDevelopmentTaskRaisesReasoning(t *testing.T) {
	rec := Recommend("debug an intermittent distributed race condition in production")

	if rec.Model != GPT55 || rec.ReasoningSetting != "GPT reasoning level: high" {
		t.Fatalf("expected high-reasoning %s for complex task, got %+v", GPT55, rec)
	}
}

func TestPreferQualityRaisesReasoningWhenTraitsJustifyIt(t *testing.T) {
	rec := RecommendWithPreference("implement a Go API endpoint", PreferQuality)

	if rec.Model != GPT55 {
		t.Fatalf("expected stronger model %s, got %s", GPT55, rec.Model)
	}
	if rec.ReasoningSetting != "GPT reasoning level: high" {
		t.Fatalf("expected quality bias to raise reasoning, got %q", rec.ReasoningSetting)
	}
}

func TestPreferCostKeepsDefaultModelForModerateTask(t *testing.T) {
	rec := RecommendWithPreference("implement a Go API endpoint", PreferCost)

	if rec.Model != GPT55 {
		t.Fatalf("expected default model %s, got %s", GPT55, rec.Model)
	}
}

func TestPreferSpeedKeepsConservativeReasoningForAmbiguousTask(t *testing.T) {
	rec := RecommendWithPreference("help me with this task", PreferSpeed)

	if rec.Model != GPT55 || rec.ReasoningSetting != "GPT reasoning level: medium" {
		t.Fatalf("expected medium-reasoning GPT 5.5, got %+v", rec)
	}
}

func TestPreferSpeedKeepsCodingCapabilityForModerateCodingTask(t *testing.T) {
	rec := RecommendWithPreference("implement a Go API endpoint", PreferSpeed)

	if rec.Model != GPT55 || rec.ReasoningSetting != "GPT reasoning level: medium" {
		t.Fatalf("expected speed preference to keep medium-reasoning GPT 5.5 for coding, got %+v", rec)
	}
}

func TestPreferQualityDoesNotAlwaysSelectHighestCost(t *testing.T) {
	rec := RecommendWithPreference("summarize these release notes", PreferQuality)

	if rec.Model != GPT55 {
		t.Fatalf("expected quality preference to keep GPT 5.5 for simple task, got %s", rec.Model)
	}
	if rec.ReasoningSetting != "GPT reasoning level: low" {
		t.Fatalf("expected low reasoning for simple task, got %q", rec.ReasoningSetting)
	}
}

func TestPreferenceDoesNotOverrideHighRiskComplexity(t *testing.T) {
	rec := RecommendWithPreference("analyze a production authentication incident", PreferCost)

	if rec.Model != GPT55 || rec.ReasoningSetting != "GPT reasoning level: high" {
		t.Fatalf("expected high-risk task to keep high-quality recommendation, got %+v", rec)
	}
}

func TestPreferQualityUsesXHighForHighRiskOrComplexTasks(t *testing.T) {
	cases := []string{
		"analyze a production authentication incident",
		"debug a complex distributed concurrency issue",
	}

	for _, task := range cases {
		rec := RecommendWithPreference(task, PreferQuality)
		if rec.Model != GPT55 || rec.ReasoningSetting != "GPT reasoning level: xhigh" {
			t.Fatalf("expected xhigh GPT 5.5 for %q, got %+v", task, rec)
		}
	}
}

func TestBuiltInRulesDoNotRecommendDeprecatedDefaultModels(t *testing.T) {
	tasks := []string{
		"fix a typo in a README",
		"implement a Go API endpoint",
		"help me with this task",
		"summarize a long document into a research brief",
		"review this visual design mockup and improve typography",
	}
	preferences := []Preference{PreferNone, PreferQuality, PreferCost, PreferSpeed}

	for _, task := range tasks {
		for _, preference := range preferences {
			rec := RecommendWithPreference(task, preference)
			if rec.Model == GPT54 || rec.Model == Sonnet46 {
				t.Fatalf("did not expect %s for %q with preference %q: %+v", rec.Model, task, preference, rec)
			}
		}
	}
}

func TestClassifierUsesTermBoundariesForShortKeywords(t *testing.T) {
	if classify("write an author bio").highRisk {
		t.Fatal("did not expect auth inside author to classify as high risk")
	}
	if classify("write a goal statement").coding {
		t.Fatal("did not expect go inside goal to classify as Go coding work")
	}
	if !classify("refactor a Go API auth module").coding {
		t.Fatal("expected standalone Go/API keywords to classify as coding")
	}
	if !classify("refactor a Go API auth module").highRisk {
		t.Fatal("expected standalone auth keyword to classify as high risk")
	}
}

func TestClassifierRecognizesBroaderModelSelectionSignals(t *testing.T) {
	cases := []struct {
		name string
		task string
		want func(taskTraits) bool
	}{
		{name: "oauth security", task: "audit OAuth token handling for PCI compliance", want: func(traits taskTraits) bool { return traits.highRisk }},
		{name: "frontend coding", task: "build a frontend component backed by a SQL query", want: func(traits taskTraits) bool { return traits.coding }},
		{name: "repo scope", task: "plan a legacy monorepo migration across multiple files", want: func(traits taskTraits) bool { return traits.largeContext }},
		{name: "diagnosis", task: "diagnose a memory leak and optimize the state machine", want: func(traits taskTraits) bool { return traits.deepReasoning }},
		{name: "longform writing", task: "edit this editorial speech for brand voice", want: func(traits taskTraits) bool { return traits.anthropicFit }},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if traits := classify(tc.task); !tc.want(traits) {
				t.Fatalf("expected %q to set requested trait, got %+v", tc.task, traits)
			}
		})
	}
}

func TestParsePreferenceRejectsEmptyOrUnsupportedValues(t *testing.T) {
	for _, value := range []string{"", "cheap", "fastest"} {
		if _, ok := ParsePreference(value); ok {
			t.Fatalf("expected %q to be rejected", value)
		}
	}
}

func TestFormatContainsOneRecommendationOnly(t *testing.T) {
	out := Format(Recommendation{Model: Opus48, ReasoningSetting: "Anthropic Effort Level: high", Reason: "Useful for demanding analysis."})

	assertHumanOnlyOutput(t, out)
}

func TestBuiltInRecommendationsStayWithinHumanOnlyOutputGuardrails(t *testing.T) {
	tasks := []string{
		"fix a typo in a README",
		"rewrite this support reply to be firm but empathetic",
		"implement a Go API endpoint",
		"debug an intermittent distributed race condition in production",
		"summarize a long document into a research brief",
		"analyze a long document and explain complex market analysis tradeoffs",
		"help me with this task",
	}
	preferences := []Preference{PreferNone, PreferQuality, PreferCost, PreferSpeed}

	for _, task := range tasks {
		for _, preference := range preferences {
			out := Format(RecommendWithPreference(task, preference))
			assertHumanOnlyOutput(t, out)
		}
	}
}

func assertHumanOnlyOutput(t *testing.T, out string) {
	t.Helper()

	trimmed := strings.TrimSpace(out)
	lines := strings.Split(trimmed, "\n")
	if len(lines) != 3 {
		t.Fatalf("expected compact three-line output, got %d lines in %q", len(lines), out)
	}
	for _, label := range []string{"Model:", "Reasoning:", "Reason:"} {
		if count := strings.Count(out, label); count != 1 {
			t.Fatalf("expected %s once, got %d in %q", label, count, out)
		}
	}
	if !strings.HasPrefix(lines[0], "Model: ") || !strings.HasPrefix(lines[1], "Reasoning: ") || !strings.HasPrefix(lines[2], "Reason: ") {
		t.Fatalf("output should directly answer with model, reasoning, and reason lines: %q", out)
	}

	lower := strings.ToLower(out)
	for _, forbidden := range []string{
		"```", "{", "}", "[", "]", "|",
		"ranked", "ranking", "option 1", "option 2", "alternative", "runner-up",
		"benchmark", "benchmarks", "latency table", "leaderboard", "live pricing", "current price",
		"$", "per token", "per 1k", "per 1m", "token cost", "exact cost",
		"api key", "credential", "provider setup", "set up an account", "export ",
	} {
		if strings.Contains(lower, forbidden) {
			t.Fatalf("output contains forbidden guardrail term %q: %q", forbidden, out)
		}
	}
}
