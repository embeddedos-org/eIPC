<!-- generated: eos-ai-scaffold -->
# Verification

Do not report success without evidence. This file defines what counts.

## States

- `PASS` — the check ran and succeeded; the output was seen.
- `FAIL` — the check ran and failed; quote the actual output.
- `NOT RUN` — the check was not executed.
- `UNKNOWN` — the check cannot be run in this environment, or its result cannot be interpreted.

`NOT RUN` and `UNKNOWN` are acceptable answers, and reporting one honestly is
never a failure. A guess dressed as `PASS` is.

## Evidence classes

- **Verified** — evidence exists; a command was run and its output observed.
- **Observed** — directly visible in the code or repository.
- **Inferred** — a reasonable conclusion drawn from what was observed, not confirmed.
- **Assumed** — information is missing and a default was taken.
- **Unknown** — cannot be determined from what is available.

## What a check requires

A check is `PASS` only when all four hold:

1. A command was executed.
2. Its output was read.
3. Its exit status was zero, or the output was otherwise unambiguous.
4. It exercised the change — a suite that skipped every relevant test is not a
   pass, it is `NOT RUN` with extra steps.

Anything short of that is `NOT RUN` or `UNKNOWN`.

### Valid

> `pnpm test:unit` — exit 0. 142 passed, 0 failed, 3 skipped. The 3 skips are
> pre-existing and unrelated to this change. → `PASS`

> `pnpm check` — exit 2. `src/api/user.ts(41,7): error TS2345: Argument of type
> 'string | undefined' is not assignable to parameter of type 'string'.`
> → `FAIL`

> No linter is configured in this repository. → `NOT RUN`

> The e2e suite needs a running database, which is not available here.
> → `UNKNOWN` — e2e behaviour is unverified.

### Not valid

> The tests should pass now.

> I've made the change, so the build will work.

> Everything looks good.

> `PASS` (no command, no output)

> Tests pass — *(when the suite was never run, only written)*

The difference is not tone. The first set can be checked by someone else; the
second set cannot.

## The gate

Applies to every change, to the extent each item exists here:

| # | Check | Applies when |
|---|-------|--------------|
| 1 | Build | The repository has a build step. |
| 2 | Type check | The language or toolchain has one. |
| 3 | Lint | A linter is configured. |
| 4 | Unit tests | Always, including tests written for this change. |
| 5 | Integration tests | The change crosses a component boundary. |
| 6 | End-to-end tests | The change affects a user-visible flow. |
| 7 | Security review | The change touches input, auth, secrets or dependencies. |
| 8 | Performance review | The change touches a hot path or adds work per request. |
| 9 | Accessibility review | The change alters UI. |
| 10 | Documentation review | The change alters behaviour, API or setup. |

Rules:

- Never report a step as passing without having run it.
- If a check fails, fix it and re-run **the whole gate**, not just the failed
  check. A fix routinely breaks something that passed ten minutes earlier.
- If a check cannot be run here, mark it `UNKNOWN` and say what that leaves
  unverified. Do not infer the result from the shape of the change.
- Report failures with actual output, not a paraphrase.
- For a UI change, render it and look at it. A UI is never described as correct
  on the strength of the diff alone.
- Run the gate before reporting completion, not after being asked whether you
  did.

## How completion gets faked

These are the specific failure modes this standard exists to catch. They are
listed because they are easy to fall into while acting in good faith.

- **Inferring a result from the diff.** The change looks right, therefore the
  tests pass. This is the single most common false `PASS`.
- **Writing tests and reporting them as run.** Authoring is not execution.
- **Editing the test until it goes green** without establishing which of the
  test and the code was wrong.
- **Deleting or skipping a failing test** to clear the board. If a test must be
  skipped, say so, in the report, with the reason.
- **Reporting the suite, not the change.** "All tests pass" while every test
  touching the new code was skipped or never written.
- **Summarising acceptance criteria** in a handoff, so the next agent satisfies
  a weaker requirement and both of you report success.
- **Silent narrowing.** Dropping the hard third of the task and reporting the
  easy two thirds as complete.
- **Treating a partially-run gate as a passed gate.** Six of ten checks green
  and four never attempted is not a pass — it is six passes and four `NOT RUN`.
- **Fixing the check instead of the code** — loosening a lint rule, widening a
  type, raising a timeout — without saying that is what happened.

If you notice yourself doing one of these, say so in the report. Catching it
costs a sentence; missing it costs whatever ships.

## Report format

```markdown
Status: <complete | incomplete | blocked>
Current mode: <see MODES.md>

Completed work: <what was done>
Files changed: <paths>

Verification:
| Check | Command | Result |
|-------|---------|--------|
| Build | `<cmd>` | `PASS` — exit 0 |
| Unit tests | `<cmd>` | `FAIL` — <actual output> |
| Lint | — | `NOT RUN` — no linter configured |

Remaining work: <what is left>
Known risks: <what could break, and what would reveal it>
Assumptions: <each one, labelled per the evidence classes above>
Recommended next step: <action, and which role>
```

A report with no `NOT RUN` and no `UNKNOWN` rows deserves a second look. It is
occasionally true, and more often a sign the gate was filled in from memory.
