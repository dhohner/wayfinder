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

func classify(task string) taskTraits {
	text := strings.ToLower(task)
	return taskTraits{
		simple:         hasAny(text, "summarize", "summary", "rewrite", "proofread", "format", "extract", "release notes", "short email", "typo", "readme", "rename", "one-line", "small", "minor", "lint", "comment", "documentation"),
		coding:         hasAny(text, "code", "coding", "implement", "refactor", "debug", "test", "typescript", "golang", "go", "python", "api", "module", "bug", "endpoint", "function"),
		largeContext:   hasAny(text, "large", "long", "many files", "repository", "repo", "codebase", "migration", "cross-service", "multi-service", "10-page", "10 page"),
		anthropicFit:   hasAny(text, "long document", "long-form", "longform", "essay", "narrative", "manuscript", "policy brief", "research brief", "research report", "market analysis", "literature review", "creative writing", "story", "tone", "voice"),
		visualDesign:   hasAny(text, "visual design", "ui design", "ux design", "interface design", "interaction design", "design system", "mockup", "wireframe", "prototype", "layout", "typography", "color palette", "brand", "branding"),
		nuancedRoutine: hasAny(text, "messy", "inconsistent", "nuanced", "firm but empathetic", "preserve intent", "multiple constraints", "overlap", "overlapping", "edge case", "requirements", "product request", "project plan", "meeting notes", "support reply", "policy"),
		deepReasoning:  hasAny(text, "architecture", "system design", "distributed", "intermittent", "root cause", "tradeoff", "complex", "race condition", "concurrency", "performance", "scalability"),
		highRisk:       hasAny(text, "security", "auth", "authentication", "payment", "billing", "production", "data loss", "incident", "compliance", "privacy", "encryption", "permissions"),
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
