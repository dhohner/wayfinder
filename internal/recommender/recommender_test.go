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
	rec := Recommend("summarize these release notes")

	if rec.Model != GPT54 {
		t.Fatalf("expected %s, got %s", GPT54, rec.Model)
	}
	if !strings.HasPrefix(rec.ReasoningSetting, "GPT reasoning level:") {
		t.Fatalf("expected GPT terminology, got %q", rec.ReasoningSetting)
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
