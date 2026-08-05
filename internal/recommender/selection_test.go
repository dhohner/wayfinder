package recommender

import (
	"fmt"
	"math"
	"math/rand"
	"sort"
	"strings"
	"testing"
)

var allOptimizations = []Optimization{OptimizeValue, OptimizeQuality, OptimizeCost, OptimizeSpeed}

// referenceRow is the reference implementation's own row form. It is kept
// separate from candidateRow so the reference shares no structure with the
// production selection it checks.
type referenceRow struct {
	key       benchmarkKey
	passAt1   int
	credits   float64
	steps     int
	published int
}

// referenceBand independently implements the documented band for one
// optimization mode over the candidates handed to it.
func referenceBand(t *testing.T, candidates []referenceRow, optimization Optimization) referenceRow {
	t.Helper()

	if len(candidates) == 0 {
		t.Fatalf("no candidates for %q", optimization)
	}

	best := candidates[0].passAt1
	for _, candidate := range candidates[1:] {
		if candidate.passAt1 > best {
			best = candidate.passAt1
		}
	}

	eligible := make([]referenceRow, 0, len(candidates))
	switch optimization {
	case OptimizeQuality:
		eligible = append(eligible, candidates...)
	case OptimizeValue:
		for _, candidate := range candidates {
			if candidate.passAt1 >= best-valueBandPoints {
				eligible = append(eligible, candidate)
			}
		}
	case OptimizeCost:
		eligible = append(eligible, candidates...)
	case OptimizeSpeed:
		for _, candidate := range candidates {
			if candidate.passAt1 >= best-speedBandPoints {
				eligible = append(eligible, candidate)
			}
		}
	default:
		t.Fatalf("unsupported optimization %q", optimization)
	}

	if optimization == OptimizeValue || optimization == OptimizeCost {
		cheapest := eligible[0].credits
		for _, candidate := range eligible[1:] {
			if candidate.credits < cheapest {
				cheapest = candidate.credits
			}
		}
		cohort := eligible[:0]
		for _, candidate := range eligible {
			if candidate.credits <= cheapest*creditTieBandRatio {
				cohort = append(cohort, candidate)
			}
		}
		eligible = cohort
	}

	sort.Slice(eligible, func(i, j int) bool {
		a, b := eligible[i], eligible[j]
		if optimization == OptimizeSpeed && a.steps != b.steps {
			return a.steps < b.steps
		}
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
	})
	return eligible[0]
}

// referenceSelection independently implements the documented bands over the
// bundled table. It intentionally does not call production selection helpers.
func referenceSelection(t *testing.T, set candidateSet, optimization Optimization) benchmarkKey {
	t.Helper()

	admissible := make([]referenceRow, 0, len(bundledBenchmarks))
	for key, entry := range bundledBenchmarks {
		passAt1, err := passAt1Points(entry.passAt1)
		if err != nil {
			t.Fatalf("cannot parse pass@1 for %v: %v", key, err)
		}
		allowed := len(set.benchmarkIDs) == 0
		for _, benchmarkID := range set.benchmarkIDs {
			if benchmarkID == key.model {
				allowed = true
				break
			}
		}
		if passAt1 < set.passAt1Floor || entry.credits > set.costBudget || !allowed {
			continue
		}
		admissible = append(admissible, referenceRow{key: key, passAt1: passAt1, credits: entry.credits, steps: entry.steps, published: entry.publishedRow})
	}
	if len(admissible) == 0 {
		t.Fatalf("candidate set %q is empty", set.name)
	}

	candidates := admissible
	if optimization != OptimizeCost && set.latencyBudget != noLatencyBudget {
		within := make([]referenceRow, 0, len(admissible))
		for _, candidate := range admissible {
			if candidate.steps <= set.latencyBudget {
				within = append(within, candidate)
			}
		}

		if optimization == OptimizeSpeed {
			candidates = within
		} else {
			// Quality and value release the budget together, and only when the
			// row one of them would answer with from inside it is dominated by a
			// row the budget excluded.
			release := len(within) == 0
			for _, mode := range []Optimization{OptimizeQuality, OptimizeValue} {
				if release {
					break
				}
				kept := referenceBand(t, within, mode)
				for _, excluded := range admissible {
					if excluded.steps > set.latencyBudget && excluded.passAt1 > kept.passAt1 && excluded.credits < kept.credits {
						release = true
						break
					}
				}
			}
			if !release {
				candidates = within
			}
		}
	}
	if len(candidates) == 0 {
		t.Fatalf("candidate set %q has no candidates for %q", set.name, optimization)
	}

	return referenceBand(t, candidates, optimization).key
}

// TestSubstantiveCodingAnchorsComeFromTheBands pins the four substantive coding
// anchors through the public recommendation path.
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
			if rec.Model != tc.wantModel || reasoningSettingText(rec) != tc.wantSetting {
				t.Fatalf("expected %s with %q, got %+v", tc.wantModel, tc.wantSetting, rec)
			}

			key := referenceSelection(t, substantiveSet, tc.optimization)
			model, ok := displayModelForBenchmarkID(key.model)
			if !ok || model != tc.wantModel || !strings.HasSuffix(tc.wantSetting, ": "+key.level) {
				t.Fatalf("independent band calculation returned %s[%s], want %s / %q", key.model, key.level, tc.wantModel, tc.wantSetting)
			}
		})
	}
}

var allCandidateSets = []candidateSet{simpleSet, routineSet, substantiveSet, anthropicSet}

