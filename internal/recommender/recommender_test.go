package recommender

import (
	"encoding/json"
	"strconv"
	"strings"
	"testing"
)

func TestRecommendReturnsOneSupportedModelAndProviderSetting(t *testing.T) {
	rec := Recommend("refactor a TypeScript auth module and explain the risk")

	if rec.Model != GPT56Sol {
		t.Fatalf("expected %s, got %s", GPT56Sol, rec.Model)
	}
	if rec.ReasoningSetting != "GPT reasoning level: high" {
		t.Fatalf("unexpected reasoning setting: %q", rec.ReasoningSetting)
	}
	if rec.Reason == "" {
		t.Fatal("expected a human-readable reason")
	}
}

func TestRecommendSimpleTaskUsesLowEffortClaudeOpus5(t *testing.T) {
	cases := []string{
		"summarize these release notes",
		"fix a typo in a README",
		"rename a variable in a small Go function",
	}

	for _, task := range cases {
		rec := Recommend(task)

		if rec.Model != Opus5 {
			t.Fatalf("expected %s for %q, got %s", Opus5, task, rec.Model)
		}
		if rec.ReasoningSetting != "Anthropic Effort Level: low" {
			t.Fatalf("expected low effort for %q, got %q", task, rec.ReasoningSetting)
		}
	}
}

func TestRecommendNuancedRoutineTaskUsesGPT56SolHighReasoning(t *testing.T) {
	cases := []string{
		"rewrite this support reply to be firm but empathetic",
		"extract requirements from a messy product request",
		"convert inconsistent meeting notes into a clean project plan",
	}

	for _, task := range cases {
		rec := Recommend(task)

		if rec.Model != GPT56Sol || rec.ReasoningSetting != "GPT reasoning level: high" {
			t.Fatalf("expected high-reasoning GPT 5.6 Sol for %q, got %+v", task, rec)
		}
	}
}

func TestRecommendAmbiguousTaskUsesConservativeOfflineDefault(t *testing.T) {
	rec := Recommend("help me with this task")

	if rec.Model != GPT56Terra {
		t.Fatalf("expected conservative default %s, got %s", GPT56Terra, rec.Model)
	}
	if rec.ReasoningSetting != "GPT reasoning level: xhigh" {
		t.Fatalf("expected GPT xhigh reasoning, got %q", rec.ReasoningSetting)
	}
}

func TestZeroValueServiceUsesBundledDefaults(t *testing.T) {
	var svc Service

	rec := svc.Recommend("help me with this task")

	if rec.Model != GPT56Terra || rec.ReasoningSetting != "GPT reasoning level: xhigh" {
		t.Fatalf("expected zero-value service to use bundled defaults, got %+v", rec)
	}
}

