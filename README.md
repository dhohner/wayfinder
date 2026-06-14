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
Reason: Best fit for complex or high-risk work where stronger reasoning is worth the extra cost; preference does not override the task's risk.
```

Wayfinder varies recommendations by task complexity and model family. Simple, low-risk work uses low-reasoning GPT 5.5, complex or high-risk development work receives stronger GPT 5.5 reasoning, long-form creative or visual design work can use Opus with Anthropic Effort Level terminology, and ambiguous input receives a conservative offline default instead of a clarification prompt.

## Preference flags

Use `--prefer quality`, `--prefer cost`, or `--prefer speed` to bias the recommendation when the task traits support it:

```sh
go run ./cmd/wayfinder --prefer cost "implement a small Go API endpoint"
```

Preferences do not blindly override complexity or risk. For example, high-risk tasks still receive stronger reasoning even with `--prefer cost` or `--prefer speed`, while `--prefer quality` can raise high-risk or complex work to the strongest GPT 5.5 reasoning setting.

## Primary v1 recommendations

- GPT 5.5
- Opus 4.8
