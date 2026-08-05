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

	// Where an ordering compares credits, the eligible candidates priced at or
	// below the cheapest eligible candidate's credits times creditTieBandRatio
	// form one tie cohort and are separated by pass rate instead of by price.
	//
	// The bundled credit figures are an estimate carrying roughly a quarter error
	// bar (see benchmarkEntry in format.go), so 10% claims well less tolerance
	// than the data's own accuracy, while being wide enough to absorb the
	// sub-percent credit differences an exact comparison would otherwise rank
	// ahead of every pass rate difference. Because the cohort is anchored on the
	// cheapest eligible candidate, 10% is also the most the rule can ever
	// overpay against the cheapest available price.
	creditTieBandRatio = 1.10

	// Pass@1 floors, in percentage points, inclusive.
	substantivePassAt1Floor = 60
	anthropicPassAt1Floor   = 60
	routinePassAt1Floor     = 60
	simplePassAt1Floor      = 50

	// A category declares a cost budget in estimated GitHub Copilot AI credits
	// and, optionally, a latency budget in benchmark steps. The two bound
	// different commitments and neither substitutes for the other: the credit
	// column already prices the tokens every step consumes, so a step ceiling is a
	// latency-proxy commitment rather than a second way to spell a cost ceiling.
	// Both budgets are inclusive.
	//
	// Which modes a budget constrains is part of what the budget means. The cost
	// budget, the pass rate floor, and the family restriction constrain every
	// mode. The latency budget never constrains cost mode, always constrains speed
	// mode, and constrains quality and value unless holding it would make either
	// of them answer with a dominated row. Without a cost budget a README typo fix
	// in quality mode would select the top of the leaderboard and no cheap option
	// would remain reachable in any mode.
	//
	// The cost budgets are denominated in whole-run credits, which include input
	// tokens. They were 100 and 60 while the bundled figures counted output tokens
	// only; carrying those values across the 2026-08-02 basis change would have
	// emptied both sets. The replacements preserve the exact membership the old
	// ceilings produced: routineCostBudget must admit claude-opus-5[medium] at 352
	// so the documented 69% quality tie-break still has both sides, and exclude
	// claude-fable-5[low] at 371.
	//
	// routineLatencyBudget is what keeps routine quality, value, and speed off
	// gpt-5.6-terra[max] at 76 steps, which buys one point of pass@1 over
	// gpt-5.6-sol[high] at 37. It was tightened from 80 to 60 after the 2026-08-02
	// Copilot price cut brought that row inside the cost budget. Routine work is
	// latency-sensitive by definition, so its latency-proxy commitment has to be
	// stated in steps and must not depend on a price staying where it is.
	//
	// simple deliberately declares no latency budget. Its ceiling of 40 steps was
	// removed because it was verified to change no selection in any of the four
	// modes once the latency budget stopped constraining cost, not because simple
	// work is indifferent to latency: simple now carries no latency-proxy
	// commitment at all, so a refresh introducing a cheap, high-scoring,
	// very long-running row would give simple tasks an unbounded default.
	// Restoring the commitment is a one-constant change.
	//
	// At each refresh, re-run the suite rather than re-tuning rule by rule, and
	// treat the dominance invariant as part of the procedure: no band except speed
	// may return a row that another row meeting the same floor, cost budget, and
	// family restriction beats on both pass rate and credits. speed is exempt
	// because steps are its objective.
	routineCostBudget    = 360.0
	routineLatencyBudget = 60
	simpleCostBudget     = 200.0
)

// Sentinels for a category that declares no budget of the given kind.
const (
	noCostBudget    = math.MaxFloat64
	noLatencyBudget = math.MaxInt32
)

// candidateSet is a filter over the bundled benchmark rows. Its admissible rows
// are every row meeting the floor, the cost budget, and the model restriction;
// the latency budget then narrows that set for the modes it constrains, and the
// optimization bands select one candidate from what remains.
type candidateSet struct {
	name         string
	passAt1Floor int
	// costBudget is the inclusive ceiling on estimated Copilot credits and
	// constrains every optimization mode.
	costBudget float64
	// latencyBudget is the inclusive ceiling on benchmark steps. It never
	// constrains cost mode, always constrains speed mode, and constrains quality
	// and value while neither of them would answer, from inside the budget, with a
	// row an excluded row dominates.
	latencyBudget int
	// benchmarkIDs restricts the set to the listed model families. An empty
	// list admits every bundled family.
	benchmarkIDs []string
	reasons      map[Optimization]reasonID
}

var substantiveSet = candidateSet{
	name:         "substantive",
	passAt1Floor: substantivePassAt1Floor,
	costBudget:   noCostBudget,
	// substantive work declares no latency budget: it is the category where a
	// long run is an acceptable price for the top of the leaderboard.
	latencyBudget: noLatencyBudget,
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
	name:          "anthropic",
	passAt1Floor:  anthropicPassAt1Floor,
	costBudget:    noCostBudget,
	latencyBudget: noLatencyBudget,
	benchmarkIDs:  []string{"claude-opus-5"},
	reasons: map[Optimization]reasonID{
		OptimizeQuality: reasonAnthropicQuality,
		OptimizeValue:   reasonAnthropicValue,
		OptimizeCost:    reasonAnthropicCost,
		OptimizeSpeed:   reasonAnthropicSpeed,
	},
}