func TestRecommendRoutineCodingTaskUsesHighValueDefault(t *testing.T) {
	rec := Recommend("implement a Go API endpoint")

	if rec.Model != GPT56Sol {
		t.Fatalf("expected %s, got %s", GPT56Sol, rec.Model)
	}
	if rec.ReasoningSetting != "GPT reasoning level: high" {
		t.Fatalf("expected GPT high reasoning, got %q", rec.ReasoningSetting)
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

	if rec.Model != GPT56Sol || rec.ReasoningSetting != "GPT reasoning level: high" {
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
		{OptimizeValue, "Anthropic Effort Level: medium"},
		{OptimizeCost, "Anthropic Effort Level: medium"},
		{OptimizeSpeed, "Anthropic Effort Level: medium"},
		{OptimizeQuality, "Anthropic Effort Level: max"},
	}

	for _, task := range tasks {
		for _, profile := range profiles {
			t.Run(task+"/"+string(profile.optimization), func(t *testing.T) {
				rec := RecommendWithOptimization(task, profile.optimization)
				if rec.Model != Opus5 || rec.ReasoningSetting != profile.wantEffort {
					t.Fatalf("expected %s with %q for %q, got %+v", Opus5, profile.wantEffort, task, rec)
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
			if rec.Model != GPT56Sol || !strings.HasPrefix(rec.ReasoningSetting, "GPT reasoning level:") {
				t.Fatalf("expected GPT coding or reasoning path for %q, got %+v", task, rec)
			}
		})
	}
}

func TestBrandVoiceDoesNotEnterVisualDesignPath(t *testing.T) {
	rec := RecommendWithOptimization("edit this editorial speech for brand voice", OptimizeValue)

	if rec.Model != Opus5 || rec.ReasoningSetting != "Anthropic Effort Level: medium" {
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
			wantModel:    Opus5,
			wantEffort:   "Anthropic Effort Level: medium",
		},
		{
			name:         "opus quality",
			task:         "summarize a long document into a research brief",
			optimization: OptimizeQuality,
			wantModel:    Opus5,
			wantEffort:   "Anthropic Effort Level: max",
		},
		{
			name:         "opus visual design",
			task:         "review this visual design mockup and improve typography",
			optimization: OptimizeValue,
			wantModel:    Opus5,
			wantEffort:   "Anthropic Effort Level: medium",
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
		GPT54:      providerGPT,
		GPT55:      providerGPT,
		GPT56Sol:   providerGPT,
		GPT56Luna:  providerGPT,
		GPT56Terra: providerGPT,
		Opus48:     providerAnthropic,
		Opus5:      providerAnthropic,
		Fable5:     providerAnthropic,
		Sonnet46:   providerAnthropic,
		Sonnet5:    providerAnthropic,
	}

	for model, want := range cases {
		if got := providerForModel(model); got != want {
			t.Fatalf("providerForModel(%q) = %q, want %q", model, got, want)
		}
	}
}

func TestRecommendComplexDevelopmentTaskRaisesReasoning(t *testing.T) {
	rec := Recommend("debug an intermittent distributed race condition in production")

	if rec.Model != GPT56Sol || rec.ReasoningSetting != "GPT reasoning level: high" {
		t.Fatalf("expected high-reasoning %s for complex task, got %+v", GPT56Sol, rec)
	}
}

func TestOptimizeQualityRaisesRoutineCodingToHigh(t *testing.T) {
	rec := RecommendWithOptimization("implement a Go API endpoint", OptimizeQuality)

	if rec.Model != GPT56Sol {
		t.Fatalf("expected stronger model %s, got %s", GPT56Sol, rec.Model)
	}
	if rec.ReasoningSetting != "GPT reasoning level: high" {
		t.Fatalf("expected quality optimization to raise routine coding reasoning, got %q", rec.ReasoningSetting)
	}
}

// TestOptimizeCostPicksTheCheapestRoutineCandidate pins cost mode to the routine
// set's credit anchor. gpt-5.6-terra[xhigh] took this from gpt-5.6-sol[medium]
// in the 2026-08-02 price refresh: 48 credits against 54, one point of pass@1
// lower and still inside the routine ceilings.
func TestOptimizeCostPicksTheCheapestRoutineCandidate(t *testing.T) {
	rec := RecommendWithOptimization("implement a Go API endpoint", OptimizeCost)

	if rec.Model != GPT56Terra || rec.ReasoningSetting != "GPT reasoning level: xhigh" {
		t.Fatalf("expected xhigh-reasoning %s, got %+v", GPT56Terra, rec)
	}
}

func TestOptimizeSpeedKeepsConservativeReasoningForAmbiguousTask(t *testing.T) {
	rec := RecommendWithOptimization("help me with this task", OptimizeSpeed)

	if rec.Model != GPT56Terra || rec.ReasoningSetting != "GPT reasoning level: xhigh" {
		t.Fatalf("expected xhigh-reasoning %s, got %+v", GPT56Terra, rec)
	}
}

func TestOptimizeSpeedKeepsCodingCapabilityForModerateCodingTask(t *testing.T) {
	rec := RecommendWithOptimization("implement a Go API endpoint", OptimizeSpeed)

	if rec.Model != GPT56Sol || rec.ReasoningSetting != "GPT reasoning level: medium" {
		t.Fatalf("expected speed optimization to keep medium-reasoning GPT 5.6 Sol for coding, got %+v", rec)
	}
}

func TestOptimizeQualityDoesNotRaiseSimpleNonCodingTask(t *testing.T) {
	rec := RecommendWithOptimization("summarize these release notes", OptimizeQuality)

	if rec.Model != GPT56Sol {
		t.Fatalf("expected quality optimization to keep GPT 5.6 Sol for simple task, got %s", rec.Model)
	}
	if rec.ReasoningSetting != "GPT reasoning level: medium" {
		t.Fatalf("expected moderate reasoning for simple non-coding task, got %q", rec.ReasoningSetting)
	}
}

func TestOptimizationDoesNotOverrideHighRiskComplexity(t *testing.T) {
	rec := RecommendWithOptimization("analyze a production authentication incident", OptimizeCost)

	// Cost mode still has to clear the substantive pass rate floor, so it drops
	// to the cheapest capable option rather than to a weak one.
	if rec.Model != GPT56Luna || rec.ReasoningSetting != "GPT reasoning level: max" {
		t.Fatalf("expected high-risk task to keep a capable recommendation, got %+v", rec)
	}
}

func TestCodeReviewAgainstChoosesOppositeFamily(t *testing.T) {
	cases := []struct {
		name      string
		against   AgainstFamily
		wantModel string
		wantLevel string
	}{
		{name: "against gpt", against: AgainstGPT, wantModel: Opus5, wantLevel: "Anthropic Effort Level: medium"},
		{name: "against claude", against: AgainstClaude, wantModel: GPT56Sol, wantLevel: "GPT reasoning level: high"},
		{name: "default reviewer", against: AgainstUnspecified, wantModel: GPT56Sol, wantLevel: "GPT reasoning level: high"},
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
		{name: "claude cost", against: AgainstGPT, profile: OptimizeCost, wantModel: Opus5, wantReason: "Anthropic Effort Level: medium"},
		{name: "claude speed", against: AgainstGPT, profile: OptimizeSpeed, wantModel: Opus5, wantReason: "Anthropic Effort Level: medium"},
		{name: "claude value", against: AgainstGPT, profile: OptimizeValue, wantModel: Opus5, wantReason: "Anthropic Effort Level: medium"},
		{name: "claude quality", against: AgainstGPT, profile: OptimizeQuality, wantModel: Opus5, wantReason: "Anthropic Effort Level: max"},
		// Review without --against gpt runs the unrestricted substantive set, so
		// its cost and quality anchors cross family: the bands select whichever
		// bundled option wins the mode, not a fixed reviewer family.
		{name: "gpt cost", against: AgainstClaude, profile: OptimizeCost, wantModel: GPT56Luna, wantReason: "GPT reasoning level: max"},
		{name: "gpt speed", against: AgainstClaude, profile: OptimizeSpeed, wantModel: GPT56Sol, wantReason: "GPT reasoning level: high"},
		{name: "gpt value", against: AgainstClaude, profile: OptimizeValue, wantModel: GPT56Sol, wantReason: "GPT reasoning level: high"},
		{name: "gpt quality", against: AgainstClaude, profile: OptimizeQuality, wantModel: Opus5, wantReason: "Anthropic Effort Level: max"},
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
			if rec.Model != Opus5 || rec.ReasoningSetting != "Anthropic Effort Level: medium" {
				t.Fatalf("expected Claude cross-family code review for %q, got %+v", task, rec)
			}
		})
	}
}

func TestAgainstDoesNotOverrideNonCodeReviewSelection(t *testing.T) {
	rec := RecommendWithOptimizationAgainst("summarize a long document into a research brief", OptimizeValue, AgainstGPT)
	if rec.Model != Opus5 || rec.ReasoningSetting != "Anthropic Effort Level: medium" {
		t.Fatalf("expected normal non-review recommendation to ignore --against, got %+v", rec)
	}

	rec = RecommendWithOptimizationAgainst("fix a typo in a README", OptimizeValue, AgainstGPT)
	if rec.Model != Opus5 || rec.ReasoningSetting != "Anthropic Effort Level: low" {
		t.Fatalf("expected simple recommendation to ignore --against, got %+v", rec)
	}
}

func TestCodeReviewHumanOutputUsesSelectedProviderTerminology(t *testing.T) {
	claudeReview := Format(RecommendWithOptimizationAgainst("code review this Go implementation", OptimizeQuality, AgainstGPT))
	assertContainsAll(t, claudeReview, "Model: Claude Opus 5", "Reasoning: Anthropic Effort Level: max")
	assertNotContainsAny(t, claudeReview, "GPT reasoning level")

	gptReview := Format(RecommendWithOptimizationAgainst("code review this Go implementation", OptimizeValue, AgainstClaude))
	assertContainsAll(t, gptReview, "Model: GPT 5.6 Sol", "Reasoning: GPT reasoning level: high")
	assertNotContainsAny(t, gptReview, "Anthropic Effort Level")
}

func TestCodingBenchmarkOptimizationMatrix(t *testing.T) {
	cases := []struct {
		name         string
		task         string
		optimization Optimization
		wantModel    string
		wantSetting  string
	}{
		{"routine default", "implement a Go API endpoint", OptimizeValue, GPT56Sol, "GPT reasoning level: high"},
		{"routine value", "implement a Go API endpoint", OptimizeValue, GPT56Sol, "GPT reasoning level: high"},
		{"routine cost", "implement a Go API endpoint", OptimizeCost, GPT56Terra, "GPT reasoning level: xhigh"},
		{"routine speed", "implement a Go API endpoint", OptimizeSpeed, GPT56Sol, "GPT reasoning level: medium"},
		{"routine quality", "implement a Go API endpoint", OptimizeQuality, GPT56Sol, "GPT reasoning level: high"},
		{"simple value", "rename a variable in a small Go function", OptimizeValue, Opus5, "Anthropic Effort Level: low"},
		{"simple cost", "rename a variable in a small Go function", OptimizeCost, GPT56Terra, "GPT reasoning level: high"},
		{"simple speed", "rename a variable in a small Go function", OptimizeSpeed, GPT56Sol, "GPT reasoning level: medium"},
		{"simple quality", "rename a variable in a small Go function", OptimizeQuality, GPT56Sol, "GPT reasoning level: medium"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := RecommendWithOptimization(tc.task, tc.optimization)
			if rec.Model != tc.wantModel || rec.ReasoningSetting != tc.wantSetting {
				t.Fatalf("expected %s with %s, got %+v", tc.wantModel, tc.wantSetting, rec)
			}
		})
	}
}

