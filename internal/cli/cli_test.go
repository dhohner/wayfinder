package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/dhohner/wayfinder/internal/recommender"
)

type stubRecommender struct {
	task           string
	optimization   recommender.Optimization
	against        recommender.AgainstFamily
	recommendation recommender.Recommendation
}

func (s *stubRecommender) RecommendWithOptimizationAgainst(task string, optimization recommender.Optimization, against recommender.AgainstFamily) recommender.Recommendation {
	s.task = task
	s.optimization = optimization
	s.against = against
	if s.recommendation != (recommender.Recommendation{}) {
		return s.recommendation
	}
	return recommender.Recommendation{Model: recommender.GPT56Sol, ReasoningSetting: "medium", Reason: "test recommendation"}
}

func TestRunParsesArgumentsAndWritesRecommendation(t *testing.T) {
	var stdout, stderr bytes.Buffer
	rec := &stubRecommender{}

	exitCode := Run([]string{"--optimize=quality", "--against", "gpt", "implement", "an", "API"}, &stdout, &stderr, rec)

	if exitCode != 0 {
		t.Fatalf("expected success exit code, got %d: %s", exitCode, stderr.String())
	}
	if rec.optimization != recommender.OptimizeQuality || rec.against != recommender.AgainstGPT || rec.task != "implement an API" {
		t.Fatalf("unexpected recommendation input: optimization=%q against=%q task=%q", rec.optimization, rec.against, rec.task)
	}
	assertContainsAll(t, stdout.String(), "Model: GPT 5.6 Sol", "Reasoning: GPT reasoning level: medium", "Reason: test recommendation")
	assertNotContainsAny(t, stdout.String(), "Pass@1", "AIC", "AIC factor", "Benchmark:")
	if stderr.Len() != 0 {
		t.Fatalf("expected no stderr output, got %q", stderr.String())
	}
}

func TestRunExplainAddsBenchmarkRationale(t *testing.T) {
	var stdout, stderr bytes.Buffer
	rec := &stubRecommender{}

	exitCode := Run([]string{"--explain", "--optimize", "cost", "implement", "an", "API"}, &stdout, &stderr, rec)

	if exitCode != 0 {
		t.Fatalf("expected success exit code, got %d: %s", exitCode, stderr.String())
	}
	if rec.optimization != recommender.OptimizeCost || rec.task != "implement an API" {
		t.Fatalf("unexpected recommendation input: optimization=%q task=%q", rec.optimization, rec.task)
	}
	assertContainsAll(t, stdout.String(), "Pass@1 61%±2%", "average cost 1.86", "Tradeoff:")
	if stderr.Len() != 0 {
		t.Fatalf("expected no stderr output, got %q", stderr.String())
	}
}

func TestRunJSONWritesOneNormalizedDocument(t *testing.T) {
	var stdout, stderr bytes.Buffer
	rec := &stubRecommender{}

	exitCode := Run([]string{"--json", "--optimize=cost", "implement", "an", "API"}, &stdout, &stderr, rec)

	if exitCode != 0 {
		t.Fatalf("expected success exit code, got %d: %s", exitCode, stderr.String())
	}
	if rec.optimization != recommender.OptimizeCost || rec.task != "implement an API" {
		t.Fatalf("unexpected recommendation input: optimization=%q task=%q", rec.optimization, rec.task)
	}
	var doc map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &doc); err != nil {
		t.Fatalf("expected valid JSON, got %q: %v", stdout.String(), err)
	}
	if got := doc["model"]; got != "gpt-5.6-sol" {
		t.Fatalf("expected normalized model, got %v in %v", got, doc)
	}
	if got := doc["reasoning"]; got != "medium" {
		t.Fatalf("expected normalized reasoning, got %v in %v", got, doc)
	}
	if got := doc["profile"]; got != "cost" {
		t.Fatalf("expected profile cost, got %v in %v", got, doc)
	}
	if got := doc["reason"]; got != "test recommendation" {
		t.Fatalf("expected reason, got %v in %v", got, doc)
	}
	benchmark, ok := doc["benchmark"].(map[string]any)
	if !ok {
		t.Fatalf("expected benchmark object in %v", doc)
	}
	if benchmark["pass_at_1"] != 0.61 || benchmark["average_cost"] != 1.86 {
		t.Fatalf("unexpected benchmark values: %v", benchmark)
	}
	if _, ok := benchmark["tradeoff"]; ok {
		t.Fatalf("did not expect tradeoff without --explain: %v", benchmark)
	}
	assertNotContainsAny(t, stdout.String(), "Model:", "Reasoning:", "Benchmark:", "Pass@1")
	if stderr.Len() != 0 {
		t.Fatalf("expected no stderr output, got %q", stderr.String())
	}
}

