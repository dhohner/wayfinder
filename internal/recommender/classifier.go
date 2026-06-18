package recommender

import "strings"

type taskTraits struct {
	simple         bool
	coding         bool
	largeContext   bool
	anthropicFit   bool
	visualDesign   bool
	nuancedRoutine bool
	deepReasoning  bool
	highRisk       bool
}

var simpleSignals = []string{
	"summarize", "summary", "rewrite", "proofread", "copy edit", "format", "extract", "release notes", "short email", "typo", "spelling", "grammar", "readme", "changelog", "rename", "one-line", "small", "minor", "quick", "lint", "comment", "documentation",
}

var codingSignals = []string{
	"code", "coding", "implement", "refactor", "debug", "test", "typescript", "javascript", "golang", "go", "python", "rust", "java", "sql", "api", "sdk", "cli", "module", "bug", "endpoint", "function", "class", "component", "frontend", "backend", "database", "schema", "query", "build", "deploy",
}

var largeContextSignals = []string{
	"large", "long", "many files", "multiple files", "whole repo", "entire repo", "repository", "repo", "codebase", "monorepo", "migration", "cross-service", "multi-service", "integration", "legacy", "10-page", "10 page", "thousands of lines",
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

func classify(task string) taskTraits {
	text := strings.ToLower(task)
	return taskTraits{
		simple:         hasAny(text, simpleSignals...),
		coding:         hasAny(text, codingSignals...),
		largeContext:   hasAny(text, largeContextSignals...),
		anthropicFit:   hasAny(text, anthropicFitSignals...),
		visualDesign:   hasAny(text, visualDesignSignals...),
		nuancedRoutine: hasAny(text, nuancedRoutineSignals...),
		deepReasoning:  hasAny(text, deepReasoningSignals...),
		highRisk:       hasAny(text, highRiskSignals...),
	}
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
