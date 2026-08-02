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
Reason: Best value for substantive work: within a few points of the top pass rate at a fraction of the credits.
```

Wayfinder does not hand-pick a model per task.
Each task category declares a candidate set - a pass rate floor plus, for cheap categories, a credit and step ceiling - and the optimization mode selects from that set by an explicit band.
Quality takes the highest pass rate, value the cheapest candidate within five points of the best, cost the cheapest candidate clearing the floor, and speed the fewest benchmark steps within ten points of the best.

By default Wayfinder selects GPT 5.6 Sol at high reasoning for substantive coding work, Claude Opus 5 for visual, UI, UX, long-form, and creative work, and cheaper, shorter settings for routine and simple tasks.

## German task descriptions

A task described in German is answered in German:

```sh
go run ./cmd/wayfinder --explain "implementiere einen kleinen Go-API-Endpunkt"
```

```text
Modell: Claude Opus 5
Reasoning: Anthropic-Effort-Stufe: low
Begründung: Bester Kompromiss für eine einfache Aufgabe: die günstigste Wahl wenige Punkte unter der höchsten Trefferquote.
Benchmark: Pass@1 58%±2%; durchschnittliche Kosten 1.66.
Geschätzte Copilot-AI-Credits: 163.1 (Eingabe- und Ausgabe-Tokens, Schätzwert).
Abwägung: Günstigste Opus-5-Einstellung, deutlich unter dem Pass@1 der höheren Stufen.
```

There is no language flag: the language follows the prompt.
Detection is an offline heuristic biased toward English, so output switches to German only when the task text carries at least two distinct German-only words.
An English prompt containing a single German word stays English.
Detection ignores capitalization, which is why German function words that English technical prompts also use as words, identifiers, or acronyms are not markers at all.

Language never changes the recommendation.
A German prompt and its English equivalent select the same model and the same effort level, and JSON output is unaffected: field names, the normalized model and reasoning IDs, the profile, and the benchmark keys stay English, while `reason` and `tradeoff` carry German text.
Effort levels, model names, and `Pass@1` are provider-defined identifiers and stay untranslated in both languages.

## Optimization flags

Use `--optimize value`, `--optimize cost`, `--optimize speed`, or `--optimize quality` to select the recommendation mode.
The default is `value`.

```sh
go run ./cmd/wayfinder --optimize cost "implement a small Go API endpoint"
```

On substantive coding work, quality selects Claude Opus 5 at max effort, value and speed select GPT 5.6 Sol at high reasoning, and cost selects GPT 5.6 Luna at max reasoning for about one twenty-fifth of the credits quality costs.
Visual, UI, and UX work uses Claude Opus 5 at medium effort by default and max effort for quality.

## Adversarial code review

Use `--against gpt` or `--against claude` for code-review tasks to choose the opposite model family:

```sh
go run ./cmd/wayfinder --against gpt "review this pull request for bugs"
```

GPT-authored work is reviewed by Claude Opus 5.
Claude-authored work, and code review without `--against`, runs the unrestricted substantive candidate set: GPT 5.6 Sol at high reasoning by default.
The flag is ignored for tasks that are not classified as code review.

## JSON output

Use `--json` for a single machine-readable recommendation document:

```sh
go run ./cmd/wayfinder --json --optimize quality "implement a small Go API endpoint"
```

JSON output uses normalized model and reasoning IDs.
Exact bundled benchmark matches include numeric `pass_at_1`, `average_cost`, and `credits_estimate` fields under `benchmark`.
`credits_estimate` is the estimated GitHub Copilot AI credit cost of the whole benchmark run, counting output tokens at the GA output rate and input tokens at a blended rate that assumes a 95% cache-hit ratio.
Because that ratio is modelled rather than measured, treat it as an estimate accurate to roughly a quarter, not a price.
`--explain` reports the same figure in text output alongside the benchmark evidence.

## Primary recommendations

- Claude Opus 5
- GPT 5.6 Sol
- GPT 5.6 Luna
- GPT 5.6 Terra