func TestRoutineCodingFeatureWorkUsesHighReasoning(t *testing.T) {
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
		if rec.Model != GPT56Sol || rec.ReasoningSetting != "GPT reasoning level: high" {
			t.Fatalf("expected routine coding feature work to use high reasoning for %q, got %+v", task, rec)
		}
	}
}

func TestCodeReviewFeatureImplementationIsNotClassifiedAsReview(t *testing.T) {
	rec := RecommendWithOptimizationAgainst("add adversarial code review model selection and --against parsing", OptimizeValue, AgainstGPT)

	if rec.Model != GPT56Sol || rec.ReasoningSetting != "GPT reasoning level: high" {
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
		if rec.Model != GPT56Sol || rec.ReasoningSetting != "GPT reasoning level: high" {
			t.Fatalf("expected correctness-heavy task to use substantive coding path for %q, got %+v", task, rec)
		}
	}
}

func TestOptimizeQualityUsesMaxEffortForHighRiskOrComplexTasks(t *testing.T) {
	cases := []string{
		"analyze a production authentication incident",
		"debug a complex distributed concurrency issue",
	}

	for _, task := range cases {
		rec := RecommendWithOptimization(task, OptimizeQuality)
		if rec.Model != Opus5 || rec.ReasoningSetting != "Anthropic Effort Level: max" {
			t.Fatalf("expected max-effort %s for %q, got %+v", Opus5, task, rec)
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
			if rec.Model == GPT54 || rec.Model == Sonnet46 || rec.Model == Opus48 {
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

func TestFormatWithExplanationAddsExactGPT56SolBenchmarkValues(t *testing.T) {
	cases := []struct{ level, passAt1, averageCost string }{
		{"low", "45%±2%", "1.07"}, {"medium", "61%±2%", "1.86"}, {"high", "69%±1%", "3.47"}, {"xhigh", "71%±1%", "4.70"}, {"max", "73%±3%", "8.39"},
	}
	for _, tc := range cases {
		out := FormatWithExplanation(gptRecommendation(GPT56Sol, tc.level, "test recommendation"))
		assertContainsAll(t, out, "Pass@1 "+tc.passAt1, "average cost "+tc.averageCost, "Tradeoff:")
	}
}

func TestFormatWithExplanationAddsExactClaudeBenchmarkValues(t *testing.T) {
	cases := []struct{ level, passAt1, averageCost string }{
		{"low", "41%±1%", "2.29"}, {"medium", "49%±2%", "3.44"}, {"high", "52%±5%", "4.28"}, {"xhigh", "54%±4%", "8.01"}, {"max", "59%±2%", "13.22"},
	}
	for _, tc := range cases {
		out := FormatWithExplanation(anthropicRecommendation(Opus48, tc.level, "test recommendation"))
		assertContainsAll(t, out, "Pass@1 "+tc.passAt1, "average cost "+tc.averageCost, "Tradeoff:")
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
		assertNotContainsAny(t, out, "Benchmark:", "Pass@1", "AIC ", "AIC factor", "60.0", "61%±2%", "49%±2%", "102.5")
	}
}

func TestFormatJSONNormalizesRecommendationAndExactBenchmark(t *testing.T) {
	out, err := FormatJSON(gptRecommendation(GPT56Sol, "high", "Balanced value choice."), OptimizeValue, false)
	if err != nil {
		t.Fatalf("expected JSON format to succeed: %v", err)
	}

	var doc map[string]any
	if err := json.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatalf("expected valid JSON, got %q: %v", out, err)
	}
	if doc["model"] != "gpt-5.6-sol" || doc["reasoning"] != "high" || doc["profile"] != "value" || doc["reason"] != "Balanced value choice." {
		t.Fatalf("unexpected normalized document: %v", doc)
	}
	benchmark, ok := doc["benchmark"].(map[string]any)
	if !ok {
		t.Fatalf("expected benchmark object: %v", doc)
	}
	if benchmark["pass_at_1"] != 0.69 || benchmark["average_cost"] != 3.47 {
		t.Fatalf("unexpected benchmark values: %v", benchmark)
	}
	if _, ok := benchmark["tradeoff"]; ok {
		t.Fatalf("did not expect tradeoff without explain: %v", benchmark)
	}
	assertNotContainsAny(t, out, "GPT reasoning level", "Model:", "Pass@1", "AIC factor")
}

func TestFormatJSONExplainIncludesTradeoff(t *testing.T) {
	out, err := FormatJSON(anthropicRecommendation(Opus48, "high", "Good fit."), OptimizeQuality, true)
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
	if doc.Model != "claude-opus-4.8" || doc.Reasoning != "high" || doc.Profile != "quality" {
		t.Fatalf("unexpected normalized fields: %+v", doc)
	}
	if doc.Benchmark.PassAt1 != 0.52 || doc.Benchmark.Tradeoff == "" {
		t.Fatalf("expected benchmark values and tradeoff: %+v", doc.Benchmark)
	}
}

var benchmarkIDDisplayModels = map[string]string{
	"gpt-5.4":           GPT54,
	"gpt-5.5":           GPT55,
	"gpt-5.6-sol":       GPT56Sol,
	"gpt-5.6-luna":      GPT56Luna,
	"gpt-5.6-terra":     GPT56Terra,
	"claude-opus-4.8":   Opus48,
	"claude-opus-5":     Opus5,
	"claude-fable-5":    Fable5,
	"claude-sonnet-4.6": Sonnet46,
	"claude-sonnet-5":   Sonnet5,
}

func benchmarkRecommendation(t *testing.T, key benchmarkKey) Recommendation {
	t.Helper()

	model, ok := benchmarkIDDisplayModels[key.model]
	if !ok {
		t.Fatalf("test needs display model mapping for %q", key.model)
	}
	switch providerForModel(model) {
	case providerGPT:
		return gptRecommendation(model, key.level, "Benchmark-backed recommendation.")
	case providerAnthropic:
		return anthropicRecommendation(model, key.level, "Benchmark-backed recommendation.")
	default:
		t.Fatalf("bundled model %q has no provider family", model)
		return Recommendation{}
	}
}

func TestFormatJSONCoversEveryBundledExactBenchmark(t *testing.T) {
	for key, entry := range bundledBenchmarks {
		t.Run(key.model+"/"+key.level, func(t *testing.T) {
			rec := benchmarkRecommendation(t, key)

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

func TestEstimatedCreditsAppearInExplanationAndJSONOnlyForExactMatch(t *testing.T) {
	exact := gptRecommendation(GPT56Sol, "high", "Balanced value choice.")

	text := FormatWithExplanation(exact)
	assertContainsAll(t, text,
		"Benchmark: Pass@1 69%±1%; average cost 3.47.",
		"Estimated Copilot AI credits: 282.1 (input and output tokens, estimate).",
		"Tradeoff:",
	)

	document, err := FormatJSON(exact, OptimizeValue, false)
	if err != nil {
		t.Fatalf("expected JSON format to succeed: %v", err)
	}
	benchmark := benchmarkObject(t, document)
	if benchmark["credits_estimate"] != 282.1 {
		t.Fatalf("expected numeric credits_estimate 282.1, got %v in %v", benchmark["credits_estimate"], benchmark)
	}
	if benchmark["pass_at_1"] != 0.69 || benchmark["average_cost"] != 3.47 {
		t.Fatalf("credits_estimate must be additive, leaving existing fields intact: %v", benchmark)
	}
	if _, ok := benchmark["tradeoff"]; ok {
		t.Fatalf("did not expect tradeoff without explain: %v", benchmark)
	}

	missing := gptRecommendation(GPT54, "high", "Unsupported level.")

	missingText := FormatWithExplanation(missing)
	assertHumanOnlyOutput(t, missingText)
	assertNotContainsAny(t, missingText, "Estimated Copilot AI credits", "106.5")

	missingDocument, err := FormatJSON(missing, OptimizeValue, true)
	if err != nil {
		t.Fatalf("expected JSON format to succeed: %v", err)
	}
	var doc map[string]any
	if err := json.Unmarshal([]byte(missingDocument), &doc); err != nil {
		t.Fatalf("expected valid JSON, got %q: %v", missingDocument, err)
	}
	if _, ok := doc["benchmark"]; ok {
		t.Fatalf("did not expect benchmark for missing exact match: %v", doc)
	}
	assertNotContainsAny(t, missingDocument, "credits_estimate", "448.5")
}

// TestCreditEstimateKeepsPublishedPrecision pins two fractional rows against literal
// expected text, so a change that rounded credits to whole numbers cannot pass by
// rounding both sides of the comparison.
func TestCreditEstimateKeepsPublishedPrecision(t *testing.T) {
	cases := []struct {
		rec  Recommendation
		want float64
		text string
	}{
		{gptRecommendation(GPT56Luna, "max", "Cheapest viable setting."), 53.6, "Estimated Copilot AI credits: 53.6 ("},
		{anthropicRecommendation(Opus5, "medium", "Strong quality per credit."), 352.0, "Estimated Copilot AI credits: 352.0 ("},
	}

	for _, tc := range cases {
		assertContainsAll(t, FormatWithExplanation(tc.rec), tc.text)

		document, err := FormatJSON(tc.rec, OptimizeCost, true)
		if err != nil {
			t.Fatalf("expected JSON format to succeed: %v", err)
		}
		if got := benchmarkObject(t, document)["credits_estimate"]; got != tc.want {
			t.Fatalf("credits_estimate = %v, want %v in %q", got, tc.want, document)
		}
	}
}

func TestEveryBundledBenchmarkReportsItsCreditEstimateAsAnEstimate(t *testing.T) {
	for key, entry := range bundledBenchmarks {
		t.Run(key.model+"/"+key.level, func(t *testing.T) {
			rec := benchmarkRecommendation(t, key)
			published := strconv.FormatFloat(entry.credits, 'f', 1, 64)

			text := FormatWithExplanation(rec)
			assertContainsAll(t, text, "Estimated Copilot AI credits: "+published+" (input and output tokens, estimate).", "average cost "+entry.averageCost)
			assertNotContainsAny(t, text, "$")

			document, err := FormatJSON(rec, OptimizeValue, false)
			if err != nil {
				t.Fatalf("expected JSON format to succeed: %v", err)
			}
			if got := benchmarkObject(t, document)["credits_estimate"]; got != entry.credits {
				t.Fatalf("credits_estimate = %v, want %v in %q", got, entry.credits, document)
			}
		})
	}
}

func TestBuiltInExplanationsNeverPresentCreditsAsAPrice(t *testing.T) {
	tasks := []string{
		"implement a Go API endpoint",
		"fix a typo in a README",
		"debug an intermittent distributed race condition in production",
		"create a UI design wireframe for onboarding",
		"help me with this task",
	}

	for _, task := range tasks {
		for _, optimization := range allOptimizations {
			out := FormatWithExplanation(RecommendWithOptimization(task, optimization))

			assertNotContainsAny(t, out, "$")
			if strings.Contains(out, "Estimated Copilot AI credits:") && !strings.Contains(out, "(input and output tokens, estimate).") {
				t.Fatalf("credit figure for %q with %q lacks its estimate qualification:\n%s", task, optimization, out)
			}
		}
	}
}

func benchmarkObject(t *testing.T, document string) map[string]any {
	t.Helper()

	var doc map[string]any
	if err := json.Unmarshal([]byte(document), &doc); err != nil {
		t.Fatalf("expected valid JSON, got %q: %v", document, err)
	}
	benchmark, ok := doc["benchmark"].(map[string]any)
	if !ok {
		t.Fatalf("expected benchmark object in %q", document)
	}
	return benchmark
}

func TestFormatJSONRejectsUnnormalizableRecommendations(t *testing.T) {
	if out, err := FormatJSON(Recommendation{Model: "Mystery", ReasoningSetting: "unknown", Reason: "test"}, OptimizeValue, false); err == nil || out != "" {
		t.Fatalf("expected normalization error and no partial output, got out=%q err=%v", out, err)
	}
	if out, err := FormatJSON(gptRecommendation(GPT56Sol, "medium", "test"), Optimization("cheap"), false); err == nil || out != "" {
		t.Fatalf("expected profile error and no partial output, got out=%q err=%v", out, err)
	}
}

func TestDefaultFormatRemainsBenchmarkFree(t *testing.T) {
	out := Format(gptRecommendation(GPT56Sol, "high", "Balanced value choice."))

	assertHumanOnlyOutput(t, out)
	assertNotContainsAny(t, out, "Pass@1", "AIC", "AIC factor", "Tradeoff", "Benchmark:")
}

func TestSpeedExplanationDoesNotClaimMeasuredLatencyAdvantage(t *testing.T) {
	rec := RecommendWithOptimization("implement a Go API endpoint", OptimizeSpeed)
	out := FormatWithExplanation(rec)
	lower := strings.ToLower(out)

	assertContainsAll(t, out, "Pass@1 61%±2%")
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

// TestMaxEffortIsReachableOnlyWhereItsBandSelectsIt replaces an earlier rule that
// no recommendation may use max effort. Max is now a legitimate pick wherever the
// bands select it, but the credit and step ceilings must keep it unreachable for
// routine and simple work.
func TestMaxEffortIsReachableOnlyWhereItsBandSelectsIt(t *testing.T) {
	capped := []string{
		"implement a Go API endpoint",
		"fix a typo in a README",
		"rewrite this support reply to be firm but empathetic",
		"help me with this task",
	}
	for _, task := range capped {
		for _, optimization := range allOptimizations {
			rec := RecommendWithOptimization(task, optimization)
			if strings.HasSuffix(rec.ReasoningSetting, ": max") {
				t.Fatalf("capped category must not reach max effort for %q with %q: %+v", task, optimization, rec)
			}
		}
	}

	uncapped := []struct {
		task         string
		optimization Optimization
		want         string
	}{
		{"debug an intermittent distributed race condition in production", OptimizeQuality, "Anthropic Effort Level: max"},
		{"debug an intermittent distributed race condition in production", OptimizeCost, "GPT reasoning level: max"},
		{"create a UI design wireframe for onboarding", OptimizeQuality, "Anthropic Effort Level: max"},
	}
	for _, tc := range uncapped {
		rec := RecommendWithOptimization(tc.task, tc.optimization)
		if rec.ReasoningSetting != tc.want {
			t.Fatalf("expected %q for %q with %q, got %+v", tc.want, tc.task, tc.optimization, rec)
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

func TestBundledBenchmarksTranscribeEveryPublishedRow(t *testing.T) {
	const publishedRows = 41
	if len(bundledBenchmarks) != publishedRows {
		t.Fatalf("bundled benchmark table has %d entries, want %d published bench_v2.md rows", len(bundledBenchmarks), publishedRows)
	}

	cases := []struct {
		key  benchmarkKey
		want benchmarkEntry
	}{
		{benchmarkKey{"claude-opus-5", "max"}, benchmarkEntry{passAt1: "74%±4%", averageCost: "11.84", steps: 99, credits: 1383.3}},
		{benchmarkKey{"claude-opus-5", "medium"}, benchmarkEntry{passAt1: "69%±1%", averageCost: "3.29", steps: 52, credits: 352.0}},
		{benchmarkKey{"claude-fable-5", "max"}, benchmarkEntry{passAt1: "70%±4%", averageCost: "21.63", steps: 88, credits: 2425.6}},
		{benchmarkKey{"claude-sonnet-5", "max"}, benchmarkEntry{passAt1: "54%±4%", averageCost: "26.40", steps: 268, credits: 2314.2}},
		{benchmarkKey{"claude-sonnet-4.6", "high"}, benchmarkEntry{passAt1: "30%±4%", averageCost: "5.52", steps: 134, credits: 674.1}},
		{benchmarkKey{"claude-opus-4.8", "max"}, benchmarkEntry{passAt1: "59%±2%", averageCost: "13.22", steps: 120, credits: 1573.6}},
		{benchmarkKey{"gpt-5.6-sol", "high"}, benchmarkEntry{passAt1: "69%±1%", averageCost: "3.47", steps: 37, credits: 282.1}},
		{benchmarkKey{"gpt-5.6-luna", "max"}, benchmarkEntry{passAt1: "67%±4%", averageCost: "3.03", steps: 102, credits: 53.6}},
		{benchmarkKey{"gpt-5.6-luna", "low"}, benchmarkEntry{passAt1: "2%±1%", averageCost: "0.07", steps: 12, credits: 0.8}},
		{benchmarkKey{"gpt-5.6-terra", "high"}, benchmarkEntry{passAt1: "54%±4%", averageCost: "1.13", steps: 34, credits: 71.0}},
		{benchmarkKey{"gpt-5.5", "low"}, benchmarkEntry{passAt1: "27%±2%", averageCost: "1.20", steps: 28, credits: 84.6}},
		{benchmarkKey{"gpt-5.4", "xhigh"}, benchmarkEntry{passAt1: "52%±2%", averageCost: "5.65", steps: 70, credits: 448.5}},
	}

	for _, tc := range cases {
		t.Run(tc.key.model+"/"+tc.key.level, func(t *testing.T) {
			entry, ok := bundledBenchmarks[tc.key]
			if !ok {
				t.Fatalf("expected a bundled entry for %v", tc.key)
			}
			if entry.passAt1 != tc.want.passAt1 || entry.averageCost != tc.want.averageCost {
				t.Fatalf("unexpected pass@1/cost for %v: got %q/%q want %q/%q", tc.key, entry.passAt1, entry.averageCost, tc.want.passAt1, tc.want.averageCost)
			}
			if entry.steps != tc.want.steps || entry.credits != tc.want.credits {
				t.Fatalf("unexpected steps/credits for %v: got %d/%v want %d/%v", tc.key, entry.steps, entry.credits, tc.want.steps, tc.want.credits)
			}
			if entry.tradeoff == "" {
				t.Fatalf("expected tradeoff prose for %v", tc.key)
			}
		})
	}
}

func TestBundledBenchmarkCreditsKeepPublishedPrecision(t *testing.T) {
	// bench_v2.md publishes 35 credit estimates with a non-zero decimal;
	// rounding any of them to whole credits is a defect.
	fractional := map[benchmarkKey]float64{
		{"claude-fable-5", "max"}:     2425.6,
		{"claude-sonnet-5", "max"}:    2314.2,
		{"claude-opus-4.8", "max"}:    1573.6,
		{"claude-fable-5", "xhigh"}:   1464.5,
		{"claude-opus-5", "max"}:      1383.3,
		{"claude-opus-5", "xhigh"}:    1047.9,
		{"claude-sonnet-5", "xhigh"}:  1010.9,
		{"claude-fable-5", "high"}:    987.2,
		{"claude-opus-4.8", "xhigh"}:  929.7,
		{"gpt-5.6-sol", "max"}:        753.3,
		{"gpt-5.5", "xhigh"}:          745.9,
		{"claude-sonnet-4.6", "high"}: 674.1,
		{"claude-fable-5", "medium"}:  627.6,
		{"claude-sonnet-5", "high"}:   616.7,
		{"gpt-5.4", "xhigh"}:          448.5,
		{"gpt-5.5", "high"}:           428.3,
		{"claude-opus-4.8", "medium"}: 381.9,
		{"claude-fable-5", "low"}:     371.4,
		{"claude-sonnet-5", "medium"}: 326.1,
		{"gpt-5.6-sol", "high"}:       282.1,
		{"claude-opus-4.8", "low"}:    247.6,
		{"gpt-5.5", "medium"}:         235.7,
		{"claude-sonnet-5", "low"}:    167.1,
		{"gpt-5.6-sol", "medium"}:     164.4,
		{"claude-opus-5", "low"}:      163.1,
		{"gpt-5.6-terra", "xhigh"}:    141.7,
		{"gpt-5.5", "low"}:            84.6,
		{"gpt-5.6-sol", "low"}:        81.8,
		{"gpt-5.6-luna", "max"}:       53.6,
		{"gpt-5.6-terra", "medium"}:   35.1,
		{"gpt-5.6-luna", "xhigh"}:     27.4,
		{"gpt-5.6-terra", "low"}:      24.2,
		{"gpt-5.6-luna", "high"}:      12.9,
		{"gpt-5.6-luna", "medium"}:    2.8,
		{"gpt-5.6-luna", "low"}:       0.8,
	}

	for key, want := range fractional {
		entry, ok := bundledBenchmarks[key]
		if !ok {
			t.Fatalf("expected a bundled entry for %v", key)
		}
		if entry.credits != want {
			t.Fatalf("credits for %v = %v, want %v", key, entry.credits, want)
		}
	}
}

func TestEveryBundledBenchmarkModelHasProviderFamily(t *testing.T) {
	for key := range bundledBenchmarks {
		model, ok := benchmarkIDDisplayModels[key.model]
		if !ok {
			t.Fatalf("bundled model %q has no display model name", key.model)
		}
		if got, want := benchmarkModelIDOf(t, model), key.model; got != want {
			t.Fatalf("benchmarkModelID(%q) = %q, want %q", model, got, want)
		}
		if providerForModel(model) == providerUnknown {
			t.Fatalf("bundled model %q (%q) has no provider family", key.model, model)
		}
	}
}

func benchmarkModelIDOf(t *testing.T, model string) string {
	t.Helper()

	id, ok := benchmarkModelID(model)
	if !ok {
		t.Fatalf("benchmarkModelID(%q) reported no normalized ID", model)
	}
	return id
}

func TestFormatJSONRendersClaudeOpus5AtMaxEffort(t *testing.T) {
	rec := anthropicRecommendation(Opus5, "max", "Benchmark-backed recommendation.")

	out, err := FormatJSON(rec, OptimizeQuality, false)
	if err != nil {
		t.Fatalf("expected JSON format to succeed: %v", err)
	}

	var doc struct {
		Model     string `json:"model"`
		Reasoning string `json:"reasoning"`
		Benchmark struct {
			PassAt1 float64 `json:"pass_at_1"`
		} `json:"benchmark"`
	}
	if err := json.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatalf("expected valid JSON, got %q: %v", out, err)
	}
	if doc.Model != "claude-opus-5" || doc.Reasoning != "max" {
		t.Fatalf("unexpected normalized fields: %+v", doc)
	}
	if doc.Benchmark.PassAt1 != 0.74 {
		t.Fatalf("expected pass_at_1 0.74, got %v", doc.Benchmark.PassAt1)
	}
	if providerForModel(Opus5) != providerAnthropic {
		t.Fatalf("expected %q in the Anthropic provider family", Opus5)
	}
}

func TestBundledBenchmarksOmitUnpublishedPairs(t *testing.T) {
	// bench_v2.md publishes only claude-sonnet-4.6[high] and gpt-5.4[xhigh].
	unpublished := []benchmarkKey{
		{"claude-sonnet-4.6", "low"},
		{"claude-sonnet-4.6", "max"},
		{"gpt-5.4", "high"},
		{"gpt-5.5", "max"},
	}

	for _, key := range unpublished {
		if _, ok := bundledBenchmarks[key]; ok {
			t.Fatalf("did not expect a bundled entry for unpublished pair %v", key)
		}
	}

	if _, ok := benchmarkForRecommendation(anthropicRecommendation(Sonnet46, "low", "Unpublished pair.")); ok {
		t.Fatalf("expected lookup to report no exact benchmark match for claude-sonnet-4.6[low]")
	}
}

func TestBundledTradeoffProseSurvivesTheOpus5Leaderboard(t *testing.T) {
	solMax := bundledBenchmarks[benchmarkKey{"gpt-5.6-sol", "max"}]
	if strings.Contains(strings.ToLower(solMax.tradeoff), "highest pass@1 in the bundled benchmark") {
		t.Fatalf("gpt-5.6-sol[max] must not claim the highest bundled Pass@1: %q", solMax.tradeoff)
	}

	opus5Max := bundledBenchmarks[benchmarkKey{"claude-opus-5", "max"}]
	if !strings.Contains(strings.ToLower(opus5Max.tradeoff), "highest pass@1 in the bundled benchmark") {
		t.Fatalf("claude-opus-5[max] holds the highest bundled Pass@1: %q", opus5Max.tradeoff)
	}
}