var routineSet = candidateSet{
	name:          "routine",
	passAt1Floor:  routinePassAt1Floor,
	costBudget:    routineCostBudget,
	latencyBudget: routineLatencyBudget,
	reasons: map[Optimization]reasonID{
		OptimizeQuality: reasonRoutineQuality,
		OptimizeValue:   reasonRoutineValue,
		OptimizeCost:    reasonRoutineCost,
		OptimizeSpeed:   reasonRoutineSpeed,
	},
}

// simpleSet is bounded by its pass rate floor and cost budget alone; see the
// tuning block for why its latency budget was removed and what restoring it
// would cost.
var simpleSet = candidateSet{
	name:          "simple",
	passAt1Floor:  simplePassAt1Floor,
	costBudget:    simpleCostBudget,
	latencyBudget: noLatencyBudget,
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
		if !validCreditEstimate(entry.credits) {
			panic(fmt.Sprintf("recommender: bundled benchmark %v has invalid credit estimate %v; credits must be finite and non-negative", key, entry.credits))
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

// admissible returns every bundled row meeting the set's pass rate floor, cost
// budget, and family restriction, in published order. The latency budget is
// deliberately not applied here: which modes it constrains is decided in
// candidatesFor, and the dominance invariant is stated over this set so that it
// cannot be satisfied by whatever the latency budget happened to remove.
func (set candidateSet) admissible() []candidateRow {
	admissible := make([]candidateRow, 0, len(orderedCandidates))
	for _, row := range orderedCandidates {
		if row.passAt1 < set.passAt1Floor || row.credits > set.costBudget {
			continue
		}
		if !set.admits(row.benchmarkID) {
			continue
		}
		admissible = append(admissible, row)
	}
	return admissible
}

// withinLatencyBudget keeps the rows at or below the set's latency budget,
// preserving published order.
func (set candidateSet) withinLatencyBudget(rows []candidateRow) []candidateRow {
	within := make([]candidateRow, 0, len(rows))
	for _, row := range rows {
		if row.steps <= set.latencyBudget {
			within = append(within, row)
		}
	}
	return within
}

// candidatesFor returns the rows one optimization mode selects from.
//
// cost is never narrowed by the latency budget: choosing cost mode says price
// outranks the latency proxy, so a latency filter ahead of the cost band would
// contradict the instruction. speed is always narrowed by it, because steps are
// the objective that mode optimizes. quality and value share all in-budget rows,
// releasing the budget when holding it would make either of them answer with a
// dominated row.
func (set candidateSet) candidatesFor(optimization Optimization) []candidateRow {
	admissible := set.admissible()
	if optimization == OptimizeCost || set.latencyBudget == noLatencyBudget {
		return admissible
	}

	within := set.withinLatencyBudget(admissible)
	if optimization == OptimizeSpeed {
		return within
	}
	if set.releasesLatencyBudget(admissible, within) {
		return admissible
	}
	return within
}

// releasesLatencyBudget reports whether the latency budget has to give way for
// quality and value: it does when the row either of them would select from
// within reaches is strictly beaten on both pass rate and credits by a row the
// budget excluded. Testing the selected rows rather than every in-budget row is
// what enforces the dominance invariant, since a cheap defensible row that no
// band ever answers with cannot keep a dominated answer in force.
//
// Both modes are tested together and released together. They are documented to
// share one candidate set, and releasing only the mode whose own answer is
// dominated could hand value a row scoring above quality's.
//
// The empty in-budget set releases: there is no answer to defend, and quality
// and value stay answerable even when the latency budget excludes every
// admissible row.
func (set candidateSet) releasesLatencyBudget(admissible, within []candidateRow) bool {
	if len(within) == 0 {
		return true
	}
	for _, optimization := range []Optimization{OptimizeQuality, OptimizeValue} {
		if set.dominatedByExcluded(admissible, selectFrom(within, optimization)) {
			return true
		}
	}
	return false
}

// dominatedByExcluded reports whether any admissible row the latency budget
// excluded beats row on both pass rate and credits.
func (set candidateSet) dominatedByExcluded(admissible []candidateRow, row candidateRow) bool {
	for _, excluded := range admissible {
		if excluded.steps > set.latencyBudget && excluded.beats(row) {
			return true
		}
	}
	return false
}

// beats reports whether row is strictly better than other on both pass rate and
// credits. A tie on either axis is not a win.
//
// Credits are compared exactly here, without the credit tie cohort's tolerance.
// This is a dominance test and never an ordering, so it is never asked to rank
// two candidates the credit estimate cannot tell apart; it only asks whether
// some row is better on both axes at once.
func (row candidateRow) beats(other candidateRow) bool {
	return row.passAt1 > other.passAt1 && row.credits < other.credits
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
	qualityFirstOrder = []rowKey{passAt1Descending, creditsAscending, stepsAscending, publishedOrder}
	speedOrder        = []rowKey{stepsAscending, passAt1Descending, creditsAscending, publishedOrder}
)

// selectFor applies the band for one optimization mode to the candidates that
// mode selects from.
func (set candidateSet) selectFor(optimization Optimization) candidateRow {
	candidates := set.candidatesFor(optimization)
	if len(candidates) == 0 {
		if len(set.admissible()) == 0 {
			panic(fmt.Sprintf("recommender: candidate set %q has no candidates; its pass rate floor, cost budget, or model restriction excludes every bundled benchmark row", set.name))
		}
		panic(fmt.Sprintf("recommender: candidate set %q has no candidates for %q; its latency budget of %d steps excludes every row meeting its pass rate floor and cost budget", set.name, optimization, set.latencyBudget))
	}
	return selectFrom(candidates, optimization)
}

// selectFrom applies one optimization mode's band to a non-empty candidate
// slice, without reference to which filters produced it. The latency release
// asks it what quality and value would answer from inside the budget, so it must
// stay free of the filtering decisions in candidatesFor.
//
// quality takes the highest pass@1 and speed the fewest steps within the speed
// band. value and cost are the two credit-ordered bands: value first narrows to
// the value band of the set's best pass@1 and cost keeps every candidate, and
// both then answer with the highest pass@1 inside the credit tie cohort around
// their cheapest eligible candidate.
func selectFrom(candidates []candidateRow, optimization Optimization) candidateRow {
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
		eligible, order = candidates, qualityFirstOrder
	case OptimizeCost:
		eligible, order = withinCreditTieCohort(candidates), qualityFirstOrder
	case OptimizeSpeed:
		eligible, order = withinBand(candidates, best-speedBandPoints), speedOrder
	default:
		eligible, order = withinCreditTieCohort(withinBand(candidates, best-valueBandPoints)), qualityFirstOrder
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

// withinCreditTieCohort keeps the candidates whose credits are close enough to
// the cheapest candidate's that the credit estimate cannot tell them apart.
// Membership is inclusive: a candidate priced at exactly the cheapest candidate's
// credits times creditTieBandRatio stays in the cohort, and one marginally above
// that is dropped. Published order is preserved, and a single candidate always
// forms a cohort of one, which orders exactly as the plain cheapest-first
// comparison did.
//
// The cohort is anchored on the single cheapest candidate and never computed
// pairwise. Approximate equality is not transitive and so cannot define a sort:
// a pairwise epsilon would make the answer depend on the order candidates were
// compared in. One anchor gives one well-defined cohort and mirrors withinBand,
// which anchors the pass rate bands on the best candidate in the set.
//
// The anchor scales the threshold, so a zero-credit anchor admits only the
// candidates that are themselves free. No bundled row is free; the cheapest is
// 0.8 credits.
func withinCreditTieCohort(candidates []candidateRow) []candidateRow {
	if len(candidates) == 0 {
		return candidates
	}

	cheapest := candidates[0].credits
	for _, row := range candidates {
		if !validCreditEstimate(row.credits) {
			panic(fmt.Sprintf("recommender: candidate %s[%s] has invalid credit estimate %v; credits must be finite and non-negative", row.benchmarkID, row.level, row.credits))
		}
		if row.credits < cheapest {
			cheapest = row.credits
		}
	}

	threshold := cheapest * creditTieBandRatio
	cohort := make([]candidateRow, 0, len(candidates))
	for _, row := range candidates {
		if row.credits <= threshold {
			cohort = append(cohort, row)
		}
	}
	return cohort
}

func validCreditEstimate(credits float64) bool {
	return credits >= 0 && !math.IsNaN(credits) && !math.IsInf(credits, 0)
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

// selection is the structural answer to a task: the benchmark row a band chose
// and the identifier of the reason for choosing it. It holds no prose and no
// language, so everything that produces a selection is language-independent by
// construction.
type selection struct {
	row    candidateRow
	reason reasonID
}

// selectionFor returns what the set's band computes for the requested
// optimization mode.
func (set candidateSet) selectionFor(optimization Optimization) selection {
	return selection{row: set.selectFor(optimization), reason: set.reasonFor(optimization)}
}

func (set candidateSet) reasonFor(optimization Optimization) reasonID {
	if reason, ok := set.reasons[optimization]; ok {
		return reason
	}
	return set.reasons[OptimizeValue]
}

// localize writes a selection out as a recommendation in lang. It is the single
// place the detected language enters the pipeline, which is what keeps detection
// one-way: nothing upstream of it can read the language back.
//
// The effort level is stored as the provider's bare identifier; which
// terminology labels it is decided when the recommendation is formatted, since
// a single category can select either provider family across modes.
func (s selection) localize(lang language) Recommendation {
	if providerForModel(s.row.displayModel) == providerUnknown {
		panic(fmt.Sprintf("recommender: selected model %q has no provider family", s.row.displayModel))
	}
	return Recommendation{
		Model:            s.row.displayModel,
		ReasoningSetting: s.row.level,
		Reason:           reasonText(s.reason, lang),
		Language:         lang,
	}
}
