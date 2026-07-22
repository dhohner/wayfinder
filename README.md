# Wayfinder

Wayfinder is a Go CLI that recommends one AI model and provider-appropriate reasoning setting for a natural-language task description.

It runs offline from bundled rules.
It does not require API keys, provider credentials, network access, or live model calls.

## Run

```sh
go run ./cmd/wayfinder "refactor a TypeScript auth module and explain the risk"
```

Example output:

```text
Model: GPT 5.6 Sol
Reasoning: GPT reasoning level: high
Reason: Balanced value choice for substantive coding work.
```

Wayfinder selects GPT 5.6 Sol for general and coding work, and Opus 4.8 for visual, UI, UX, long-form, and creative work.
Simple tasks use Sol low, routine work uses medium, and high-risk or deeply complex work uses high.
Quality optimization uses Sol xhigh for substantive coding work to avoid the max-cost jump.

## Optimization flags

Use `--optimize value`, `--optimize cost`, `--optimize speed`, or `--optimize quality` to select the recommendation mode.
The default is `value`.

```sh
go run ./cmd/wayfinder --optimize cost "implement a small Go API endpoint"
```

Visual, UI, and UX work uses Opus 4.8 low by default and high for quality.

## Adversarial code review

Use `--against gpt` or `--against claude` for code-review tasks to choose the opposite model family:

```sh
go run ./cmd/wayfinder --against gpt "review this pull request for bugs"
```

GPT-authored work is reviewed by Opus 4.8.
Claude-authored work is reviewed by GPT 5.6 Sol.
Without `--against`, code review defaults to GPT 5.6 Sol.
The flag is ignored for tasks that are not classified as code review.

## JSON output

Use `--json` for a single machine-readable recommendation document:

```sh
go run ./cmd/wayfinder --json --optimize quality "implement a small Go API endpoint"
```

JSON output uses normalized model and reasoning IDs.
Exact bundled benchmark matches include numeric `pass_at_1` and `average_cost` fields under `benchmark`.

## Primary recommendations

- GPT 5.6 Sol
- Opus 4.8
