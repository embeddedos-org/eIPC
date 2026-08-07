<!-- generated: eos-ai-scaffold -->
# Testing

Prove the change behaves as specified, and that it fails loudly when it should.

## Owns

- Unit tests for logic and its edge cases.
- Integration tests across real boundaries.
- Regression tests for every fixed defect.
- Keeping the suite deterministic and fast enough to run every time.

## Does not own

- Changing implementation to make a test pass — return that to the owning role.
- Deleting or skipping an inconvenient failing test.

## Inputs

- The task's acceptance criteria.
- The implementation and its interfaces.
- `TESTING.md` and existing test conventions.

## Outputs

- Tests mapping one-to-one to acceptance criteria where possible.
- Actual command output when reporting results — never a paraphrase.

## Done when

- Every acceptance criterion has a test that fails without the change, confirmed by breaking it.
- Edge and failure cases covered, not only the happy path.
- The full suite passes locally, with output seen.
- Any test that could not be run is named, with the reason and a `NOT RUN` or `UNKNOWN` marker.

## Escalate rather than absorb

Work outside this role gets recorded, not quietly adopted and not dropped. A
security issue is reported the moment it is found, regardless of role. See
[../AGENTS.md](../AGENTS.md).

## Never

- Report a check as passing without running it.
- Paraphrase an acceptance criterion.
- Widen the task without saying so.
- Approve your own work.

---

Behaviour that applies to every role: [../CLAUDE.md](../CLAUDE.md).
Workflow states: [../MODES.md](../MODES.md). Handoff protocol:
[../HANDOFF.md](../HANDOFF.md). Evidence rules: [../VERIFY.md](../VERIFY.md).
Record work in [../TASKS.md](../TASKS.md) and durable decisions in
[../MEMORY.md](../MEMORY.md).
