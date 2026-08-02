package recommender

import (
	"math/rand"
	"sort"
	"strings"
	"testing"
)

var allOptimizations = []Optimization{OptimizeValue, OptimizeQuality, OptimizeCost, OptimizeSpeed}

// referenceSelection is an independent implementation of the documented bands.
// The production selector must agree with it, which is what proves the anchors
// are computed from the bundled data rather than written down as constants.
func referenceSelection(t *testing.T, set candidateSet, optimization Optimization) (model, level string) {
	t.Helper()

	type row struct {
		model, level string
		passAt1      int
		credits      float64
		steps        int
		published    int
	}
	var candidates []row
	for key, entry := range bundledBenchmarks {
		passAt1, err := passAt1Points(entry.passAt1)
		if err != nil {
			t.Fatalf("cannot parse pass@1 for %v: %v", key, err)
		}
		if passAt1 < set.passAt1Floor || entry.credits > set.creditCap || entry.steps > set.stepCap {
			continue
		}
		if !set.admits(key.model) {
			continue
		}
		display, ok := displayModelForBenchmarkID(key.model)
		if !ok {
			t.Fatalf("no display name for %q", key.model)
		}
		candidates = append(candidates, row{display, key.level, passAt1, entry.credits, entry.steps, entry.publishedRow})
	}
	if len(candidates) == 0 {
		t.Fatalf("candidate set %q is empty", set.name)
	}

	best := 0
	for _, c := range candidates {
		if c.passAt1 > best {
			best = c.passAt1
		}
	}

	var eligible []row
	var less func(a, b row) bool
	switch optimization {
	case OptimizeQuality:
		eligible = candidates
		less = func(a, b row) bool {
			if a.passAt1 != b.passAt1 {
				return a.passAt1 > b.passAt1
			}
			if a.credits != b.credits {
				return a.credits < b.credits
			}
			if a.steps != b.steps {
				return a.steps < b.steps
			}
			return a.published < b.published
		}
	case OptimizeCost, OptimizeValue:
		for _, c := range candidates {
			if optimization == OptimizeCost || c.passAt1 >= best-valueBandPoints {
				eligible = append(eligible, c)
			}
		}
		less = func(a, b row) bool {
			if a.credits != b.credits {
				return a.credits < b.credits
			}
			if a.passAt1 != b.passAt1 {
				return a.passAt1 > b.passAt1
			}
			if a.steps != b.steps {
				return a.steps < b.steps
			}
			return a.published < b.published
		}
	case OptimizeSpeed:
		for _, c := range candidates {
			if c.passAt1 >= best-speedBandPoints {
				eligible = append(eligible, c)
			}
		}
		less = func(a, b row) bool {
			if a.steps != b.steps {
				return a.steps < b.steps
			}
			if a.passAt1 != b.passAt1 {
				return a.passAt1 > b.passAt1
			}
			if a.credits != b.credits {
				return a.credits < b.credits
			}
			return a.published < b.published
		}
	default:
		t.Fatalf("unsupported optimization %q", optimization)
	}

	sort.Slice(eligible, func(i, j int) bool { return less(eligible[i], eligible[j]) })
	return eligible[0].model, eligible[0].level
}

// TestSubstantiveCodingAnchorsComeFromTheBands pins the four substantive coding
// anchors and proves each one is what the bands compute over the substantive
// candidate set rather than a hardcoded constant.
func TestSubstantiveCodingAnchorsComeFromTheBands(t *testing.T) {
	const task = "implement a small Go API endpoint with OAuth token handling"

	cases := []struct {
		optimization Optimization
		wantModel    string
		wantSetting  string
	}{
		{OptimizeQuality, Opus5, "Anthropic Effort Level: max"},
		{OptimizeValue, GPT56Sol, "GPT reasoning level: high"},
		{OptimizeCost, GPT56Luna, "GPT reasoning level: max"},
		{OptimizeSpeed, GPT56Sol, "GPT reasoning level: high"},
	}

	for _, tc := range cases {
		t.Run(string(tc.optimization), func(t *testing.T) {
			rec := RecommendWithOptimization(task, tc.optimization)
			if rec.Model != tc.wantModel || rec.ReasoningSetting != tc.wantSetting {
				t.Fatalf("expected %s with %q, got %+v", tc.wantModel, tc.wantSetting, rec)
			}

			model, level := referenceSelection(t, substantiveSet, tc.optimization)
			if model != tc.wantModel || !strings.HasSuffix(tc.wantSetting, ": "+level) {
				t.Fatalf("band over the substantive set produced %s[%s], which does not match the pinned anchor %s / %q", model, level, tc.wantModel, tc.wantSetting)
			}
		})
	}
}