// TestEveryCandidateSetAndModeSelectsThePinnedBenchmarkRow states all sixteen
// answers as bundled benchmark rows, so a change to a band, a budget, or the
// bundled data has to move an expectation here rather than move a
// recommendation silently.
func TestEveryCandidateSetAndModeSelectsThePinnedBenchmarkRow(t *testing.T) {
	pinned := map[string]map[Optimization]benchmarkKey{
		"simple": {
			OptimizeQuality: {"gpt-5.6-luna", "max"},
			OptimizeValue:   {"gpt-5.6-luna", "max"},
			OptimizeCost:    {"gpt-5.6-luna", "xhigh"},
			OptimizeSpeed:   {"gpt-5.6-sol", "medium"},
		},
		"routine": {
			OptimizeQuality: {"gpt-5.6-sol", "high"},
			OptimizeValue:   {"gpt-5.6-sol", "high"},
			OptimizeCost:    {"gpt-5.6-luna", "max"},
			OptimizeSpeed:   {"gpt-5.6-sol", "medium"},
		},
		"substantive": {
			OptimizeQuality: {"claude-opus-5", "max"},
			OptimizeValue:   {"gpt-5.6-sol", "high"},
			OptimizeCost:    {"gpt-5.6-luna", "max"},
			OptimizeSpeed:   {"gpt-5.6-sol", "high"},
		},
		"anthropic": {
			OptimizeQuality: {"claude-opus-5", "max"},
			OptimizeValue:   {"claude-opus-5", "medium"},
			OptimizeCost:    {"claude-opus-5", "medium"},
			OptimizeSpeed:   {"claude-opus-5", "medium"},
		},
	}

	for _, set := range allCandidateSets {
		for _, optimization := range allOptimizations {
			t.Run(set.name+"/"+string(optimization), func(t *testing.T) {
				want, ok := pinned[set.name][optimization]
				if !ok {
					t.Fatalf("no pinned row for %s/%s", set.name, optimization)
				}
				selected := set.selectFor(optimization)
				if selected.benchmarkID != want.model || selected.level != want.level {
					t.Fatalf("%s/%s selected %s[%s], want %s[%s]", set.name, optimization, selected.benchmarkID, selected.level, want.model, want.level)
				}
			})
		}
	}
}

// TestEveryCandidateSetAnchorEqualsItsBandResult checks production selection
// against an independent implementation of the documented bands.
func TestEveryCandidateSetAnchorEqualsItsBandResult(t *testing.T) {
	for _, set := range allCandidateSets {
		for _, optimization := range allOptimizations {
			t.Run(set.name+"/"+string(optimization), func(t *testing.T) {
				selected := set.selectFor(optimization)
				want := referenceSelection(t, set, optimization)
				if selected.benchmarkID != want.model || selected.level != want.level {
					t.Fatalf("selection %s[%s] disagrees with independent band calculation %s[%s]", selected.benchmarkID, selected.level, want.model, want.level)
				}
			})
		}
	}
}

// TestNoBandReturnsADominatedCandidate is the standing guard on the outcome this
// change exists for: a band may not answer with a row that another row meeting
// the same pass rate floor, cost budget, and family restriction beats on both
// pass rate and credits.
//
// The comparison set deliberately ignores the latency budget, so the invariant
// cannot be satisfied by whatever that budget happened to remove. speed is
// exempt: steps are its objective, and refusing a cheaper, higher-scoring row
// that takes three times as many steps is the correct answer there.
func TestNoBandReturnsADominatedCandidate(t *testing.T) {
	modes := []Optimization{OptimizeQuality, OptimizeValue, OptimizeCost}

	for _, set := range allCandidateSets {
		for _, optimization := range modes {
			t.Run(set.name+"/"+string(optimization), func(t *testing.T) {
				selected := set.selectFor(optimization)
				for _, row := range set.admissible() {
					if row.beats(selected) {
						t.Fatalf("candidate set %q in %q mode returned %s[%s] (%d%% pass@1, %.1f credits), which %s[%s] (%d%% pass@1, %.1f credits) beats on both pass rate and credits",
							set.name, optimization,
							selected.benchmarkID, selected.level, selected.passAt1, selected.credits,
							row.benchmarkID, row.level, row.passAt1, row.credits)
					}
				}
			})
		}
	}
}

// TestNoBandReturnsADominatedCandidateOnSyntheticSets extends the same
// invariant past the bundled rows, where the latency release is dormant. Random
// sets exercise the release, so a rule that only avoids dominated answers on the
// data that happens to be bundled fails here.
//
// It also holds the quality/value relation over the same sets, because releasing
// the budget for one mode and not the other is how that relation would break.
func TestNoBandReturnsADominatedCandidateOnSyntheticSets(t *testing.T) {
	generator := rand.New(rand.NewSource(7))
	models := []struct {
		displayModel string
		benchmarkID  string
	}{
		{GPT56Sol, "gpt-5.6-sol"},
		{GPT56Luna, "gpt-5.6-luna"},
		{GPT56Terra, "gpt-5.6-terra"},
		{Opus5, "claude-opus-5"},
	}
	levels := []string{"low", "medium", "high", "xhigh", "max"}

	exercised := 0
	for round := 0; round < 2000; round++ {
		rows := make([]candidateRow, 0, len(models)*len(levels))
		for i := 0; i < 2+generator.Intn(6); i++ {
			model := models[generator.Intn(len(models))]
			rows = append(rows, candidateRow{
				displayModel: model.displayModel,
				benchmarkID:  model.benchmarkID,
				level:        levels[generator.Intn(len(levels))],
				passAt1:      40 + generator.Intn(50),
				credits:      float64(1 + generator.Intn(500)),
				steps:        10 + generator.Intn(100),
				publishedRow: i + 1,
			})
		}
		withSyntheticCandidates(t, rows)

		set := candidateSet{
			name:          fmt.Sprintf("synthetic-%d", round),
			passAt1Floor:  40 + generator.Intn(30),
			costBudget:    float64(1 + generator.Intn(500)),
			latencyBudget: 10 + generator.Intn(100),
		}
		admissible := set.admissible()
		if len(admissible) == 0 {
			continue
		}
		if len(set.withinLatencyBudget(admissible)) != len(admissible) {
			exercised++
		}

		for _, optimization := range []Optimization{OptimizeQuality, OptimizeValue, OptimizeCost} {
			selected := set.selectFor(optimization)
			for _, row := range admissible {
				if row.beats(selected) {
					t.Fatalf("round %d: %q returned %s[%s] (%d%%, %.1f credits) at a latency budget of %d steps, which %s[%s] (%d%%, %.1f credits) beats on both axes",
						round, optimization, selected.benchmarkID, selected.level, selected.passAt1, selected.credits, set.latencyBudget,
						row.benchmarkID, row.level, row.passAt1, row.credits)
				}
			}
		}

		quality := set.selectFor(OptimizeQuality)
		value := set.selectFor(OptimizeValue)
		if quality.passAt1 < value.passAt1 || value.credits > quality.credits {
			t.Fatalf("round %d: quality %s[%s] (%d%%, %.1f credits) and value %s[%s] (%d%%, %.1f credits) inverted",
				round, quality.benchmarkID, quality.level, quality.passAt1, quality.credits, value.benchmarkID, value.level, value.passAt1, value.credits)
		}
	}

	// Without sets whose latency budget actually excludes something, the release
	// is never reached and this test would assert nothing about it.
	if exercised < 100 {
		t.Fatalf("only %d generated sets had a binding latency budget; the release is barely exercised", exercised)
	}
}

