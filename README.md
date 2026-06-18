# Wayfinder

Wayfinder is a Go CLI that recommends one AI model and provider-appropriate reasoning setting for a natural-language task description.

It runs offline from bundled rules. It does not require API keys, provider credentials, network access, or live model calls.

## Run

```sh
go run ./cmd/wayfinder "refactor a TypeScript auth module and explain the risk"
```

Example output:

```text
Model: GPT 5.5
Reasoning: GPT reasoning level: high
Reason: Balanced value choice for substantive coding work.
```

Wayfinder varies recommendations by task complexity and model family. Simple, low-risk work uses low-reasoning GPT 5.5, routine coding uses medium GPT 5.5 reasoning, high-risk or deeply complex coding uses higher GPT 5.5 reasoning, visual/UI/UX design work uses Opus with low Anthropic Effort Level by default, long-form creative work can use Opus, and ambiguous input receives a conservative offline default instead of a clarification prompt.

## Optimization flags

Use `--optimize value`, `--optimize cost`, `--optimize speed`, or `--optimize quality` to select the recommendation mode. The default is `value`:

```sh
go run ./cmd/wayfinder --optimize cost "implement a small Go API endpoint"
```

For routine coding, value, cost, and speed select GPT 5.5 medium, while quality raises it to high. High-risk, correctness-heavy, large-context, or deeply complex coding still uses high by default and xhigh for quality. Genuinely simple coding stays at low except quality, which raises it to medium. Visual/UI/UX design stays on Opus 4.8 low for value, cost, and speed; quality raises it to Opus 4.8 medium.

## Adversarial code review

Use `--against gpt` or `--against claude` for code-review tasks to choose the opposite model family:

```sh
go run ./cmd/wayfinder --against gpt "review this pull request for bugs"
```

GPT-authored work is reviewed by Opus 4.8. Claude-authored work is reviewed by GPT 5.5. Without `--against`, code review defaults to GPT 5.5. The flag is ignored for tasks that are not classified as code review.

## JSON output

Use `--json` for a single machine-readable recommendation document:

```sh
go run ./cmd/wayfinder --json --optimize quality "implement a small Go API endpoint"
```

JSON output uses normalized model and reasoning IDs. Exact bundled benchmark matches include numeric `pass_at_1`, `aic`, and `aic_factor` fields under `benchmark`; recommendations without an exact match omit `benchmark`.

## Primary v1 recommendations

- GPT 5.5
- Opus 4.8
