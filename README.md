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

Wayfinder varies recommendations by task complexity and model family. Simple, low-risk work uses low-reasoning GPT 5.5, substantive coding defaults to higher GPT 5.5 reasoning, long-form creative or visual design work can use Opus with Anthropic Effort Level terminology, and ambiguous input receives a conservative offline default instead of a clarification prompt.

## Optimization flags

Use `--optimize value`, `--optimize cost`, `--optimize speed`, or `--optimize quality` to select the recommendation mode. The default is `value`:

```sh
go run ./cmd/wayfinder --optimize cost "implement a small Go API endpoint"
```

For substantive coding, value selects GPT 5.5 high, cost and speed select medium, and quality selects xhigh. Genuinely simple coding stays at low except quality, which raises it to medium.

## JSON output

Use `--json` for a single machine-readable recommendation document:

```sh
go run ./cmd/wayfinder --json --optimize quality "implement a small Go API endpoint"
```

JSON output uses normalized model and reasoning IDs. Exact bundled benchmark matches include numeric `pass_at_1`, `aic`, and `aic_factor` fields under `benchmark`; recommendations without an exact match omit `benchmark`.

## Primary v1 recommendations

- GPT 5.5
- Opus 4.8