// TestQualityAndValueCannotInvert holds the two modes in the relation their
// names promise. It also asserts they were handed the identical candidate set,
// because a release that fired for one and not the other is the way the relation
// would break.
func TestQualityAndValueCannotInvert(t *testing.T) {
	for _, set := range allCandidateSets {
		t.Run(set.name, func(t *testing.T) {
			quality := set.selectFor(OptimizeQuality)
			value := set.selectFor(OptimizeValue)

			if quality.passAt1 < value.passAt1 {
				t.Fatalf("candidate set %q: quality returned %s[%s] at %d%% below value's %s[%s] at %d%%", set.name, quality.benchmarkID, quality.level, quality.passAt1, value.benchmarkID, value.level, value.passAt1)
			}
			if value.credits > quality.credits {
				t.Fatalf("candidate set %q: value returned %s[%s] at %.1f credits above quality's %s[%s] at %.1f", set.name, value.benchmarkID, value.level, value.credits, quality.benchmarkID, quality.level, quality.credits)
			}

			qualityCandidates := set.candidatesFor(OptimizeQuality)
			valueCandidates := set.candidatesFor(OptimizeValue)
			if len(qualityCandidates) != len(valueCandidates) {
				t.Fatalf("candidate set %q handed quality %d candidates and value %d", set.name, len(qualityCandidates), len(valueCandidates))
			}
			for i := range qualityCandidates {
				if qualityCandidates[i] != valueCandidates[i] {
					t.Fatalf("candidate set %q handed quality %+v and value %+v at position %d", set.name, qualityCandidates[i], valueCandidates[i], i)
				}
			}
		})
	}
}

// TestDominatedRowsDoNotBeatEachOtherOnATie pins the strictness of the
// comparison the release and the dominance invariant are both built on.
func TestDominatedRowsDoNotBeatEachOtherOnATie(t *testing.T) {
	cheapLowScore := candidateRow{benchmarkID: "gpt-5.6-luna", level: "xhigh", passAt1: 57, credits: 27.4, steps: 71}
	sameCreditsHigherScore := candidateRow{benchmarkID: "gpt-5.6-luna", level: "max", passAt1: 67, credits: 27.4, steps: 102}
	samePassAt1Cheaper := candidateRow{benchmarkID: "gpt-5.6-terra", level: "high", passAt1: 57, credits: 10.0, steps: 34}

	if cheapLowScore.beats(sameCreditsHigherScore) || sameCreditsHigherScore.beats(cheapLowScore) {
		t.Fatalf("rows tying on credits must not beat each other: %+v and %+v", cheapLowScore, sameCreditsHigherScore)
	}
	if cheapLowScore.beats(samePassAt1Cheaper) || samePassAt1Cheaper.beats(cheapLowScore) {
		t.Fatalf("rows tying on pass rate must not beat each other: %+v and %+v", cheapLowScore, samePassAt1Cheaper)
	}
	if !samePassAt1Cheaper.beats(candidateRow{passAt1: 50, credits: 20.0}) {
		t.Fatalf("a strictly higher pass rate at strictly lower credits must beat the other row")
	}
}

// TestRoutineLatencyBudgetBindsSomething keeps routineLatencyBudget honest: a
// budget that changed no selection would assert a latency-proxy commitment the
// code does not make, which is why the simple category stopped declaring one.
func TestRoutineLatencyBudgetBindsSomething(t *testing.T) {
	bounded := routineSet.selectFor(OptimizeQuality)
	if bounded.benchmarkID != "gpt-5.6-sol" || bounded.level != "high" || bounded.steps != 37 {
		t.Fatalf("routine quality is %s[%s] at %d steps, want gpt-5.6-sol[high] at 37", bounded.benchmarkID, bounded.level, bounded.steps)
	}

	unbounded := routineSet
	unbounded.latencyBudget = noLatencyBudget
	if got := unbounded.selectFor(OptimizeQuality); got.benchmarkID != "gpt-5.6-terra" || got.level != "max" || got.steps != 76 {
		t.Fatalf("without its latency budget routine quality is %s[%s] at %d steps, want gpt-5.6-terra[max] at 76; the budget binds nothing", got.benchmarkID, got.level, got.steps)
	}
}

// TestSimpleLatencyBudgetWasInertWhenItWasRemoved verifies that restoring the
// simple category's 40-step ceiling changes no mode: cost ignores it, speed
// already selects a row inside it, and quality and value release it. A benchmark
// refresh fails this test if that ceiling becomes load-bearing.
func TestSimpleLatencyBudgetWasInertWhenItWasRemoved(t *testing.T) {
	const removedSimpleLatencyBudget = 40

	restored := simpleSet
	restored.latencyBudget = removedSimpleLatencyBudget

	for _, optimization := range allOptimizations {
		t.Run(string(optimization), func(t *testing.T) {
			if got, want := restored.selectFor(optimization), simpleSet.selectFor(optimization); got != want {
				t.Fatalf("restoring the removed %d-step budget changes %q from %s[%s] to %s[%s]", removedSimpleLatencyBudget, optimization, want.benchmarkID, want.level, got.benchmarkID, got.level)
			}
		})
	}
}

// withSyntheticCandidates replaces the bundled rows for the duration of one
// test. The release mechanism is dormant on the bundled data, so the only way to
// exercise it is against rows built to fire it.
func withSyntheticCandidates(t *testing.T, rows []candidateRow) {
	t.Helper()

	original := orderedCandidates
	t.Cleanup(func() { orderedCandidates = original })
	orderedCandidates = rows
}

