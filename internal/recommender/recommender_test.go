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

func TestRecommendSimpleTaskCanUseLowerCostCandidate(t *testing.T) {
	cases := []string{
		"summarize these release notes",
		"fix a typo in a README",
		"rename a variable in a small Go function",
	}

	for _, task := range cases {
		rec := Recommend(task)

		if rec.Model != GPT54 {
			t.Fatalf("expected %s for %q, got %s", GPT54, task, rec.Model)
		}
		if rec.ReasoningSetting != "GPT reasoning level: low" {
			t.Fatalf("expected low reasoning for %q, got %q", task, rec.ReasoningSetting)
		}
	}
}

func TestRecommendNuancedRoutineTaskUsesGPT55LowReasoning(t *testing.T) {
	cases := []string{
		"rewrite this support reply to be firm but empathetic",
		"extract requirements from a messy product request",
		"convert inconsistent meeting notes into a clean project plan",
	}

	for _, task := range cases {
		rec := Recommend(task)

		if rec.Model != GPT55 || rec.ReasoningSetting != "GPT reasoning level: low" {
			t.Fatalf("expected low-reasoning GPT 5.5 for %q, got %+v", task, rec)
		}
	}
}

func TestRecommendAmbiguousTaskUsesConservativeOfflineDefault(t *testing.T) {
	rec := Recommend("help me with this task")

	if rec.Model != GPT54 {
		t.Fatalf("expected conservative default %s, got %s", GPT54, rec.Model)
	}
	if rec.ReasoningSetting != "GPT reasoning level: medium" {
		t.Fatalf("expected GPT medium reasoning, got %q", rec.ReasoningSetting)
	}
}

func TestRecommendCodingTaskAvoidsMediumEffortSonnetDefault(t *testing.T) {
	rec := Recommend("implement a Go API endpoint")

	if rec.Model != GPT55 {
		t.Fatalf("expected %s, got %s", GPT55, rec.Model)
	}
	if rec.ReasoningSetting != "GPT reasoning level: low" {
		t.Fatalf("expected GPT low reasoning, got %q", rec.ReasoningSetting)
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
			name:       "sonnet default",
			task:       "summarize a long document into a research brief",
			preference: PreferNone,
			wantModel:  Sonnet46,
			wantEffort: "Anthropic Effort Level: medium",
		},
		{
			name:       "opus quality",
			task:       "summarize a long document into a research brief",
			preference: PreferQuality,
			wantModel:  Opus48,
			wantEffort: "Anthropic Effort Level: medium",
		},
		{
			name:       "opus complex",
			task:       "analyze a long document and explain complex market analysis tradeoffs",
			preference: PreferSpeed,
			wantModel:  Opus48,
			wantEffort: "Anthropic Effort Level: high",
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
	if rec.ReasoningSetting != "GPT reasoning level: medium" {
		t.Fatalf("expected quality bias to raise reasoning, got %q", rec.ReasoningSetting)
	}
}

func TestPreferCostCanChooseLowerCostForModerateTask(t *testing.T) {
	rec := RecommendWithPreference("implement a Go API endpoint", PreferCost)

	if rec.Model != GPT54 {
		t.Fatalf("expected lower-cost %s, got %s", GPT54, rec.Model)
	}
}

func TestPreferSpeedLowersReasoningWhenTaskIsNotComplex(t *testing.T) {
	rec := RecommendWithPreference("help me with this task", PreferSpeed)

	if rec.Model != GPT54 || rec.ReasoningSetting != "GPT reasoning level: low" {
		t.Fatalf("expected fast low-reasoning GPT 5.4, got %+v", rec)
	}
}

func TestPreferSpeedKeepsCodingCapabilityForModerateCodingTask(t *testing.T) {
	rec := RecommendWithPreference("implement a Go API endpoint", PreferSpeed)

	if rec.Model != GPT55 || rec.ReasoningSetting != "GPT reasoning level: low" {
		t.Fatalf("expected speed preference to keep low-reasoning GPT 5.5 for coding, got %+v", rec)
	}
}

func TestPreferQualityDoesNotAlwaysSelectHighestCost(t *testing.T) {
	rec := RecommendWithPreference("summarize these release notes", PreferQuality)

	if rec.Model != GPT54 {
		t.Fatalf("expected quality preference to keep cheaper model for simple task, got %s", rec.Model)
	}
	if rec.ReasoningSetting != "GPT reasoning level: medium" {
		t.Fatalf("expected reasoning boost, got %q", rec.ReasoningSetting)
	}
}

func TestPreferenceDoesNotOverrideHighRiskComplexity(t *testing.T) {
	rec := RecommendWithPreference("analyze a production authentication incident", PreferCost)

	if rec.Model != GPT55 || rec.ReasoningSetting != "GPT reasoning level: high" {
		t.Fatalf("expected high-risk task to keep high-quality recommendation, got %+v", rec)
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

	for _, label := range []string{"Model:", "Reasoning:", "Reason:"} {
		if count := strings.Count(out, label); count != 1 {
			t.Fatalf("expected %s once, got %d in %q", label, count, out)
		}
	}
	if strings.Contains(out, "API key") || strings.Contains(out, "provider credentials") {
		t.Fatalf("output should not ask for provider setup: %q", out)
	}
}
