package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/dhohner/wayfinder/internal/recommender"
)

type stubRecommender struct {
	task         string
	optimization recommender.Optimization
}

func (s *stubRecommender) RecommendWithOptimization(task string, optimization recommender.Optimization) recommender.Recommendation {
	s.task = task
	s.optimization = optimization
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

func TestRunRejectsInvalidArguments(t *testing.T) {
	cases := [][]string{
		{},
		{"--optimize"},
		{"--optimize", ""},
		{"--optimize=cheap"},
		{"--explain=false", "task"},
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