// TestLatencyBudgetIsReleasedWhenEveryAdmittedRowIsDominated exercises the soft
// release: inside the latency budget sit two rows that a single row outside it
// beats on both pass rate and credits, which is exactly the case where holding
// the budget would make quality and value answer with a dominated row.
func TestLatencyBudgetIsReleasedWhenEveryAdmittedRowIsDominated(t *testing.T) {
	insideMidScore := candidateRow{displayModel: GPT56Terra, benchmarkID: "gpt-5.6-terra", level: "high", passAt1: 55, credits: 300.0, steps: 30, publishedRow: 1}
	insideTopScore := candidateRow{displayModel: GPT56Sol, benchmarkID: "gpt-5.6-sol", level: "medium", passAt1: 60, credits: 400.0, steps: 20, publishedRow: 2}
	outsideDominant := candidateRow{displayModel: GPT56Luna, benchmarkID: "gpt-5.6-luna", level: "max", passAt1: 70, credits: 100.0, steps: 90, publishedRow: 3}
	withSyntheticCandidates(t, []candidateRow{insideMidScore, insideTopScore, outsideDominant})

	set := candidateSet{name: "synthetic", passAt1Floor: 50, costBudget: noCostBudget, latencyBudget: 40, reasons: routineSet.reasons}

	admissible := set.admissible()
	within := set.withinLatencyBudget(admissible)
	if !set.releasesLatencyBudget(admissible, within) {
		t.Fatal("expected every in-budget row to be dominated and the latency budget to be released")
	}

	for _, optimization := range []Optimization{OptimizeQuality, OptimizeValue} {
		if got := set.selectFor(optimization); got != outsideDominant {
			t.Fatalf("%q must select from every admissible row once the budget is released, got %s[%s]", optimization, got.benchmarkID, got.level)
		}
		for _, lang := range allLanguages {
			reason := strings.ToLower(reasonText(set.reasonFor(optimization), lang))
			assertNotContainsAny(t, reason, "step limit", "latency safeguard", "schrittgrenz", "latenzvorg")
		}
	}

	// Speed keeps the budget: it declines the dominant row precisely because that
	// row costs 90 steps.
	if got := set.selectFor(OptimizeSpeed); got != insideTopScore {
		t.Fatalf("speed must stay inside the latency budget, got %s[%s] at %d steps", got.benchmarkID, got.level, got.steps)
	}

	// Cost never applied the budget in the first place, so it reaches the cheapest
	// admissible row whether or not the release fires.
	if got := set.selectFor(OptimizeCost); got != outsideDominant {
		t.Fatalf("cost must never be filtered by the latency budget, got %s[%s] at %.1f credits", got.benchmarkID, got.level, got.credits)
	}
}

// TestLatencyBudgetHoldsWhenTheSelectedRowsAreNotDominated is the other side of
// the release: a dominated in-budget row that no band ever answers with is not a
// reason to release, so quality and value keep the budget and keep their
// undominated in-budget answer.
func TestLatencyBudgetHoldsWhenTheSelectedRowsAreNotDominated(t *testing.T) {
	insideSelected := candidateRow{displayModel: GPT56Sol, benchmarkID: "gpt-5.6-sol", level: "high", passAt1: 80, credits: 100.0, steps: 20, publishedRow: 1}
	insideDominated := candidateRow{displayModel: GPT56Luna, benchmarkID: "gpt-5.6-luna", level: "xhigh", passAt1: 60, credits: 200.0, steps: 30, publishedRow: 2}
	outsideCheaper := candidateRow{displayModel: GPT56Terra, benchmarkID: "gpt-5.6-terra", level: "max", passAt1: 70, credits: 90.0, steps: 90, publishedRow: 3}
	withSyntheticCandidates(t, []candidateRow{insideSelected, insideDominated, outsideCheaper})

	set := candidateSet{name: "mixed-frontier", passAt1Floor: 50, costBudget: noCostBudget, latencyBudget: 40}
	admissible := set.admissible()
	within := set.withinLatencyBudget(admissible)

	if !outsideCheaper.beats(insideDominated) {
		t.Fatal("this case needs an excluded row that dominates an in-budget row nothing selects")
	}
	if set.releasesLatencyBudget(admissible, within) {
		t.Fatal("a dominated row that no band selects must not release the latency budget")
	}
	for _, optimization := range []Optimization{OptimizeQuality, OptimizeValue} {
		if got := set.selectFor(optimization); got != insideSelected {
			t.Fatalf("%q selected %s[%s]; want the undominated in-budget row %s[%s]", optimization, got.benchmarkID, got.level, insideSelected.benchmarkID, insideSelected.level)
		}
	}
}

// TestLatencyBudgetIsReleasedWhenTheSelectedRowIsDominated covers the case a
// whole-set dominance test misses: one cheap in-budget row nothing selects is
// defensible, while the row quality actually answers with is beaten on both pass
// rate and credits by a row the budget excluded.
func TestLatencyBudgetIsReleasedWhenTheSelectedRowIsDominated(t *testing.T) {
	insideTopScore := candidateRow{displayModel: GPT56Sol, benchmarkID: "gpt-5.6-sol", level: "high", passAt1: 70, credits: 100.0, steps: 20, publishedRow: 1}
	insideDefensible := candidateRow{displayModel: GPT56Luna, benchmarkID: "gpt-5.6-luna", level: "xhigh", passAt1: 60, credits: 50.0, steps: 30, publishedRow: 2}
	outsideDominant := candidateRow{displayModel: GPT56Terra, benchmarkID: "gpt-5.6-terra", level: "max", passAt1: 80, credits: 90.0, steps: 90, publishedRow: 3}
	withSyntheticCandidates(t, []candidateRow{insideTopScore, insideDefensible, outsideDominant})

	set := candidateSet{name: "dominated-answer", passAt1Floor: 50, costBudget: noCostBudget, latencyBudget: 40}
	admissible := set.admissible()
	within := set.withinLatencyBudget(admissible)

	if outsideDominant.beats(insideDefensible) {
		t.Fatal("this case needs one in-budget row no excluded row dominates")
	}
	if !outsideDominant.beats(insideTopScore) {
		t.Fatal("this case needs the excluded row to dominate the in-budget quality answer")
	}
	if !set.releasesLatencyBudget(admissible, within) {
		t.Fatal("a dominated in-budget answer must release the latency budget")
	}

	for _, optimization := range []Optimization{OptimizeQuality, OptimizeValue} {
		if got := set.selectFor(optimization); got != outsideDominant {
			t.Fatalf("%q must answer with the dominating row once the budget is released, got %s[%s]", optimization, got.benchmarkID, got.level)
		}
	}
	if got := set.selectFor(OptimizeSpeed); got != insideTopScore {
		t.Fatalf("speed must stay inside the latency budget, got %s[%s] at %d steps", got.benchmarkID, got.level, got.steps)
	}
}

