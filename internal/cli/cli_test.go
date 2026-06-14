package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/dhohner/wayfinder/internal/recommender"
)

type stubRecommender struct {
	task       string
	preference recommender.Preference
}

func (s *stubRecommender) RecommendWithPreference(task string, preference recommender.Preference) recommender.Recommendation {
	s.task = task
	s.preference = preference
	return recommender.Recommendation{Model: recommender.GPT55, ReasoningSetting: "GPT reasoning level: medium", Reason: "test recommendation"}
}

func TestRunParsesArgumentsAndWritesRecommendation(t *testing.T) {
	var stdout, stderr bytes.Buffer
	rec := &stubRecommender{}

	exitCode := Run([]string{"--prefer=quality", "implement", "an", "API"}, &stdout, &stderr, rec)

	if exitCode != 0 {
		t.Fatalf("expected success exit code, got %d: %s", exitCode, stderr.String())
	}
	if rec.preference != recommender.PreferQuality || rec.task != "implement an API" {
		t.Fatalf("unexpected recommendation input: preference=%q task=%q", rec.preference, rec.task)
	}
	assertContainsAll(t, stdout.String(), "Model: GPT 5.5", "Reasoning: GPT reasoning level: medium", "Reason: test recommendation")
	if stderr.Len() != 0 {
		t.Fatalf("expected no stderr output, got %q", stderr.String())
	}
}

func TestRunRejectsInvalidArguments(t *testing.T) {
	cases := [][]string{
		{},
		{"--prefer"},
		{"--prefer", ""},
		{"--prefer=cheap"},
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
