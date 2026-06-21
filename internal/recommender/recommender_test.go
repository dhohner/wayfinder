package recommender

import (
	"encoding/json"
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

func TestRecommendRoutineCodingTaskUsesMediumValueDefault(t *testing.T) {
	rec := Recommend("implement a Go API endpoint")

	if rec.Model != GPT55 {
		t.Fatalf("expected %s, got %s", GPT55, rec.Model)
	}
	if rec.ReasoningSetting != "GPT reasoning level: medium" {
		t.Fatalf("expected GPT medium reasoning, got %q", rec.ReasoningSetting)
	}
}

func TestRecommendImplementationIssueAboutVisualDesignUsesCodingPath(t *testing.T) {
	task := `# Recommend Claude for visual, UI, and UX tasks

Wayfinder should preserve task-specific model fit for visual design, UI, and UX work by selecting Claude Opus 4.8 at a low starting reasoning level.

This rule is limited to visual and interface-design work such as visual design, UI design, UX design, interaction design, design systems, mockups, wireframes, layout, and typography. Software architecture, system design, and general coding must continue through the GPT coding or reasoning policies.

Acceptance criteria:
- A visual, UI, or UX task selects Claude Opus 4.8 with low reasoning by default.
- Software architecture, system design, and general coding prompts do not enter the visual-design recommendation path.`

	rec := Recommend(task)

	if rec.Model != GPT55 || rec.ReasoningSetting != "GPT reasoning level: medium" {
		t.Fatalf("expected routine coding path for implementation issue, got %+v", rec)
	}
}

func TestVisualDesignOptimizationMatrix(t *testing.T) {
	tasks := []string{
		"review this visual design mockup and improve typography",
		"create a UI design wireframe for onboarding",
		"improve the UX design and interaction design for this checkout flow",
		"audit the UX for onboarding",
		"define a design system layout and color palette",
		"create screen design options using design tokens",
		"develop a visual identity and brand design system",
	}
	profiles := []struct {
		optimization Optimization
		wantEffort   string
	}{
		{OptimizeValue, "Anthropic Effort Level: low"},
		{OptimizeCost, "Anthropic Effort Level: low"},
		{OptimizeSpeed, "Anthropic Effort Level: low"},
		{OptimizeQuality, "Anthropic Effort Level: medium"},
	}

	for _, task := range tasks {
		for _, profile := range profiles {
			t.Run(task+"/"+string(profile.optimization), func(t *testing.T) {
				rec := RecommendWithOptimization(task, profile.optimization)
				if rec.Model != Opus48 || rec.ReasoningSetting != profile.wantEffort {
					t.Fatalf("expected Opus 4.8 with %q for %q, got %+v", profile.wantEffort, task, rec)
				}
			})
		}
	}
}

func TestVisualDesignPathDoesNotCaptureCodingOrTechnicalDesign(t *testing.T) {
	cases := []string{
		"implement the UI design in TypeScript",
		"build the UI in React",
		"fix a layout bug in a frontend component",
		"plan software architecture for a design system service",
		"system design for UI design tooling",
		"technical design for UI design tooling",
		"software design for a UX analytics service",
	}

	for _, task := range cases {
		t.Run(task, func(t *testing.T) {
			rec := RecommendWithOptimization(task, OptimizeValue)
			if rec.Model != GPT55 || !strings.HasPrefix(rec.ReasoningSetting, "GPT reasoning level:") {
				t.Fatalf("expected GPT coding or reasoning path for %q, got %+v", task, rec)
			}
		})
	}
}

func TestBrandVoiceDoesNotEnterVisualDesignPath(t *testing.T) {
	rec := RecommendWithOptimization("edit this editorial speech for brand voice", OptimizeValue)

	if rec.Model != Opus48 || rec.ReasoningSetting != "Anthropic Effort Level: medium" {
		t.Fatalf("expected brand voice writing to stay on long-form Anthropic fit path, got %+v", rec)
	}
}

func TestAnthropicRecommendationsUseEffortLevelTerminology(t *testing.T) {
	cases := []struct {
		name         string
		task         string
		optimization Optimization
		wantModel    string
		wantEffort   string
	}{
		{
			name:         "opus default",
			task:         "summarize a long document into a research brief",
			optimization: OptimizeValue,
			wantModel:    Opus48,
			wantEffort:   "Anthropic Effort Level: medium",
		},
		{
			name:         "opus quality",
			task:         "summarize a long document into a research brief",
			optimization: OptimizeQuality,
			wantModel:    Opus48,
			wantEffort:   "Anthropic Effort Level: high",
		},
		{
			name:         "opus visual design",
			task:         "review this visual design mockup and improve typography",
			optimization: OptimizeValue,
			wantModel:    Opus48,
			wantEffort:   "Anthropic Effort Level: low",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := RecommendWithOptimization(tc.task, tc.optimization)
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
		RecommendWithOptimization("fix a typo in a README", OptimizeQuality),
		RecommendWithOptimization("implement a Go API endpoint", OptimizeQuality),
		RecommendWithOptimization("summarize a long document into a research brief", OptimizeCost),
		RecommendWithOptimization("analyze a long document and explain complex research tradeoffs", OptimizeQuality),
		RecommendWithOptimization("debug an intermittent distributed race condition", OptimizeSpeed),
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

func TestOptimizeQualityRaisesRoutineCodingToHigh(t *testing.T) {
	rec := RecommendWithOptimization("implement a Go API endpoint", OptimizeQuality)

	if rec.Model != GPT55 {
		t.Fatalf("expected stronger model %s, got %s", GPT55, rec.Model)
	}
	if rec.ReasoningSetting != "GPT reasoning level: high" {
		t.Fatalf("expected quality optimization to raise routine coding reasoning, got %q", rec.ReasoningSetting)
	}
}

func TestOptimizeCostKeepsDefaultModelForModerateTask(t *testing.T) {
	rec := RecommendWithOptimization("implement a Go API endpoint", OptimizeCost)

	if rec.Model != GPT55 {
		t.Fatalf("expected default model %s, got %s", GPT55, rec.Model)
	}
}

func TestOptimizeSpeedKeepsConservativeReasoningForAmbiguousTask(t *testing.T) {
	rec := RecommendWithOptimization("help me with this task", OptimizeSpeed)

	if rec.Model != GPT55 || rec.ReasoningSetting != "GPT reasoning level: medium" {
		t.Fatalf("expected medium-reasoning GPT 5.5, got %+v", rec)
	}
}

func TestOptimizeSpeedKeepsCodingCapabilityForModerateCodingTask(t *testing.T) {
	rec := RecommendWithOptimization("implement a Go API endpoint", OptimizeSpeed)

	if rec.Model != GPT55 || rec.ReasoningSetting != "GPT reasoning level: medium" {
		t.Fatalf("expected speed optimization to keep medium-reasoning GPT 5.5 for coding, got %+v", rec)
	}
}

func TestOptimizeQualityDoesNotRaiseSimpleNonCodingTask(t *testing.T) {
	rec := RecommendWithOptimization("summarize these release notes", OptimizeQuality)

	if rec.Model != GPT55 {
		t.Fatalf("expected quality optimization to keep GPT 5.5 for simple task, got %s", rec.Model)
	}
	if rec.ReasoningSetting != "GPT reasoning level: low" {
		t.Fatalf("expected low reasoning for simple non-coding task, got %q", rec.ReasoningSetting)
	}
}

func TestOptimizationDoesNotOverrideHighRiskComplexity(t *testing.T) {
	rec := RecommendWithOptimization("analyze a production authentication incident", OptimizeCost)

	if rec.Model != GPT55 || rec.ReasoningSetting != "GPT reasoning level: high" {
		t.Fatalf("expected high-risk task to keep high-quality recommendation, got %+v", rec)
	}
}

func TestCodeReviewAgainstChoosesOppositeFamily(t *testing.T) {
	cases := []struct {
		name      string
		against   AgainstFamily
		wantModel string
		wantLevel string
	}{
		{name: "against gpt", against: AgainstGPT, wantModel: Opus48, wantLevel: "Anthropic Effort Level: high"},
		{name: "against claude", against: AgainstClaude, wantModel: GPT55, wantLevel: "GPT reasoning level: high"},
		{name: "default reviewer", against: AgainstUnspecified, wantModel: GPT55, wantLevel: "GPT reasoning level: high"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := RecommendWithOptimizationAgainst("perform an adversarial code review of this Go implementation", OptimizeValue, tc.against)
			if rec.Model != tc.wantModel || rec.ReasoningSetting != tc.wantLevel {
				t.Fatalf("expected %s with %q, got %+v", tc.wantModel, tc.wantLevel, rec)
			}
		})
	}
}

func TestCodeReviewOptimizationMatrix(t *testing.T) {
	cases := []struct {
		name       string
		against    AgainstFamily
		profile    Optimization
		wantModel  string
		wantReason string
	}{
		{name: "claude cost", against: AgainstGPT, profile: OptimizeCost, wantModel: Opus48, wantReason: "Anthropic Effort Level: medium"},
		{name: "claude speed", against: AgainstGPT, profile: OptimizeSpeed, wantModel: Opus48, wantReason: "Anthropic Effort Level: medium"},
		{name: "claude value", against: AgainstGPT, profile: OptimizeValue, wantModel: Opus48, wantReason: "Anthropic Effort Level: high"},
		{name: "claude quality", against: AgainstGPT, profile: OptimizeQuality, wantModel: Opus48, wantReason: "Anthropic Effort Level: xhigh"},
		{name: "gpt cost", against: AgainstClaude, profile: OptimizeCost, wantModel: GPT55, wantReason: "GPT reasoning level: medium"},
		{name: "gpt speed", against: AgainstClaude, profile: OptimizeSpeed, wantModel: GPT55, wantReason: "GPT reasoning level: medium"},
		{name: "gpt value", against: AgainstClaude, profile: OptimizeValue, wantModel: GPT55, wantReason: "GPT reasoning level: high"},
		{name: "gpt quality", against: AgainstClaude, profile: OptimizeQuality, wantModel: GPT55, wantReason: "GPT reasoning level: xhigh"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := RecommendWithOptimizationAgainst("review this pull request for bugs", tc.profile, tc.against)
			if rec.Model != tc.wantModel || rec.ReasoningSetting != tc.wantReason {
				t.Fatalf("expected %s with %q, got %+v", tc.wantModel, tc.wantReason, rec)
			}
		})
	}
}

func TestCodeReviewClassifierRecognizesAuditAndSecurityReviewPhrasing(t *testing.T) {
	cases := []string{
		"audit this PR for edge-case bugs",
		"security review this authentication code",
	}

	for _, task := range cases {
		t.Run(task, func(t *testing.T) {
			rec := RecommendWithOptimizationAgainst(task, OptimizeValue, AgainstGPT)
			if rec.Model != Opus48 || rec.ReasoningSetting != "Anthropic Effort Level: high" {
				t.Fatalf("expected Claude cross-family code review for %q, got %+v", task, rec)
			}
		})
	}
}

func TestAgainstDoesNotOverrideNonCodeReviewSelection(t *testing.T) {
	rec := RecommendWithOptimizationAgainst("summarize a long document into a research brief", OptimizeValue, AgainstGPT)
	if rec.Model != Opus48 || rec.ReasoningSetting != "Anthropic Effort Level: medium" {
		t.Fatalf("expected normal non-review recommendation to ignore --against, got %+v", rec)
	}

	rec = RecommendWithOptimizationAgainst("fix a typo in a README", OptimizeValue, AgainstGPT)
	if rec.Model != GPT55 || rec.ReasoningSetting != "GPT reasoning level: low" {
		t.Fatalf("expected simple coding recommendation to ignore --against, got %+v", rec)
	}
}

func TestCodeReviewHumanOutputUsesSelectedProviderTerminology(t *testing.T) {
	claudeReview := Format(RecommendWithOptimizationAgainst("code review this Go implementation", OptimizeQuality, AgainstGPT))
	assertContainsAll(t, claudeReview, "Model: Opus 4.8", "Reasoning: Anthropic Effort Level: xhigh")
	assertNotContainsAny(t, claudeReview, "GPT reasoning level")

	gptReview := Format(RecommendWithOptimizationAgainst("code review this Go implementation", OptimizeQuality, AgainstClaude))
	assertContainsAll(t, gptReview, "Model: GPT 5.5", "Reasoning: GPT reasoning level: xhigh")
	assertNotContainsAny(t, gptReview, "Anthropic Effort Level")
}

func TestCodingBenchmarkOptimizationMatrix(t *testing.T) {
	cases := []struct {
		name         string
		task         string
		optimization Optimization
		want         string
	}{
		{"routine default", "implement a Go API endpoint", OptimizeValue, "GPT reasoning level: medium"},
		{"routine value", "implement a Go API endpoint", OptimizeValue, "GPT reasoning level: medium"},
		{"routine cost", "implement a Go API endpoint", OptimizeCost, "GPT reasoning level: medium"},
		{"routine speed", "implement a Go API endpoint", OptimizeSpeed, "GPT reasoning level: medium"},
		{"routine quality", "implement a Go API endpoint", OptimizeQuality, "GPT reasoning level: high"},
		{"simple value", "rename a variable in a small Go function", OptimizeValue, "GPT reasoning level: low"},
		{"simple cost", "rename a variable in a small Go function", OptimizeCost, "GPT reasoning level: low"},
		{"simple speed", "rename a variable in a small Go function", OptimizeSpeed, "GPT reasoning level: low"},
		{"simple quality", "rename a variable in a small Go function", OptimizeQuality, "GPT reasoning level: medium"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := RecommendWithOptimization(tc.task, tc.optimization)
			if rec.Model != GPT55 || rec.ReasoningSetting != tc.want {
				t.Fatalf("expected %s with %s, got %+v", GPT55, tc.want, rec)
			}
		})
	}
}

func TestRoutineCodingFeatureWorkUsesMediumReasoning(t *testing.T) {
	cases := []string{
		"tune visual design recommendations",
		"add explain mode with benchmark rationale",
		"add optimization modes for recommendations",
		"add a --json CLI flag and write tests for normalized recommendation output",
		"add explain mode with benchmark tradeoff formatting and CLI coverage",
		"extract CLI runner and recommender service into focused packages with tests",
		"broaden classifier signal coverage for examples like OAuth compliance, memory leak diagnosis, visual design, and brand voice editing",
		"tune visual design recommendation rules so technical/system design prompts stay on the coding path",
	}

	for _, task := range cases {
		rec := RecommendWithOptimization(task, OptimizeValue)
		if rec.Model != GPT55 || rec.ReasoningSetting != "GPT reasoning level: medium" {
			t.Fatalf("expected routine coding feature work to use medium reasoning for %q, got %+v", task, rec)
		}
	}
}

func TestCodeReviewFeatureImplementationIsNotClassifiedAsReview(t *testing.T) {
	rec := RecommendWithOptimizationAgainst("add adversarial code review model selection and --against parsing", OptimizeValue, AgainstGPT)

	if rec.Model != GPT55 || rec.ReasoningSetting != "GPT reasoning level: medium" {
		t.Fatalf("expected code-review feature implementation to stay on routine coding path, got %+v", rec)
	}
}

func TestCorrectnessHeavyCodingIsNotSimpleCoding(t *testing.T) {
	cases := []string{
		"make a small parser for typed comparison with stable ordering",
		"quickly fix edge cases for arbitrarily large values without precision loss",
		"rename this function but preserve current behavior and required behavior",
		"ensure stable ordering for arbitrarily large values without precision loss",
		"handle edge cases for large values",
	}

	for _, task := range cases {
		rec := RecommendWithOptimization(task, OptimizeValue)
		if rec.Model != GPT55 || rec.ReasoningSetting != "GPT reasoning level: high" {
			t.Fatalf("expected correctness-heavy task to use substantive coding path for %q, got %+v", task, rec)
		}
	}
}

func TestOptimizeQualityUsesXHighForHighRiskOrComplexTasks(t *testing.T) {
	cases := []string{
		"analyze a production authentication incident",
		"debug a complex distributed concurrency issue",
	}

	for _, task := range cases {
		rec := RecommendWithOptimization(task, OptimizeQuality)
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
	optimizations := []Optimization{OptimizeValue, OptimizeQuality, OptimizeCost, OptimizeSpeed}

	for _, task := range tasks {
		for _, optimization := range optimizations {
			rec := RecommendWithOptimization(task, optimization)
			if rec.Model == GPT54 || rec.Model == Sonnet46 {
				t.Fatalf("did not expect %s for %q with optimization %q: %+v", rec.Model, task, optimization, rec)
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

func TestParseOptimizationRejectsEmptyOrUnsupportedValues(t *testing.T) {
	for _, value := range []string{"", "cheap", "fastest"} {
		if _, ok := ParseOptimization(value); ok {
			t.Fatalf("expected %q to be rejected", value)
		}
	}
}

func TestParseAgainstFamilyRejectsEmptyOrUnsupportedValues(t *testing.T) {
	for _, value := range []string{"", "anthropic", "gemini"} {
		if _, ok := ParseAgainstFamily(value); ok {
			t.Fatalf("expected %q to be rejected", value)
		}
	}
}

func TestFormatContainsOneRecommendationOnly(t *testing.T) {
	out := Format(Recommendation{Model: Opus48, ReasoningSetting: "Anthropic Effort Level: high", Reason: "Useful for demanding analysis."})

	assertHumanOnlyOutput(t, out)
}

func TestFormatWithExplanationAddsExactGPT55BenchmarkValues(t *testing.T) {
	cases := []struct {
		level     string
		passAt1   string
		aic       string
		aicFactor string
	}{
		{level: "low", passAt1: "27%±2%", aic: "28.2", aicFactor: "1.00"},
		{level: "medium", passAt1: "54%±3%", aic: "60.0", aicFactor: "2.13"},
		{level: "high", passAt1: "64%±3%", aic: "93.0", aicFactor: "3.30"},
		{level: "xhigh", passAt1: "67%±6%", aic: "138.0", aicFactor: "4.89"},
	}

	for _, tc := range cases {
		t.Run(tc.level, func(t *testing.T) {
			out := FormatWithExplanation(gptRecommendation(GPT55, tc.level, "test recommendation"))

			assertContainsAll(t, out, "Pass@1 "+tc.passAt1, "AIC "+tc.aic, "AIC factor "+tc.aicFactor, "Tradeoff:")
		})
	}
}

func TestFormatWithExplanationAddsExactClaudeBenchmarkValues(t *testing.T) {
	cases := []struct {
		name      string
		rec       Recommendation
		passAt1   string
		aic       string
		aicFactor string
	}{
		{name: "opus low", rec: anthropicRecommendation(Opus48, "low", "test recommendation"), passAt1: "41%±1%", aic: "72.5", aicFactor: "2.57"},
		{name: "opus medium", rec: anthropicRecommendation(Opus48, "medium", "test recommendation"), passAt1: "49%±2%", aic: "102.5", aicFactor: "3.63"},
		{name: "opus high", rec: anthropicRecommendation(Opus48, "high", "test recommendation"), passAt1: "52%±5%", aic: "125.0", aicFactor: "4.43"},
		{name: "sonnet high", rec: anthropicRecommendation(Sonnet46, "high", "test recommendation"), passAt1: "30%±4%", aic: "114.0", aicFactor: "4.04"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out := FormatWithExplanation(tc.rec)

			assertContainsAll(t, out, "Pass@1 "+tc.passAt1, "AIC "+tc.aic, "AIC factor "+tc.aicFactor, "Tradeoff:")
		})
	}
}

func TestFormatWithExplanationDoesNotApproximateMissingBenchmarkMatch(t *testing.T) {
	cases := []Recommendation{
		gptRecommendation(GPT54, "high", "Unsupported level."),
		anthropicRecommendation(Sonnet46, "medium", "Unsupported level."),
	}

	for _, rec := range cases {
		out := FormatWithExplanation(rec)

		assertHumanOnlyOutput(t, out)
		assertNotContainsAny(t, out, "Benchmark:", "Pass@1", "AIC ", "AIC factor", "60.0", "54%±3%", "49%±2%", "102.5")
	}
}

func TestFormatJSONNormalizesRecommendationAndExactBenchmark(t *testing.T) {
	out, err := FormatJSON(gptRecommendation(GPT55, "high", "Balanced value choice."), OptimizeValue, false)
	if err != nil {
		t.Fatalf("expected JSON format to succeed: %v", err)
	}

	var doc map[string]any
	if err := json.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatalf("expected valid JSON, got %q: %v", out, err)
	}
	if doc["model"] != "gpt-5.5" || doc["reasoning"] != "high" || doc["profile"] != "value" || doc["reason"] != "Balanced value choice." {
		t.Fatalf("unexpected normalized document: %v", doc)
	}
	benchmark, ok := doc["benchmark"].(map[string]any)
	if !ok {
		t.Fatalf("expected benchmark object: %v", doc)
	}
	if benchmark["pass_at_1"] != 0.64 || benchmark["aic"] != 93.0 || benchmark["aic_factor"] != 3.3 {
		t.Fatalf("unexpected benchmark values: %v", benchmark)
	}
	if _, ok := benchmark["tradeoff"]; ok {
		t.Fatalf("did not expect tradeoff without explain: %v", benchmark)
	}
	assertNotContainsAny(t, out, "GPT reasoning level", "Model:", "Pass@1", "AIC factor")
}

func TestFormatJSONExplainIncludesTradeoff(t *testing.T) {
	out, err := FormatJSON(anthropicRecommendation(Opus48, "medium", "Good fit."), OptimizeQuality, true)
	if err != nil {
		t.Fatalf("expected JSON format to succeed: %v", err)
	}

	var doc struct {
		Model     string `json:"model"`
		Reasoning string `json:"reasoning"`
		Profile   string `json:"profile"`
		Benchmark struct {
			PassAt1  float64 `json:"pass_at_1"`
			Tradeoff string  `json:"tradeoff"`
		} `json:"benchmark"`
	}
	if err := json.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatalf("expected valid JSON, got %q: %v", out, err)
	}
	if doc.Model != "claude-opus-4.8" || doc.Reasoning != "medium" || doc.Profile != "quality" {
		t.Fatalf("unexpected normalized fields: %+v", doc)
	}
	if doc.Benchmark.PassAt1 != 0.49 || doc.Benchmark.Tradeoff == "" {
		t.Fatalf("expected benchmark values and tradeoff: %+v", doc.Benchmark)
	}
}

func TestFormatJSONCoversEveryBundledExactBenchmark(t *testing.T) {
	for key, entry := range bundledBenchmarks {
		t.Run(key.model+"/"+key.level, func(t *testing.T) {
			var rec Recommendation
			switch key.model {
			case "gpt-5.4":
				rec = gptRecommendation(GPT54, key.level, "Benchmark-backed recommendation.")
			case "gpt-5.5":
				rec = gptRecommendation(GPT55, key.level, "Benchmark-backed recommendation.")
			case "claude-opus-4.8":
				rec = anthropicRecommendation(Opus48, key.level, "Benchmark-backed recommendation.")
			case "claude-sonnet-4.6":
				rec = anthropicRecommendation(Sonnet46, key.level, "Benchmark-backed recommendation.")
			default:
				t.Fatalf("test needs display model mapping for %q", key.model)
			}

			out, err := FormatJSON(rec, OptimizeValue, true)
			if err != nil {
				t.Fatalf("expected JSON format to succeed: %v", err)
			}
			expected, err := entry.jsonBenchmark(true)
			if err != nil {
				t.Fatalf("expected bundled benchmark to be parseable: %v", err)
			}

			var doc struct {
				Benchmark *jsonBenchmark `json:"benchmark"`
			}
			if err := json.Unmarshal([]byte(out), &doc); err != nil {
				t.Fatalf("expected valid JSON, got %q: %v", out, err)
			}
			if doc.Benchmark == nil {
				t.Fatalf("expected benchmark object for exact match %v in %q", key, out)
			}
			if *doc.Benchmark != *expected {
				t.Fatalf("unexpected benchmark values: got %+v want %+v", doc.Benchmark, expected)
			}
		})
	}
}

func TestFormatJSONOmitsBenchmarkForMissingExactMatch(t *testing.T) {
	cases := []struct {
		rec           Recommendation
		profile       Optimization
		wantModel     string
		wantReasoning string
	}{
		{gptRecommendation(GPT54, "high", "Unsupported level."), OptimizeSpeed, "gpt-5.4", "high"},
		{anthropicRecommendation(Sonnet46, "medium", "Unsupported level."), OptimizeValue, "claude-sonnet-4.6", "medium"},
	}

	for _, tc := range cases {
		out, err := FormatJSON(tc.rec, tc.profile, true)
		if err != nil {
			t.Fatalf("expected JSON format to succeed: %v", err)
		}

		var doc map[string]any
		if err := json.Unmarshal([]byte(out), &doc); err != nil {
			t.Fatalf("expected valid JSON, got %q: %v", out, err)
		}
		if doc["model"] != tc.wantModel || doc["reasoning"] != tc.wantReasoning || doc["profile"] != string(tc.profile) {
			t.Fatalf("unexpected normalized fields: %v", doc)
		}
		if _, ok := doc["benchmark"]; ok {
			t.Fatalf("did not expect benchmark for missing exact match: %v", doc)
		}
	}
}

func TestFormatJSONRejectsUnnormalizableRecommendations(t *testing.T) {
	if out, err := FormatJSON(Recommendation{Model: "Mystery", ReasoningSetting: "unknown", Reason: "test"}, OptimizeValue, false); err == nil || out != "" {
		t.Fatalf("expected normalization error and no partial output, got out=%q err=%v", out, err)
	}
	if out, err := FormatJSON(gptRecommendation(GPT55, "medium", "test"), Optimization("cheap"), false); err == nil || out != "" {
		t.Fatalf("expected profile error and no partial output, got out=%q err=%v", out, err)
	}
}

func TestDefaultFormatRemainsBenchmarkFree(t *testing.T) {
	out := Format(gptRecommendation(GPT55, "high", "Balanced value choice."))

	assertHumanOnlyOutput(t, out)
	assertNotContainsAny(t, out, "Pass@1", "AIC", "AIC factor", "Tradeoff", "Benchmark:")
}

func TestSpeedExplanationDoesNotClaimMeasuredLatencyAdvantage(t *testing.T) {
	rec := RecommendWithOptimization("implement a Go API endpoint", OptimizeSpeed)
	out := FormatWithExplanation(rec)
	lower := strings.ToLower(out)

	assertContainsAll(t, out, "Pass@1 54%±3%")
	assertNotContainsAny(t, lower, "empirically faster", "measured faster", "latency advantage", "the data contains no latency measurements")
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
	optimizations := []Optimization{OptimizeValue, OptimizeQuality, OptimizeCost, OptimizeSpeed}

	for _, task := range tasks {
		for _, optimization := range optimizations {
			out := Format(RecommendWithOptimization(task, optimization))
			assertHumanOnlyOutput(t, out)
		}
	}
}

func assertContainsAll(t *testing.T, got string, wants ...string) {
	t.Helper()
	for _, want := range wants {
		if !strings.Contains(got, want) {
			t.Fatalf("expected output to contain %q, got:\n%s", want, got)
		}
	}
}

func assertNotContainsAny(t *testing.T, got string, wants ...string) {
	t.Helper()
	for _, want := range wants {
		if strings.Contains(got, want) {
			t.Fatalf("expected output not to contain %q, got:\n%s", want, got)
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