// TestLatencyBudgetIsReleasedForBothModesWhenOnlyValuesAnswerIsDominated pins
// that the release is decided over quality and value together. Quality's
// in-budget answer is defensible here and value's is not, and releasing value
// alone would hand it a row scoring above quality's.
func TestLatencyBudgetIsReleasedForBothModesWhenOnlyValuesAnswerIsDominated(t *testing.T) {
	insideTopScore := candidateRow{displayModel: GPT56Sol, benchmarkID: "gpt-5.6-sol", level: "high", passAt1: 80, credits: 500.0, steps: 20, publishedRow: 1}
	insideValueAnswer := candidateRow{displayModel: GPT56Luna, benchmarkID: "gpt-5.6-luna", level: "xhigh", passAt1: 76, credits: 100.0, steps: 30, publishedRow: 2}
	outsideDominant := candidateRow{displayModel: GPT56Terra, benchmarkID: "gpt-5.6-terra", level: "max", passAt1: 78, credits: 50.0, steps: 90, publishedRow: 3}
	withSyntheticCandidates(t, []candidateRow{insideTopScore, insideValueAnswer, outsideDominant})

	set := candidateSet{name: "value-only-dominated", passAt1Floor: 50, costBudget: noCostBudget, latencyBudget: 40}
	within := set.withinLatencyBudget(set.admissible())

	if got := selectFrom(within, OptimizeQuality); got != insideTopScore {
		t.Fatalf("this case needs quality's in-budget answer to be %s[%s], got %s[%s]", insideTopScore.benchmarkID, insideTopScore.level, got.benchmarkID, got.level)
	}
	if outsideDominant.beats(insideTopScore) {
		t.Fatal("this case needs quality's in-budget answer to be defensible")
	}
	if got := selectFrom(within, OptimizeValue); got != insideValueAnswer || !outsideDominant.beats(got) {
		t.Fatalf("this case needs value's in-budget answer to be the dominated %s[%s], got %s[%s]", insideValueAnswer.benchmarkID, insideValueAnswer.level, got.benchmarkID, got.level)
	}

	quality := set.selectFor(OptimizeQuality)
	value := set.selectFor(OptimizeValue)
	if quality != insideTopScore {
		t.Fatalf("quality must answer with %s[%s] from the released set, got %s[%s]", insideTopScore.benchmarkID, insideTopScore.level, quality.benchmarkID, quality.level)
	}
	if value != outsideDominant {
		t.Fatalf("value must answer with %s[%s] from the released set, got %s[%s]", outsideDominant.benchmarkID, outsideDominant.level, value.benchmarkID, value.level)
	}
	if quality.passAt1 < value.passAt1 || value.credits > quality.credits {
		t.Fatalf("quality %s[%s] (%d%%, %.1f credits) and value %s[%s] (%d%%, %.1f credits) inverted", quality.benchmarkID, quality.level, quality.passAt1, quality.credits, value.benchmarkID, value.level, value.passAt1, value.credits)
	}
}

// With nothing inside the budget there is no row left to defend, so quality and
// value fall back to every row admitted by the floor and cost budget.
func TestLatencyBudgetThatAdmitsNothingCannotMakeACategoryUnanswerable(t *testing.T) {
	cheaper := candidateRow{displayModel: GPT56Luna, benchmarkID: "gpt-5.6-luna", level: "xhigh", passAt1: 57, credits: 27.4, steps: 71, publishedRow: 1}
	stronger := candidateRow{displayModel: GPT56Sol, benchmarkID: "gpt-5.6-sol", level: "high", passAt1: 69, credits: 282.1, steps: 80, publishedRow: 2}
	withSyntheticCandidates(t, []candidateRow{cheaper, stronger})

	set := candidateSet{name: "unreachable-latency", passAt1Floor: 50, costBudget: noCostBudget, latencyBudget: 10}

	if got := set.selectFor(OptimizeQuality); got != stronger {
		t.Fatalf("quality must fall back to the rows the floor and cost budget admit, got %s[%s]", got.benchmarkID, got.level)
	}
	if got := set.selectFor(OptimizeValue); got != stronger {
		t.Fatalf("value must fall back to the rows the floor and cost budget admit, got %s[%s]", got.benchmarkID, got.level)
	}
	if got := set.selectFor(OptimizeCost); got != cheaper {
		t.Fatalf("cost must reach the cheapest admissible row, got %s[%s]", got.benchmarkID, got.level)
	}
}

