package recommender

import "strings"

type taskTraits struct {
	simple           bool
	coding           bool
	codeReview       bool
	largeContext     bool
	anthropicFit     bool
	visualDesign     bool
	nuancedRoutine   bool
	deepReasoning    bool
	highRisk         bool
	correctnessHeavy bool
	against          AgainstFamily
}

var simpleSignals = []string{
	"summarize", "summary", "rewrite", "proofread", "copy edit", "format", "extract", "release notes", "short email", "typo", "spelling", "grammar", "readme", "changelog", "rename", "one-line", "small", "minor", "quick", "lint", "comment", "documentation",
}

var codingSignals = []string{
	"code", "coding", "implement", "refactor", "debug", "test", "typescript", "javascript", "golang", "go", "python", "rust", "java", "sql", "api", "sdk", "cli", "module", "bug", "endpoint", "function", "class", "component", "frontend", "backend", "database", "schema", "query", "build", "deploy", "parser", "parse",
}

var largeContextSignals = []string{
	"long", "many files", "multiple files", "whole repo", "entire repo", "repository", "repo", "codebase", "monorepo", "migration", "cross-service", "multi-service", "integration", "legacy", "10-page", "10 page", "thousands of lines",
}

var codeReviewSignals = []string{
	"code review", "review code", "review the code", "review this code", "review my code", "adversarial code review", "pull request review", "review pull request", "review a pull request", "review this pull request", "review the pull request", "review pr", "review a pr", "review this pr", "review the pr", "review diff", "review a diff", "review this diff", "review the diff", "review implementation", "review an implementation", "review this implementation", "review the implementation", "implementation review", "review patch", "review a patch", "review the patch", "review this patch", "audit code", "audit the code", "audit this code", "audit pull request", "audit a pull request", "audit this pull request", "audit pr", "audit a pr", "audit this pr", "audit diff", "audit this diff", "audit patch", "audit this patch",
}

var anthropicFitSignals = []string{
	"long document", "long-form", "longform", "essay", "narrative", "manuscript", "policy brief", "research brief", "research report", "market analysis", "literature review", "creative writing", "story", "tone", "voice", "brand voice", "editorial", "script", "speech",
}

var visualDesignSignals = []string{
	"visual design", "ui design", "ux design", "interface design", "interaction design", "design system", "mockup", "wireframe", "prototype", "layout", "typography", "color palette", "brand", "branding", "accessibility review", "a11y", "figma",
}

var nuancedRoutineSignals = []string{
	"messy", "inconsistent", "nuanced", "firm but empathetic", "preserve intent", "multiple constraints", "overlap", "overlapping", "edge case", "edge cases", "requirements", "product request", "project plan", "meeting notes", "support reply", "policy", "stakeholder", "prioritize", "triage",
}

var deepReasoningSignals = []string{
	"architecture", "system design", "distributed", "intermittent", "root cause", "tradeoff", "trade-off", "complex", "race condition", "concurrency", "deadlock", "performance", "scalability", "optimize", "profiling", "memory leak", "algorithm", "state machine", "data model", "investigate", "diagnose",
}

var highRiskSignals = []string{
	"security", "auth", "authentication", "authorization", "oauth", "sso", "rbac", "permission", "permissions", "secret", "token", "payment", "billing", "invoice", "production", "data loss", "incident", "compliance", "privacy", "pii", "gdpr", "hipaa", "pci", "encryption", "legal", "medical", "financial", "finance", "access control",
}

var correctnessHeavySignals = []string{
	"parse", "parser", "parsing", "typed comparison", "type comparison", "stable ordering", "stable sort", "edge case", "edge cases", "arbitrarily large", "large values", "big integer", "bignum", "precision", "precise", "lossless", "lossless precision", "required behavior", "current behavior", "preserve behavior", "guarantee", "guarantees", "rounding", "overflow",
}

func classify(task string) taskTraits {
	text := strings.ToLower(task)
	correctnessHeavy := hasAny(text, correctnessHeavySignals...)
	coding := hasAny(text, codingSignals...) || isCorrectnessHeavyCoding(text, correctnessHeavy)
	return taskTraits{
		simple:           hasAny(text, simpleSignals...),
		coding:           coding,
		codeReview:       isCodeReview(text, coding),
		largeContext:     hasAny(text, largeContextSignals...),
		anthropicFit:     hasAny(text, anthropicFitSignals...),
		visualDesign:     hasAny(text, visualDesignSignals...),
		nuancedRoutine:   hasAny(text, nuancedRoutineSignals...),
		deepReasoning:    hasAny(text, deepReasoningSignals...),
		highRisk:         hasAny(text, highRiskSignals...),
		correctnessHeavy: correctnessHeavy,
	}
}

func isCodeReview(text string, coding bool) bool {
	if hasAny(text, codeReviewSignals...) {
		return true
	}
	return coding && hasAny(text, "review", "audit") && hasAny(text,
		"code", "implementation", "pull request", "pr", "diff", "patch", "bug", "bugs", "module", "function", "class", "component", "endpoint", "repository", "repo", "codebase",
		"typescript", "javascript", "golang", "go", "python", "rust", "java", "sql",
	)
}

func isCorrectnessHeavyCoding(text string, correctnessHeavy bool) bool {
	return correctnessHeavy && hasAny(text, "compare", "comparison", "ensure", "fix", "handle", "implement", "make", "preserve", "sort", "support", "validate", "behavior")
}

func hasAny(text string, needles ...string) bool {
	for _, needle := range needles {
		if containsTerm(text, needle) {
			return true
		}
	}
	return false
}

func containsTerm(text, needle string) bool {
	needle = strings.TrimSpace(needle)
	if needle == "" {
		return false
	}

	for start := strings.Index(text, needle); start >= 0; {
		end := start + len(needle)
		if hasBoundary(text, start-1) && hasBoundary(text, end) {
			return true
		}

		next := strings.Index(text[start+1:], needle)
		if next < 0 {
			return false
		}
		start += next + 1
	}
	return false
}

func hasBoundary(text string, index int) bool {
	if index < 0 || index >= len(text) {
		return true
	}
	c := text[index]
	return !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9'))
}
