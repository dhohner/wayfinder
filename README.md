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
Each task category declares a candidate set - a pass rate floor, a credit budget, and a step budget where the category declares one - and the optimization mode selects from that set by an explicit band.
Quality takes the highest pass rate, value uses the lowest-credit cohort within five points of the best, cost uses the lowest-credit cohort clearing the floor and credit budget, and speed takes the fewest benchmark steps within ten points of the best.
Candidates priced within a tenth of the cheapest one are treated as indistinguishable on price, so value and cost separate those by pass rate instead.

The step budget constrains the benchmark proxy for wall-clock latency rather than price, so it does not constrain every mode.
Cost is never narrowed by it: asking for cost says price outranks the latency proxy, so cost accepts a longer run to reach a lower-credit answer.
Speed is always narrowed by it.
Quality and value stay within the budget unless holding it would make either of them answer with a row an excluded row beats on both pass rate and credits, in which case both release it.
The test is on the rows the two modes would actually select, not on every in-budget row: a cheap, low-scoring row that no band ever answers with is no reason to keep a beaten answer in force.
Both modes release together, because a release that fired for one alone could hand value a row scoring above quality's.

The simple category declares no step budget at all.
Its former 40-step ceiling was removed once cost stopped being narrowed by a step budget, because it then changed no selection in any of the four modes, and a budget that binds nothing asserts a latency-proxy commitment the code does not make.
Simple work therefore carries no latency-proxy commitment: quality and value select its longest-running admitted setting, while cost also accepts a substantially longer run than speed.
Those three answers, routine cost, and the ambiguous-task default each state the longer run in their reason line.
Substantive work declares no step budget either and its quality and cost answers run just as long, but they carry no such disclosure: substantive is the category where a long run is the accepted price of the answer, so there is no shorter answer for the reason line to correct.

By default Wayfinder selects GPT 5.6 Sol at high reasoning for substantive and routine coding work, Claude Opus 5 at medium effort for visual, UI, UX, long-form, and creative work, and GPT 5.6 Luna at max reasoning for simple tasks, which is the cheapest of those defaults and also the longest-running.

## German task descriptions

A task described in German is answered in German:

```sh
go run ./cmd/wayfinder --explain "implementiere einen kleinen Go-API-Endpunkt"
```

```text
Modell: GPT 5.6 Luna
Reasoning: GPT-Reasoning-Stufe: max
Begründung: Bester Kompromiss für eine einfache Aufgabe: die beste Trefferquote unter den günstigsten Optionen nahe der höchsten Trefferquote, und sie braucht deutlich mehr Schritte, als eine einfache Aufgabe sonst benötigt.
Benchmark: Pass@1 67%±4%; durchschnittliche Kosten 3.03.
Geschätzte Copilot-AI-Credits: 53.6 (Eingabe- und Ausgabe-Tokens, Schätzwert).
Abwägung: Höchstes Luna-Pass@1 bei geringen Kosten, aber mit deutlichem Abstand die langsamste Einstellung.
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

A simple task in the default mode answers with the cheap, long-running setting and discloses the run:

```sh
go run ./cmd/wayfinder "implement a small Go API endpoint"
```

```text
Model: GPT 5.6 Luna
Reasoning: GPT reasoning level: max
Reason: Best value for a simple task: the best pass rate among the lowest-credit choices near the top score, and it takes noticeably more steps than a simple task usually needs.
```

```sh
go run ./cmd/wayfinder --optimize cost "implement a small Go API endpoint"
```

```text
Model: GPT 5.6 Luna
Reasoning: GPT reasoning level: xhigh
Reason: Best pass rate among the lowest-credit choices that clear the floor for a simple, low-risk task, and it trades a longer run for the lower credit cost.
```

On substantive coding work, quality selects Claude Opus 5 at max effort, value and speed select GPT 5.6 Sol at high reasoning, and cost selects GPT 5.6 Luna at max reasoning for about one twenty-fifth of the credits quality costs.
Visual, UI, and UX work uses Claude Opus 5 at medium effort by default and max effort for quality.
On a simple task, quality and value select GPT 5.6 Luna at max reasoning at 102 benchmark steps and cost selects it at extra-high reasoning at 71, against the 31 steps of the fastest setting the category admits.
On routine work, quality, value, and speed stay inside the routine step budget on GPT 5.6 Sol, while cost leaves that budget for GPT 5.6 Luna at max reasoning and says so in its reason line.
Tasks too vague to classify use the routine cost anchor regardless of the requested optimization, staying deliberately conservative when the request gives too little signal to optimize.

`--optimize speed` is the unchanged route to a short answer:

```sh
go run ./cmd/wayfinder --optimize speed "implement a small Go API endpoint"
```

```text
Model: GPT 5.6 Sol
Reasoning: GPT reasoning level: medium
Reason: Fewest steps for a simple task while staying close to the top pass rate.
```

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

Every recommendation Wayfinder returns is one of these three, across all four categories, all four modes, and the ambiguous-task default.
The bundled data carries rows from other models, and the bands weigh them as candidates, but no band answers with one.
