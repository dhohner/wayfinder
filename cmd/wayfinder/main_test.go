package main

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/dhohner/wayfinder/internal/recommender"
)

func TestParseArgsAcceptsPreferForms(t *testing.T) {
	pref, task, ok := parseArgs([]string{"--prefer", "quality", "implement", "an", "API"})
	if !ok || pref != recommender.PreferQuality || task != "implement an API" {
		t.Fatalf("unexpected parsed args: pref=%q task=%q ok=%v", pref, task, ok)
	}

	pref, task, ok = parseArgs([]string{"--prefer=speed", "summarize", "notes"})
	if !ok || pref != recommender.PreferSpeed || task != "summarize notes" {
		t.Fatalf("unexpected parsed args: pref=%q task=%q ok=%v", pref, task, ok)
	}
}

func TestParseArgsRejectsInvalidPreferenceFlags(t *testing.T) {
	cases := [][]string{
		{"--prefer"},
		{"--prefer", ""},
		{"--prefer=cheap"},
		{"--unknown", "task"},
	}

	for _, args := range cases {
		if _, _, ok := parseArgs(args); ok {
			t.Fatalf("expected args to be rejected: %#v", args)
		}
	}
}

func TestCLIOutputReflectsTaskComplexity(t *testing.T) {
	cases := []struct {
		name string
		task string
		want []string
	}{
		{
			name: "simple",
			task: "fix a typo in a README",
			want: []string{"Model: GPT 5.5", "Reasoning: GPT reasoning level: low"},
		},
		{
			name: "complex",
			task: "debug an intermittent distributed race condition in production",
			want: []string{"Model: GPT 5.5", "Reasoning: GPT reasoning level: high"},
		},
		{
			name: "ambiguous",
			task: "help me with this task",
			want: []string{"Model: GPT 5.5", "Reasoning: GPT reasoning level: medium", "Conservative default"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := exec.Command("go", "run", ".", tc.task)
			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("CLI failed: %v\n%s", err, out)
			}

			output := string(out)
			for _, want := range tc.want {
				if !strings.Contains(output, want) {
					t.Fatalf("expected CLI output to contain %q, got:\n%s", want, output)
				}
			}
			if strings.Contains(output, "?") || strings.Contains(strings.ToLower(output), "clarify") {
				t.Fatalf("CLI should recommend instead of asking a clarification question, got:\n%s", output)
			}
		})
	}
}
