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
Reason: Best fit for complex or high-risk work where stronger reasoning is worth the extra cost.
```

## Supported v1 candidates

- GPT 5.4
- GPT 5.5
- Opus 4.8
- Sonnet 4.6
