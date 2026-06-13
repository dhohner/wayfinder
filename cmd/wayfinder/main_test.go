package main

import (
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
