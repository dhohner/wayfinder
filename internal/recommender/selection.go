package recommender

import (
	"fmt"
	"math"
	"sort"
	"strconv"
)

// Selection tuning. These values are the whole judgment surface of the
// recommendation engine: every recommended model and effort level is computed
// from the bundled benchmark rows by the bands below, so a benchmark refresh is
// re-run rather than re-tuned rule by rule. Review them at each refresh.
const (
	// A candidate is eligible for value if its pass@1 is within valueBandPoints
	// percentage points of the best pass@1 in its set, and for speed if it is
	// within speedBandPoints. Both bands are inclusive.
	valueBandPoints = 5
	speedBandPoints = 10

	// Pass@1 floors, in percentage points, inclusive.
	substantivePassAt1Floor = 60
	anthropicPassAt1Floor   = 60
	routinePassAt1Floor     = 60
	simplePassAt1Floor      = 50

	// Ceilings keep low-cost, low-latency categories low-cost and low-latency.
	// Without them a README typo fix in quality mode would select the top of the
	// leaderboard and no cheap option would remain reachable in any mode.
	// Credits are estimated GitHub Copilot AI credits; steps are the bundled
	// latency proxy. Both ceilings are inclusive.
	routineCreditCap = 100.0
	routineStepCap   = 80
	simpleCreditCap  = 60.0
	simpleStepCap    = 40
)

// Sentinels for a category that declares no ceiling.
const (
	noCreditCap = math.MaxFloat64
	noStepCap   = math.MaxInt32
)

// candidateSet is a filter over the bundled benchmark rows. Its candidates are
// every row meeting the floor, both ceilings, and the model restriction; the
// optimization bands then select one candidate from that set.
type candidateSet struct {
	name         string
	passAt1Floor int
	creditCap    float64
	stepCap      int
	// benchmarkIDs restricts the set to the listed model families. An empty
	// list admits every bundled family.
	benchmarkIDs []string
	reasons      map[Optimization]reasonID
}

var substantiveSet = candidateSet{
	name:         "substantive",
	passAt1Floor: substantivePassAt1Floor,
	creditCap:    noCreditCap,
	stepCap:      noStepCap,
	reasons: map[Optimization]reasonID{
		OptimizeQuality: reasonSubstantiveQuality,
		OptimizeValue:   reasonSubstantiveValue,
		OptimizeCost:    reasonSubstantiveCost,
		OptimizeSpeed:   reasonSubstantiveSpeed,
	},
}

// anthropicSet backs the categories that are a better fit for Claude: visual,
// UI, and UX design work, long-form and creative work, and review of
// GPT-authored code. Restricting it to Opus 5 is what keeps every mode in those
// categories on one Claude model.
var anthropicSet = candidateSet{
	name:         "anthropic",
	passAt1Floor: anthropicPassAt1Floor,
	creditCap:    noCreditCap,
	stepCap:      noStepCap,
	benchmarkIDs: []string{"claude-opus-5"},
	reasons: map[Optimization]reasonID{
		OptimizeQuality: reasonAnthropicQuality,
		OptimizeValue:   reasonAnthropicValue,
		OptimizeCost:    reasonAnthropicCost,
		OptimizeSpeed:   reasonAnthropicSpeed,
	},
}

var routineSet = candidateSet{
	name:         "routine",
	passAt1Floor: routinePassAt1Floor,
	creditCap:    routineCreditCap,
	stepCap:      routineStepCap,
	reasons: map[Optimization]reasonID{
		OptimizeQuality: reasonRoutineQuality,
		OptimizeValue:   reasonRoutineValue,
		OptimizeCost:    reasonRoutineCost,
		OptimizeSpeed:   reasonRoutineSpeed,
	},
}

var simpleSet = candidateSet{
	name:         "simple",
	passAt1Floor: simplePassAt1Floor,
	creditCap:    simpleCreditCap,
	stepCap:      simpleStepCap,
	reasons: map[Optimization]reasonID{
		OptimizeQuality: reasonSimpleQuality,
		OptimizeValue:   reasonSimpleValue,
		OptimizeCost:    reasonSimpleCost,
		OptimizeSpeed:   reasonSimpleSpeed,
	},
}

// candidateRow is one bundled benchmark row in comparable form.
//
// passAt1 is the published point estimate in whole percentage points; the
// published error margin is display-only and never enters a comparison.
// publishedRow is the row's position in the bench_v2.md results table and is
// used only as the final deterministic tie-break.
type candidateRow struct {
	displayModel string
	benchmarkID  string
	level        string
	passAt1      int
	credits      float64
	steps        int
	publishedRow int
}

// orderedCandidates is every bundled benchmark row in published order. The
// bundled table is a map with randomized iteration order, so selection reads
// this slice instead to stay deterministic.
var orderedCandidates = buildOrderedCandidates()

func buildOrderedCandidates() []candidateRow {
	rows := make([]candidateRow, 0, len(bundledBenchmarks))
	for key, entry := range bundledBenchmarks {
		model, ok := displayModelForBenchmarkID(key.model)
		if !ok {
			panic(fmt.Sprintf("recommender: bundled benchmark model %q has no display name", key.model))
		}
		passAt1, err := passAt1Points(entry.passAt1)
		if err != nil {
			panic(fmt.Sprintf("recommender: bundled benchmark %v: %v", key, err))
		}
		rows = append(rows, candidateRow{
			displayModel: model,
			benchmarkID:  key.model,
			level:        key.level,
			passAt1:      passAt1,
			credits:      entry.credits,
			steps:        entry.steps,
			publishedRow: entry.publishedRow,
		})
	}

	sort.Slice(rows, func(i, j int) bool { return rows[i].publishedRow < rows[j].publishedRow })
	for i, row := range rows {
		if row.publishedRow != i+1 {
			panic(fmt.Sprintf("recommender: bundled benchmark rows are not numbered 1..%d in published order; got %d at position %d", len(rows), row.publishedRow, i+1))
		}
	}
	return rows
}

