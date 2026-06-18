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
	recommendation recommender.Recommendation
}

func (s *stubRecommender) RecommendWithOptimization(task string, optimization recommender.Optimization) recommender.Recommendation {
	s.task = task
	s.optimization = optimization
	if s.recommendation != (recommender.Recommendation{}) {
		return s.recommendation
	}
	return recommender.Recommendation{Model: recommender.GPT55, ReasoningSetting: "GPT reasoning level: medium", Reason: "test recommendation"}
}

func TestRunParsesArgumentsAndWritesRecommendation(t *testing.T) {
	var stdout, stderr bytes.Buffer
	rec := &stubRecommender{}

	exitCode := Run([]string{"--optimize=quality", "implement", "an", "API"}, &stdout, &stderr, rec)

	if exitCode != 0 {
		t.Fatalf("expected success exit code, got %d: %s", exitCode, stderr.String())
	}
	if rec.optimization != recommender.OptimizeQuality || rec.task != "implement an API" {
		t.Fatalf("unexpected recommendation input: optimization=%q task=%q", rec.optimization, rec.task)
	}
	assertContainsAll(t, stdout.String(), "Model: GPT 5.5", "Reasoning: GPT reasoning level: medium", "Reason: test recommendation")
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
	assertContainsAll(t, stdout.String(), "Pass@1 48%±3%", "AIC 57.0", "AIC factor 1.00", "Tradeoff:")
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
	if got := doc["model"]; got != "gpt-5.5" {
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
	if benchmark["pass_at_1"] != 0.48 || benchmark["aic"] != 57.0 || benchmark["aic_factor"] != 1.0 {
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
	rec := &stubRecommender{recommendation: recommender.Recommendation{Model: recommender.GPT55, ReasoningSetting: "GPT reasoning level: low", Reason: "simple task"}}

	exitCode := Run([]string{"--json", "fix", "a", "typo"}, &stdout, &stderr, rec)

	if exitCode != 0 {
		t.Fatalf("expected success exit code, got %d: %s", exitCode, stderr.String())
	}
	var doc map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &doc); err != nil {
		t.Fatalf("expected valid JSON, got %q: %v", stdout.String(), err)
	}
	if doc["model"] != "gpt-5.5" || doc["reasoning"] != "low" || doc["profile"] != "value" {
		t.Fatalf("unexpected normalized fields: %v", doc)
	}
	if _, ok := doc["benchmark"]; ok {
		t.Fatalf("did not expect benchmark for missing exact match: %v", doc)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected no stderr output, got %q", stderr.String())
	}
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
	assertContainsAll(t, stdout.String(), "Model: GPT 5.5", "Reasoning: GPT reasoning level: high")
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
