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