func TestEveryCandidateSetAnchorEqualsItsBandResult(t *testing.T) {
	sets := []candidateSet{substantiveSet, anthropicSet, routineSet, simpleSet}

	for _, set := range sets {
		for _, optimization := range allOptimizations {
			t.Run(set.name+"/"+string(optimization), func(t *testing.T) {
				selected := set.selectFor(optimization)
				model, level := referenceSelection(t, set, optimization)
				if selected.displayModel != model || selected.level != level {
					t.Fatalf("selection %s[%s] disagrees with the documented bands %s[%s]", selected.displayModel, selected.level, model, level)
				}
			})
		}
	}
}

// TestAnthropicCategoriesRecommendClaudeOpus5 covers the three categories REQ-03
// routes to Claude: visual, UI, and UX design work, long-form and creative work,
// and review of GPT-authored code.
func TestAnthropicCategoriesRecommendClaudeOpus5(t *testing.T) {
	cases := []struct {
		name    string
		task    string
		against AgainstFamily
	}{
		{"visual design", "create a UI design wireframe for onboarding", AgainstUnspecified},
		{"long-form", "summarize a long document into a research brief", AgainstUnspecified},
		{"creative", "edit this editorial speech for brand voice", AgainstUnspecified},
		{"review against gpt", "review this pull request for bugs", AgainstGPT},
	}
	wantEffort := map[Optimization]string{
		OptimizeQuality: "Anthropic Effort Level: max",
		OptimizeValue:   "Anthropic Effort Level: medium",
		OptimizeCost:    "Anthropic Effort Level: medium",
		OptimizeSpeed:   "Anthropic Effort Level: medium",
	}

	for _, tc := range cases {
		for _, optimization := range allOptimizations {
			t.Run(tc.name+"/"+string(optimization), func(t *testing.T) {
				rec := RecommendWithOptimizationAgainst(tc.task, optimization, tc.against)
				if rec.Model != Opus5 {
					t.Fatalf("expected %s, got %+v", Opus5, rec)
				}
				if rec.ReasoningSetting != wantEffort[optimization] {
					t.Fatalf("expected %q, got %+v", wantEffort[optimization], rec)
				}
			})
		}
	}
}

func TestNoRulePathRecommendsOpus48(t *testing.T) {
	tasks := []string{
		"fix a typo in a README",
		"rename a variable in a small Go function",
		"implement a Go API endpoint",
		"implement a small Go API endpoint with OAuth token handling",
		"debug an intermittent distributed race condition in production",
		"review this pull request for bugs",
		"create a UI design wireframe for onboarding",
		"summarize a long document into a research brief",
		"rewrite this support reply to be firm but empathetic",
		"plan a legacy monorepo migration across multiple files",
		"help me with this task",
	}
	families := []AgainstFamily{AgainstUnspecified, AgainstGPT, AgainstClaude}

	for _, task := range tasks {
		for _, optimization := range allOptimizations {
			for _, against := range families {
				rec := RecommendWithOptimizationAgainst(task, optimization, against)
				if rec.Model == Opus48 {
					t.Fatalf("no rule may recommend %s: %q / %q / %q gave %+v", Opus48, task, optimization, against, rec)
				}
			}
		}
	}

	// Opus 4.8 stays a known, renderable model when supplied directly.
	out, err := FormatJSON(anthropicRecommendation(Opus48, "high", "Supplied directly."), OptimizeQuality, true)
	if err != nil {
		t.Fatalf("expected %s to remain renderable: %v", Opus48, err)
	}
	assertContainsAll(t, out, `"model":"claude-opus-4.8"`, `"reasoning":"high"`)
}