// TestCandidateSetWithNoAdmissibleRowFailsLoudly keeps the build-time contract
// that an over-tightened floor or cost budget is a defect to be reported, not a
// silently empty answer.
func TestCandidateSetWithNoAdmissibleRowFailsLoudly(t *testing.T) {
	set := candidateSet{name: "impossible", passAt1Floor: 99, costBudget: 1.0, latencyBudget: noLatencyBudget}

	defer func() {
		recovered := recover()
		message, ok := recovered.(string)
		if !ok {
			t.Fatalf("expected a string panic, got %v", recovered)
		}
		if !strings.Contains(message, `"impossible"`) {
			t.Fatalf("panic must name the candidate set, got %q", message)
		}
	}()

	set.selectFor(OptimizeValue)
	t.Fatal("expected an empty candidate set to panic")
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
				if reasoningSettingText(rec) != wantEffort[optimization] {
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
	candidates := substantiveSet.candidatesFor(OptimizeValue)
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
	candidates := substantiveSet.candidatesFor(OptimizeSpeed)
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
	//
	// The latency-budget cases run in speed mode, the one mode the budget
	// constrains strictly and unconditionally, so the boundary they assert is the
	// budget's own and not the outcome of the soft release.
	routineWithoutLatency := routineSet
	routineWithoutLatency.latencyBudget = noLatencyBudget

	cases := []struct {
		name         string
		set          candidateSet
		optimization Optimization
		wantModel    string
		wantLevel    string
		wantIncluded bool
	}{
		{"floor 60 admits exactly 60", routineWithoutLatency, OptimizeQuality, GPT56Terra, "xhigh", true},
		{"floor 50 admits exactly 50", candidateSet{name: "floor50", passAt1Floor: 50, costBudget: noCostBudget, latencyBudget: noLatencyBudget}, OptimizeQuality, Sonnet5, "xhigh", true},
		{"cost budget admits exactly the budget", candidateSet{name: "credits141.7", passAt1Floor: 50, costBudget: 141.7, latencyBudget: noLatencyBudget}, OptimizeQuality, GPT56Terra, "xhigh", true},
		{"cost budget excludes just above the budget", candidateSet{name: "credits141.6", passAt1Floor: 50, costBudget: 141.6, latencyBudget: noLatencyBudget}, OptimizeQuality, GPT56Terra, "xhigh", false},
		{"latency budget admits exactly the budget", candidateSet{name: "steps37", passAt1Floor: 50, costBudget: noCostBudget, latencyBudget: 37}, OptimizeSpeed, GPT56Sol, "high", true},
		{"latency budget excludes just above the budget", candidateSet{name: "steps36", passAt1Floor: 50, costBudget: noCostBudget, latencyBudget: 36}, OptimizeSpeed, GPT56Sol, "high", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := containsRow(tc.set.candidatesFor(tc.optimization), tc.wantModel, tc.wantLevel); got != tc.wantIncluded {
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
	candidates := routineSet.candidatesFor(OptimizeQuality)
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

// TestCreditTieCohortSeparatesNearEqualPricesByPassRate proves the boundary
// behavior of the credit tie cohort. The rule is inert on the bundled rows, so
// every cohort there holds one candidate and the bundled table cannot exercise
// the cohort at all; these cases are built to sit on and around the threshold.
//
// boundary is computed from a float64 anchor exactly as withinCreditTieCohort
// computes it, and the inside and outside cases step one ULP off it. Writing the
// boundary as the literal 110.0 would understate it: the float64 nearest 1.10 is
// a hair above 1.10, so the evaluated product is 110.00000000000001 and a
// hand-written 110.0 lands just inside the cohort rather than on its edge. The
// separate 110.0 case below pins that inclusion.
func TestCreditTieCohortSeparatesNearEqualPricesByPassRate(t *testing.T) {
	anchorCredits := 100.0
	boundary := anchorCredits * creditTieBandRatio

	cases := []struct {
		name         string
		rows         []candidateRow
		optimization Optimization
		wantModel    string
		wantLevel    string
	}{
		{
			name: "a sub-percent credit gap loses to three pass rate points",
			rows: []candidateRow{
				{displayModel: Opus5, benchmarkID: "claude-opus-5", level: "low", passAt1: 58, credits: 163.1, steps: 36, publishedRow: 1},
				{displayModel: GPT56Sol, benchmarkID: "gpt-5.6-sol", level: "medium", passAt1: 61, credits: 164.4, steps: 31, publishedRow: 2},
			},
			optimization: OptimizeValue,
			wantModel:    "gpt-5.6-sol",
			wantLevel:    "medium",
		},
		{
			name: "a candidate at exactly 110.0 against a 100.0 anchor is inside the cohort",
			rows: []candidateRow{
				{displayModel: GPT56Luna, benchmarkID: "gpt-5.6-luna", level: "xhigh", passAt1: 58, credits: anchorCredits, steps: 71, publishedRow: 1},
				{displayModel: GPT56Sol, benchmarkID: "gpt-5.6-sol", level: "high", passAt1: 61, credits: 110.0, steps: 37, publishedRow: 2},
			},
			optimization: OptimizeCost,
			wantModel:    "gpt-5.6-sol",
			wantLevel:    "high",
		},
		{
			name: "a candidate exactly at the evaluated boundary is inside the cohort",
			rows: []candidateRow{
				{displayModel: GPT56Luna, benchmarkID: "gpt-5.6-luna", level: "xhigh", passAt1: 58, credits: anchorCredits, steps: 71, publishedRow: 1},
				{displayModel: GPT56Sol, benchmarkID: "gpt-5.6-sol", level: "high", passAt1: 61, credits: boundary, steps: 37, publishedRow: 2},
			},
			optimization: OptimizeCost,
			wantModel:    "gpt-5.6-sol",
			wantLevel:    "high",
		},
		{
			name: "a candidate one ULP below the boundary is inside the cohort",
			rows: []candidateRow{
				{displayModel: GPT56Luna, benchmarkID: "gpt-5.6-luna", level: "xhigh", passAt1: 58, credits: anchorCredits, steps: 71, publishedRow: 1},
				{displayModel: GPT56Sol, benchmarkID: "gpt-5.6-sol", level: "high", passAt1: 61, credits: math.Nextafter(boundary, 0), steps: 37, publishedRow: 2},
			},
			optimization: OptimizeCost,
			wantModel:    "gpt-5.6-sol",
			wantLevel:    "high",
		},
		{
			name: "a candidate one ULP above the boundary is outside the cohort",
			rows: []candidateRow{
				{displayModel: GPT56Luna, benchmarkID: "gpt-5.6-luna", level: "xhigh", passAt1: 58, credits: anchorCredits, steps: 71, publishedRow: 1},
				{displayModel: GPT56Sol, benchmarkID: "gpt-5.6-sol", level: "high", passAt1: 61, credits: math.Nextafter(boundary, math.Inf(1)), steps: 37, publishedRow: 2},
			},
			optimization: OptimizeCost,
			wantModel:    "gpt-5.6-luna",
			wantLevel:    "xhigh",
		},
		{
			name: "a cohort of one answers with the cheapest candidate",
			rows: []candidateRow{
				{displayModel: GPT56Luna, benchmarkID: "gpt-5.6-luna", level: "xhigh", passAt1: 58, credits: anchorCredits, steps: 71, publishedRow: 1},
				{displayModel: Opus5, benchmarkID: "claude-opus-5", level: "max", passAt1: 74, credits: 2 * anchorCredits, steps: 99, publishedRow: 2},
			},
			optimization: OptimizeCost,
			wantModel:    "gpt-5.6-luna",
			wantLevel:    "xhigh",
		},
		{
			name: "equal credits sit in the same cohort",
			rows: []candidateRow{
				{displayModel: GPT56Luna, benchmarkID: "gpt-5.6-luna", level: "xhigh", passAt1: 58, credits: anchorCredits, steps: 71, publishedRow: 1},
				{displayModel: GPT56Sol, benchmarkID: "gpt-5.6-sol", level: "high", passAt1: 61, credits: anchorCredits, steps: 37, publishedRow: 2},
			},
			optimization: OptimizeCost,
			wantModel:    "gpt-5.6-sol",
			wantLevel:    "high",
		},
		{
			name: "a full tie inside the cohort resolves by published row order",
			rows: []candidateRow{
				{displayModel: GPT56Sol, benchmarkID: "gpt-5.6-sol", level: "high", passAt1: 61, credits: anchorCredits, steps: 37, publishedRow: 1},
				{displayModel: GPT56Luna, benchmarkID: "gpt-5.6-luna", level: "max", passAt1: 61, credits: anchorCredits, steps: 37, publishedRow: 2},
			},
			optimization: OptimizeCost,
			wantModel:    "gpt-5.6-sol",
			wantLevel:    "high",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			withSyntheticCandidates(t, tc.rows)
			set := candidateSet{name: "credit-cohort", passAt1Floor: 50, costBudget: noCostBudget, latencyBudget: noLatencyBudget}

			if got := set.selectFor(tc.optimization); got.benchmarkID != tc.wantModel || got.level != tc.wantLevel {
				t.Fatalf("%q selected %s[%s] at %.20g credits, want %s[%s]", tc.optimization, got.benchmarkID, got.level, got.credits, tc.wantModel, tc.wantLevel)
			}
		})
	}
}

func TestCostReasonsDoNotClaimTheTieCohortWinnerIsExactlyCheapest(t *testing.T) {
	for _, set := range allCandidateSets {
		for _, lang := range allLanguages {
			reason := strings.ToLower(reasonText(set.reasonFor(OptimizeCost), lang))
			assertNotContainsAny(t, reason, "cheapest", "günstigste wahl", "günstigste einstellung")
		}
	}
	for _, lang := range allLanguages {
		reason := strings.ToLower(reasonText(reasonAmbiguousDefault, lang))
		assertNotContainsAny(t, reason, "cheapest", "günstigste wahl", "günstigste einstellung")
	}
}

func TestCreditTieCohortRejectsInvalidCredits(t *testing.T) {
	cases := []float64{-1, math.NaN(), math.Inf(1)}
	for _, credits := range cases {
		t.Run(fmt.Sprint(credits), func(t *testing.T) {
			defer func() {
				message, ok := recover().(string)
				if !ok || !strings.Contains(message, "finite and non-negative") {
					t.Fatalf("expected diagnostic invalid-credit panic, got %v", message)
				}
			}()
			withinCreditTieCohort([]candidateRow{{benchmarkID: "invalid", level: "test", credits: credits}})
			t.Fatal("expected invalid credits to panic")
		})
	}
}

// TestCreditTieCohortIsAnchoredNotPairwise pins the anchor: a chain of rows each
// within creditTieBandRatio of its neighbour but not of the cheapest row must not
// pull the far end into the cohort, because a pairwise epsilon would make the
// cohort depend on comparison order.
func TestCreditTieCohortIsAnchoredNotPairwise(t *testing.T) {
	rows := []candidateRow{
		{displayModel: GPT56Luna, benchmarkID: "gpt-5.6-luna", level: "xhigh", passAt1: 58, credits: 100.0, steps: 71, publishedRow: 1},
		{displayModel: GPT56Sol, benchmarkID: "gpt-5.6-sol", level: "medium", passAt1: 61, credits: 109.0, steps: 31, publishedRow: 2},
		{displayModel: Opus5, benchmarkID: "claude-opus-5", level: "max", passAt1: 74, credits: 118.0, steps: 99, publishedRow: 3},
	}

	cohort := withinCreditTieCohort(rows)
	if len(cohort) != 2 || cohort[0].publishedRow != 1 || cohort[1].publishedRow != 2 {
		t.Fatalf("the cohort must hold the two rows within 10%% of the cheapest, got %d rows: %+v", len(cohort), cohort)
	}

	withSyntheticCandidates(t, rows)
	set := candidateSet{name: "anchored-cohort", passAt1Floor: 50, costBudget: noCostBudget, latencyBudget: noLatencyBudget}
	if got := set.selectFor(OptimizeCost); got.benchmarkID != "gpt-5.6-sol" || got.level != "medium" {
		t.Fatalf("cost must answer with the best pass rate inside the anchored cohort, got %s[%s] at %.1f credits", got.benchmarkID, got.level, got.credits)
	}
}

// TestCreditTieCohortIsInertOnTheBundledRows is the verification the tuning
// comment relies on: over the bundled table the 10% cohort returns the same rows
// as the strict cheapest-first ordering it replaced, which is what makes this
// rule a correctness guard rather than a retuning. It covers the value and cost
// bands, the only two that compare credits.
//
// The pre-change ordering is written out here rather than kept as production
// code, so a refresh that makes the cohort load-bearing fails here and has to be
// acknowledged instead of moving a recommendation silently.
func TestCreditTieCohortIsInertOnTheBundledRows(t *testing.T) {
	cheapestFirst := []rowKey{creditsAscending, passAt1Descending, stepsAscending, publishedOrder}

	for _, set := range allCandidateSets {
		for _, optimization := range []Optimization{OptimizeValue, OptimizeCost} {
			t.Run(set.name+"/"+string(optimization), func(t *testing.T) {
				banded := set.selectFor(optimization)

				candidates := set.candidatesFor(optimization)
				if optimization == OptimizeValue {
					candidates = withinBand(candidates, bestPassAt1(t, candidates)-valueBandPoints)
				}

				cheapest := candidates[0]
				for _, row := range candidates[1:] {
					if lessBy(cheapestFirst, row, cheapest) {
						cheapest = row
					}
				}
				if banded != cheapest {
					t.Fatalf("the cohort moved %s/%s from %s[%s] at %.1f credits to %s[%s] at %.1f", set.name, optimization, cheapest.benchmarkID, cheapest.level, cheapest.credits, banded.benchmarkID, banded.level, banded.credits)
				}
			})
		}
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

	selections := make([]selection, 0, len(sets)*len(allOptimizations)+1)
	for _, set := range sets {
		for _, optimization := range allOptimizations {
			selections = append(selections, set.selectionFor(optimization))
		}
	}
	selections = append(selections, ambiguousDefaultSelection())

	used := map[reasonID]bool{}
	for _, sel := range selections {
		used[sel.reason] = true

		for _, lang := range allLanguages {
			out := Format(sel.localize(lang))
			assertHumanOnlyOutput(t, out, lang)

			lower := strings.ToLower(out)
			assertNotContainsAny(t, lower, "empirically faster", "measured faster", "latency advantage")
			assertNotContainsAny(t, lower, "empirisch schneller", "gemessen schneller", "latenzvorteil")
			switch providerForModel(sel.row.displayModel) {
			case providerGPT:
				// Effort is Anthropic's terminology in both languages, so a GPT
				// recommendation must not carry it in either.
				assertNotContainsAny(t, lower, "effort")
			case providerAnthropic:
				assertNotContainsAny(t, lower, "stronger reasoning", "equivalent terminology")
				assertNotContainsAny(t, lower, "stärkeres reasoning", "entsprechende terminologie")
			}
		}
	}

	for id := range reasonCopy {
		if !used[id] {
			t.Fatalf("reason %q is registered but unreachable", id)
		}
	}
	if len(used) != len(reasonCopy) {
		t.Fatalf("%d reasons are in use but %d are registered", len(used), len(reasonCopy))
	}
}

// longRunDisclosures are the phrases that count as telling a reader the answer
// runs long, per language.
var longRunDisclosures = map[language][]string{
	languageEnglish: {"more steps", "longer run", "past the step count"},
	languageGerman:  {"mehr Schritte", "mehr Schritten", "längeren Lauf", "Schrittzahl"},
}

// TestCheapCategoryAnswersDiscloseTheirLongerRun covers answers for cheap or
// simple work that run long: simple quality, value, and cost, routine cost, and
// the ambiguous default. Each is checked against the fewest-step row its set
// admits before its copy is held to disclosing that trade.
func TestCheapCategoryAnswersDiscloseTheirLongerRun(t *testing.T) {
	// The multiple of the fewest-step admissible row at which a run stops being
	// an implementation detail and has to be stated in the recommendation.
	const longRunStepRatio = 2.0

	cases := []struct {
		name string
		// set is where the fewest-step baseline is measured: the set the
		// selection's row was drawn from.
		set       candidateSet
		selection selection
	}{
		{"simple/quality", simpleSet, simpleSet.selectionFor(OptimizeQuality)},
		{"simple/value", simpleSet, simpleSet.selectionFor(OptimizeValue)},
		{"simple/cost", simpleSet, simpleSet.selectionFor(OptimizeCost)},
		{"routine/cost", routineSet, routineSet.selectionFor(OptimizeCost)},
		{"ambiguous/default", routineSet, ambiguousDefaultSelection()},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			selected := tc.selection.row
			fewest := selected.steps
			for _, row := range tc.set.admissible() {
				if row.steps < fewest {
					fewest = row.steps
				}
			}
			if float64(selected.steps) < float64(fewest)*longRunStepRatio {
				t.Fatalf("%s selected %s[%s] at %d steps against %d for the fewest-step row admissible to %s, which is no longer the long run the copy discloses",
					tc.name, selected.benchmarkID, selected.level, selected.steps, fewest, tc.set.name)
			}

			for _, lang := range allLanguages {
				reason := reasonText(tc.selection.reason, lang)
				if !containsAny(reason, longRunDisclosures[lang]) {
					t.Fatalf("%s in %s answers at %d steps without disclosing the longer run: %q", tc.name, lang, selected.steps, reason)
				}
				if strings.Count(reason, "\n") != 0 {
					t.Fatalf("%s in %s spans more than one line: %q", tc.name, lang, reason)
				}
			}
		})
	}
}

// TestSimpleCopyDoesNotPromiseAShortAnswer holds the other half of the
// disclosure: the modes that now run long must not also describe themselves as
// short or fast. Simple speed is exempt because it is the mode that still
// answers with the fewest-step row.
func TestSimpleCopyDoesNotPromiseAShortAnswer(t *testing.T) {
	promises := map[language][]string{
		languageEnglish: {"short", "fast", "quick"},
		languageGerman:  {"kurz", "schnell", "zügig"},
	}

	for _, optimization := range []Optimization{OptimizeQuality, OptimizeValue, OptimizeCost} {
		for _, lang := range allLanguages {
			reason := strings.ToLower(reasonText(simpleSet.reasonFor(optimization), lang))
			for _, promise := range promises[lang] {
				if strings.Contains(reason, promise) {
					t.Fatalf("simple/%s in %s promises %q while answering at %d steps: %q", optimization, lang, promise, simpleSet.selectFor(optimization).steps, reason)
				}
			}
		}
	}
}

// TestAnthropicValueCopyStatesTheActualCreditFraction guards the only piece of
// reason copy that does arithmetic on the bundled figures. The copy names a
// simple fraction of what the quality answer costs, which is honest only while
// it is the nearest simple fraction to the observed ratio. Nothing else here
// compares two categories' credits, so a refresh that moves the ratio would
// otherwise leave the sentence stale without failing anything, which is how the
// earlier claim of a third survived a change of credit basis.
func TestAnthropicValueCopyStatesTheActualCreditFraction(t *testing.T) {
	// Midpoints to the neighbouring simple fractions 1/5 and 1/3: outside them,
	// a different fraction describes the ratio better than a quarter does.
	const (
		nearestToAQuarterFrom = (1.0/5.0 + 1.0/4.0) / 2
		nearestToAQuarterTo   = (1.0/4.0 + 1.0/3.0) / 2
	)

	claimed := map[language]string{
		languageEnglish: "a quarter of the credits",
		languageGerman:  "ein Viertel der Credits",
	}

	value := anthropicSet.selectFor(OptimizeValue)
	quality := anthropicSet.selectFor(OptimizeQuality)
	ratio := value.credits / quality.credits

	if ratio < nearestToAQuarterFrom || ratio >= nearestToAQuarterTo {
		t.Fatalf("anthropic value %s[%s] costs %.1f credits against %.1f for quality %s[%s], a ratio of %.3f that %q no longer describes",
			value.benchmarkID, value.level, value.credits, quality.credits, quality.benchmarkID, quality.level, ratio, claimed[languageEnglish])
	}

	for _, lang := range allLanguages {
		reason := reasonText(anthropicSet.reasonFor(OptimizeValue), lang)
		if !strings.Contains(reason, claimed[lang]) {
			t.Fatalf("anthropic/value in %s states a credit fraction other than %q, which this test no longer guards: %q", lang, claimed[lang], reason)
		}
	}
}

func containsAny(text string, terms []string) bool {
	for _, term := range terms {
		if strings.Contains(text, term) {
			return true
		}
	}
	return false
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