func TestRunJSONExplainStaysJSONAndIncludesExplanationData(t *testing.T) {
	var stdout, stderr bytes.Buffer
	rec := &stubRecommender{}

	exitCode := Run([]string{"--json", "--explain", "implement", "an", "API"}, &stdout, &stderr, rec)

	if exitCode != 0 {
		t.Fatalf("expected success exit code, got %d: %s", exitCode, stderr.String())
	}
	var doc struct {
		Benchmark struct {
			Tradeoff string `json:"tradeoff"`
		} `json:"benchmark"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &doc); err != nil {
		t.Fatalf("expected valid JSON, got %q: %v", stdout.String(), err)
	}
	if doc.Benchmark.Tradeoff == "" {
		t.Fatalf("expected explanation tradeoff in JSON document: %q", stdout.String())
	}
	assertNotContainsAny(t, stdout.String(), "Model:", "Reasoning:", "Benchmark:")
	if stderr.Len() != 0 {
		t.Fatalf("expected no stderr output, got %q", stderr.String())
	}
}

func TestRunJSONOmitsBenchmarkWhenNoExactMatch(t *testing.T) {
	var stdout, stderr bytes.Buffer
	rec := &stubRecommender{recommendation: recommender.Recommendation{Model: recommender.GPT54, ReasoningSetting: "high", Reason: "unsupported level"}}

	exitCode := Run([]string{"--json", "fix", "a", "typo"}, &stdout, &stderr, rec)

	if exitCode != 0 {
		t.Fatalf("expected success exit code, got %d: %s", exitCode, stderr.String())
	}
	var doc map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &doc); err != nil {
		t.Fatalf("expected valid JSON, got %q: %v", stdout.String(), err)
	}
	if doc["model"] != "gpt-5.4" || doc["reasoning"] != "high" || doc["profile"] != "value" {
		t.Fatalf("unexpected normalized fields: %v", doc)
	}
	if _, ok := doc["benchmark"]; ok {
		t.Fatalf("did not expect benchmark for missing exact match: %v", doc)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected no stderr output, got %q", stderr.String())
	}
}

func TestRunReportsEstimatedCreditsInExplanationAndJSON(t *testing.T) {
	var explainOut, explainErr bytes.Buffer

	if exitCode := Run([]string{"--explain", "implement", "an", "API"}, &explainOut, &explainErr, &stubRecommender{}); exitCode != 0 {
		t.Fatalf("expected success exit code, got %d: %s", exitCode, explainErr.String())
	}
	assertContainsAll(t, explainOut.String(), "average cost 1.86", "Estimated Copilot AI credits: 164.4 (input and output tokens, estimate).", "Tradeoff:")
	assertNotContainsAny(t, explainOut.String(), "$")

	var jsonOut, jsonErr bytes.Buffer

	if exitCode := Run([]string{"--json", "implement", "an", "API"}, &jsonOut, &jsonErr, &stubRecommender{}); exitCode != 0 {
		t.Fatalf("expected success exit code, got %d: %s", exitCode, jsonErr.String())
	}
	var doc struct {
		Benchmark map[string]any `json:"benchmark"`
	}
	if err := json.Unmarshal(jsonOut.Bytes(), &doc); err != nil {
		t.Fatalf("expected valid JSON, got %q: %v", jsonOut.String(), err)
	}
	if doc.Benchmark["credits_estimate"] != 164.4 {
		t.Fatalf("expected credits_estimate 164.4 without --explain, got %v in %v", doc.Benchmark["credits_estimate"], doc.Benchmark)
	}
	if _, ok := doc.Benchmark["tradeoff"]; ok {
		t.Fatalf("did not expect tradeoff without --explain: %v", doc.Benchmark)
	}

	var defaultOut, defaultErr bytes.Buffer

	if exitCode := Run([]string{"implement", "an", "API"}, &defaultOut, &defaultErr, &stubRecommender{}); exitCode != 0 {
		t.Fatalf("expected success exit code, got %d: %s", exitCode, defaultErr.String())
	}
	assertNotContainsAny(t, defaultOut.String(), "Estimated Copilot AI credits", "credits_estimate", "$")
}

func TestRunJSONFormatErrorReturnsNonZeroWithoutPartialDocument(t *testing.T) {
	var stdout, stderr bytes.Buffer
	rec := &stubRecommender{recommendation: recommender.Recommendation{Model: "Mystery", ReasoningSetting: "unknown", Reason: "test"}}

	exitCode := Run([]string{"--json", "task"}, &stdout, &stderr, rec)

	if exitCode == 0 {
		t.Fatalf("expected non-zero exit code")
	}
	if stdout.Len() != 0 {
		t.Fatalf("expected no partial JSON on stdout, got %q", stdout.String())
	}
	assertContainsAll(t, stderr.String(), "unsupported model")
}

func TestRunRejectsInvalidArguments(t *testing.T) {
	cases := [][]string{
		{},
		{"--optimize"},
		{"--optimize", ""},
		{"--optimize=cheap"},
		{"--against"},
		{"--against", "gemini", "task"},
		{"--against="},
		{"--against=gemini", "task"},
		{"--explain=false", "task"},
		{"--json=false", "task"},
		{"--explain", "--optimize"},
		{"--prefer=quality", "task"},
		{"--unknown", "task"},
	}

	for _, args := range cases {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			var stdout, stderr bytes.Buffer

			exitCode := Run(args, &stdout, &stderr, &stubRecommender{})

			if exitCode != 2 {
				t.Fatalf("expected usage exit code, got %d", exitCode)
			}
			if stdout.Len() != 0 {
				t.Fatalf("expected no stdout output, got %q", stdout.String())
			}
			assertContainsAll(t, stderr.String(), "Usage: wayfinder")
		})
	}
}

func TestRunUsesDefaultRecommender(t *testing.T) {
	var stdout, stderr bytes.Buffer

	exitCode := Run([]string{"debug an intermittent distributed race condition in production"}, &stdout, &stderr, nil)

	if exitCode != 0 {
		t.Fatalf("expected success exit code, got %d: %s", exitCode, stderr.String())
	}
	assertContainsAll(t, stdout.String(), "Model: GPT 5.6 Sol", "Reasoning: GPT reasoning level: high")
}

func TestRunAgainstUsesDefaultRecommenderForCodeReview(t *testing.T) {
	var stdout, stderr bytes.Buffer

	exitCode := Run([]string{"--against=gpt", "review this pull request for bugs"}, &stdout, &stderr, nil)

	if exitCode != 0 {
		t.Fatalf("expected success exit code, got %d: %s", exitCode, stderr.String())
	}
	assertContainsAll(t, stdout.String(), "Model: Claude Opus 5", "Reasoning: Anthropic Effort Level: medium")
}

// TestRunAnswersAGermanTaskInGermanWithoutChangingTheDocument exercises the
// real recommender end to end: a German task must read in German while the
// machine-readable document keeps its English field names and identifiers, and
// the command must still succeed with its benchmark evidence intact.
func TestRunAnswersAGermanTaskInGermanWithoutChangingTheDocument(t *testing.T) {
	const germanTask = "implementiere einen kleinen Go-API-Endpunkt"

	var textOut, textErr bytes.Buffer
	if exitCode := Run([]string{"--explain", germanTask}, &textOut, &textErr, nil); exitCode != 0 {
		t.Fatalf("expected success exit code, got %s", textErr.String())
	}
	assertContainsAll(t, textOut.String(), "Modell: ", "Reasoning: ", "Begründung: ", "durchschnittliche Kosten", "Geschätzte Copilot-AI-Credits: ", "(Eingabe- und Ausgabe-Tokens, Schätzwert).", "Abwägung: ")
	assertNotContainsAny(t, textOut.String(), "Model: ", "Reason: ", "average cost", "Tradeoff: ", "$")

	var jsonOut, jsonErr bytes.Buffer
	if exitCode := Run([]string{"--json", "--explain", germanTask}, &jsonOut, &jsonErr, nil); exitCode != 0 {
		t.Fatalf("expected success exit code, got %s", jsonErr.String())
	}
	var doc struct {
		Model     string `json:"model"`
		Reasoning string `json:"reasoning"`
		Profile   string `json:"profile"`
		Reason    string `json:"reason"`
		Benchmark *struct {
			PassAt1  float64 `json:"pass_at_1"`
			Tradeoff string  `json:"tradeoff"`
		} `json:"benchmark"`
	}
	if err := json.Unmarshal(jsonOut.Bytes(), &doc); err != nil {
		t.Fatalf("expected valid JSON, got %q: %v", jsonOut.String(), err)
	}
	if doc.Model != "gpt-5.6-luna" || doc.Reasoning != "max" || doc.Profile != "value" {
		t.Fatalf("expected unchanged English identifiers, got %+v", doc)
	}
	if doc.Benchmark == nil || doc.Benchmark.PassAt1 != 0.67 {
		t.Fatalf("expected the benchmark block to survive localization: %q", jsonOut.String())
	}
	if !strings.Contains(doc.Reason, "Trefferquote") || !strings.Contains(doc.Benchmark.Tradeoff, "Einstellung") {
		t.Fatalf("expected German reason and tradeoff text, got %q and %q", doc.Reason, doc.Benchmark.Tradeoff)
	}
	assertNotContainsAny(t, doc.Reason+" "+doc.Benchmark.Tradeoff, "pass rate", "Lowest-cost")
}

func TestRunWithNilWritersDiscardsOutput(t *testing.T) {
	exitCode := Run([]string{"help me with this task"}, nil, nil, nil)

	if exitCode != 0 {
		t.Fatalf("expected success exit code, got %d", exitCode)
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