func TestValueBandAdmitsTheExactBoundaryCandidate(t *testing.T) {
	candidates := substantiveSet.candidates()
	best := bestPassAt1(t, candidates)
	if best != 74 {
		t.Fatalf("expected the substantive set's best pass@1 to be 74, got %d", best)
	}

	within := withinBand(candidates, best-valueBandPoints)
	if !containsRow(within, GPT56Sol, "high") {
		t.Fatalf("value band must admit a candidate at exactly %d points", best-valueBandPoints)
	}
	if row, ok := findRow(candidates, GPT56Sol, "high"); !ok || row.passAt1 != best-valueBandPoints {
		t.Fatalf("this boundary case needs a candidate at exactly %d points, got %+v", best-valueBandPoints, row)
	}
}

func TestSpeedBandAdmitsTheExactBoundaryCandidate(t *testing.T) {
	candidates := substantiveSet.candidates()
	best := bestPassAt1(t, candidates)

	boundary, ok := findRow(candidates, GPT55, "high")
	if !ok || boundary.passAt1 != best-speedBandPoints {
		t.Fatalf("this boundary case needs a candidate at exactly %d points, got %+v (found=%v)", best-speedBandPoints, boundary, ok)
	}
	if !containsRow(withinBand(candidates, best-speedBandPoints), GPT55, "high") {
		t.Fatalf("speed band must admit a candidate at exactly %d points", best-speedBandPoints)
	}
}

func TestPassAt1FloorsAndCeilingsAreInclusive(t *testing.T) {
	// gpt-5.6-terra[xhigh] sits at exactly 60% pass@1 and exactly 141.7 credits.
	// gpt-5.6-sol[high] sits at exactly 37 steps. claude-sonnet-5[xhigh] sits at
	// exactly 50% pass@1.
	cases := []struct {
		name         string
		set          candidateSet
		wantModel    string
		wantLevel    string
		wantIncluded bool
	}{
		{"floor 60 admits exactly 60", routineSet, GPT56Terra, "xhigh", true},
		{"floor 50 admits exactly 50", candidateSet{name: "floor50", passAt1Floor: 50, creditCap: noCreditCap, stepCap: noStepCap}, Sonnet5, "xhigh", true},
		{"credit cap admits exactly the cap", candidateSet{name: "credits141.7", passAt1Floor: 50, creditCap: 141.7, stepCap: noStepCap}, GPT56Terra, "xhigh", true},
		{"credit cap excludes just above the cap", candidateSet{name: "credits141.6", passAt1Floor: 50, creditCap: 141.6, stepCap: noStepCap}, GPT56Terra, "xhigh", false},
		{"step cap admits exactly the cap", candidateSet{name: "steps37", passAt1Floor: 50, creditCap: noCreditCap, stepCap: 37}, GPT56Sol, "high", true},
		{"step cap excludes just above the cap", candidateSet{name: "steps36", passAt1Floor: 50, creditCap: noCreditCap, stepCap: 36}, GPT56Sol, "high", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := containsRow(tc.set.candidates(), tc.wantModel, tc.wantLevel); got != tc.wantIncluded {
				t.Fatalf("candidate %s[%s] included=%v, want %v", tc.wantModel, tc.wantLevel, got, tc.wantIncluded)
			}
		})
	}
}

// TestQualityBreaksPassAt1TiesByLowerCredits covers the real tie in the routine
// set: gpt-5.6-sol[high] and claude-opus-5[medium] both score 69%, and quality
// must take the cheaper one. The guard that the tie is the top of the set is
// what makes this a tie-break case rather than an ordinary best-score pick.
func TestQualityBreaksPassAt1TiesByLowerCredits(t *testing.T) {
	candidates := routineSet.candidates()
	sol, okSol := findRow(candidates, GPT56Sol, "high")
	opus, okOpus := findRow(candidates, Opus5, "medium")
	if !okSol || !okOpus {
		t.Fatalf("this tie-break case needs both candidates in the routine set")
	}
	if sol.passAt1 != opus.passAt1 {
		t.Fatalf("expected a pass@1 tie, got %d and %d", sol.passAt1, opus.passAt1)
	}
	if !(sol.credits < opus.credits) {
		t.Fatalf("expected %s[high] to be the cheaper candidate, got %v and %v", GPT56Sol, sol.credits, opus.credits)
	}
	for _, row := range candidates {
		if row.passAt1 > sol.passAt1 {
			t.Fatalf("the tie must be the top of the routine set, found %s[%s] at %d%%", row.displayModel, row.level, row.passAt1)
		}
	}

	selected := routineSet.selectFor(OptimizeQuality)
	if selected.displayModel != GPT56Sol || selected.level != "high" {
		t.Fatalf("quality must break the tie toward lower credits, got %s[%s]", selected.displayModel, selected.level)
	}
}