// passAt1Points parses the published pass@1 point estimate, discarding the
// display-only error margin: "73%±3%" parses as 73.
func passAt1Points(value string) (int, error) {
	parsed, err := strconv.Atoi(passAt1NumericText(value))
	if err != nil {
		return 0, fmt.Errorf("cannot parse pass@1 %q: %w", value, err)
	}
	return parsed, nil
}

func (set candidateSet) candidates() []candidateRow {
	candidates := make([]candidateRow, 0, len(orderedCandidates))
	for _, row := range orderedCandidates {
		if row.passAt1 < set.passAt1Floor || row.credits > set.creditCap || row.steps > set.stepCap {
			continue
		}
		if !set.admits(row.benchmarkID) {
			continue
		}
		candidates = append(candidates, row)
	}
	return candidates
}

func (set candidateSet) admits(benchmarkID string) bool {
	if len(set.benchmarkIDs) == 0 {
		return true
	}
	for _, allowed := range set.benchmarkIDs {
		if allowed == benchmarkID {
			return true
		}
	}
	return false
}

// rowKey projects a candidate onto one ordering key; lower always wins, so keys
// that should be maximized are negated.
type rowKey func(candidateRow) float64

func passAt1Descending(row candidateRow) float64 { return -float64(row.passAt1) }
func creditsAscending(row candidateRow) float64  { return row.credits }
func stepsAscending(row candidateRow) float64    { return float64(row.steps) }
func publishedOrder(row candidateRow) float64    { return float64(row.publishedRow) }

// Total orderings, each key breaking exact ties in the previous one. Published
// row order terminates every ordering, so selection is fully deterministic.
var (
	qualityOrder = []rowKey{passAt1Descending, creditsAscending, stepsAscending, publishedOrder}
	creditOrder  = []rowKey{creditsAscending, passAt1Descending, stepsAscending, publishedOrder}
	speedOrder   = []rowKey{stepsAscending, passAt1Descending, creditsAscending, publishedOrder}
)

// selectFor applies the band for one optimization mode to the set.
//
// quality takes the highest pass@1, value the cheapest candidate still within
// the value band of the set's best pass@1, cost the cheapest candidate meeting
// the floor, and speed the fewest steps within the speed band.
func (set candidateSet) selectFor(optimization Optimization) candidateRow {
	candidates := set.candidates()
	if len(candidates) == 0 {
		panic(fmt.Sprintf("recommender: candidate set %q has no candidates; its floor or ceilings exclude every bundled benchmark row", set.name))
	}

	best := candidates[0].passAt1
	for _, row := range candidates[1:] {
		if row.passAt1 > best {
			best = row.passAt1
		}
	}

	var eligible []candidateRow
	var order []rowKey
	switch optimization {
	case OptimizeQuality:
		eligible, order = candidates, qualityOrder
	case OptimizeCost:
		eligible, order = candidates, creditOrder
	case OptimizeSpeed:
		eligible, order = withinBand(candidates, best-speedBandPoints), speedOrder
	default:
		eligible, order = withinBand(candidates, best-valueBandPoints), creditOrder
	}

	selected := eligible[0]
	for _, row := range eligible[1:] {
		if lessBy(order, row, selected) {
			selected = row
		}
	}
	return selected
}

// withinBand keeps candidates at or above a pass@1 threshold. Membership is
// inclusive: a candidate exactly at the threshold stays eligible.
func withinBand(candidates []candidateRow, threshold int) []candidateRow {
	within := make([]candidateRow, 0, len(candidates))
	for _, row := range candidates {
		if row.passAt1 >= threshold {
			within = append(within, row)
		}
	}
	return within
}

func lessBy(order []rowKey, a, b candidateRow) bool {
	for _, key := range order {
		left, right := key(a), key(b)
		if left != right {
			return left < right
		}
	}
	return false
}

// recommendFromSet returns the recommendation the set's band computes for the
// requested optimization mode.
func recommendFromSet(set candidateSet, optimization Optimization) Recommendation {
	selected := set.selectFor(optimization)
	return recommendationFor(selected.displayModel, selected.level, set.reasonFor(optimization))
}

func (set candidateSet) reasonFor(optimization Optimization) reasonID {
	if reason, ok := set.reasons[optimization]; ok {
		return reason
	}
	return set.reasons[OptimizeValue]
}

// recommendationFor labels the recommendation with the terminology of the
// selected model's provider. A single category can select either family across
// modes, so terminology follows the selection rather than the rule.
func recommendationFor(model, level string, reason reasonID) Recommendation {
	text := reasonText(reason)
	switch providerForModel(model) {
	case providerGPT:
		return gptRecommendation(model, level, text)
	case providerAnthropic:
		return anthropicRecommendation(model, level, text)
	default:
		panic(fmt.Sprintf("recommender: selected model %q has no provider family", model))
	}
}