func TestSelectionIsDeterministicAcrossCandidateOrdering(t *testing.T) {
	tasks := []string{
		"implement a small Go API endpoint with OAuth token handling",
		"implement a Go API endpoint",
		"fix a typo in a README",
		"create a UI design wireframe for onboarding",
		"review this pull request for bugs",
		"help me with this task",
	}

	baseline := map[string]Recommendation{}
	for _, task := range tasks {
		for _, optimization := range allOptimizations {
			baseline[task+"/"+string(optimization)] = RecommendWithOptimization(task, optimization)
		}
	}

	original := orderedCandidates
	defer func() { orderedCandidates = original }()

	shuffler := rand.New(rand.NewSource(1))
	for round := 0; round < 25; round++ {
		shuffled := append([]candidateRow(nil), original...)
		shuffler.Shuffle(len(shuffled), func(i, j int) { shuffled[i], shuffled[j] = shuffled[j], shuffled[i] })
		orderedCandidates = shuffled

		for _, task := range tasks {
			for _, optimization := range allOptimizations {
				got := RecommendWithOptimization(task, optimization)
				want := baseline[task+"/"+string(optimization)]
				if got != want {
					t.Fatalf("round %d: %q with %q changed from %+v to %+v", round, task, optimization, want, got)
				}
			}
		}
	}
}

func TestOrderedCandidatesCoverEveryBundledRowInPublishedOrder(t *testing.T) {
	if len(orderedCandidates) != len(bundledBenchmarks) {
		t.Fatalf("ordered candidates hold %d rows, want %d", len(orderedCandidates), len(bundledBenchmarks))
	}
	for i, row := range orderedCandidates {
		if row.publishedRow != i+1 {
			t.Fatalf("row %d has published order %d", i, row.publishedRow)
		}
	}
	if orderedCandidates[0].benchmarkID != "claude-opus-5" || orderedCandidates[0].level != "max" {
		t.Fatalf("expected the first published row to be claude-opus-5[max], got %+v", orderedCandidates[0])
	}
}

func TestEveryReasonIdentifierResolvesToGuardrailSafeText(t *testing.T) {
	sets := []candidateSet{substantiveSet, anthropicSet, routineSet, simpleSet}

	used := map[reasonID]bool{reasonAmbiguousDefault: true}
	for _, set := range sets {
		for _, optimization := range allOptimizations {
			id := set.reasonFor(optimization)
			used[id] = true

			selected := set.selectFor(optimization)
			out := Format(recommendationFor(selected.displayModel, selected.level, id))
			assertHumanOnlyOutput(t, out)

			lower := strings.ToLower(out)
			assertNotContainsAny(t, lower, "empirically faster", "measured faster", "latency advantage")
			switch providerForModel(selected.displayModel) {
			case providerGPT:
				assertNotContainsAny(t, lower, "effort")
			case providerAnthropic:
				assertNotContainsAny(t, lower, "stronger reasoning", "equivalent terminology")
			}
		}
	}

	for id := range englishReasons {
		if !used[id] {
			t.Fatalf("reason %q is registered but unreachable", id)
		}
	}
	if len(used) != len(englishReasons) {
		t.Fatalf("%d reasons are in use but %d are registered", len(used), len(englishReasons))
	}
}

func bestPassAt1(t *testing.T, candidates []candidateRow) int {
	t.Helper()
	if len(candidates) == 0 {
		t.Fatal("expected a non-empty candidate set")
	}
	best := candidates[0].passAt1
	for _, row := range candidates[1:] {
		if row.passAt1 > best {
			best = row.passAt1
		}
	}
	return best
}

func findRow(candidates []candidateRow, model, level string) (candidateRow, bool) {
	for _, row := range candidates {
		if row.displayModel == model && row.level == level {
			return row, true
		}
	}
	return candidateRow{}, false
}

func containsRow(candidates []candidateRow, model, level string) bool {
	_, ok := findRow(candidates, model, level)
	return ok
}
