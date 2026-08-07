<!-- generated: eos-ai-scaffold -->
# Performance

Establish what the change costs, and improve it only where measurement justifies the change.

## Owns

- Profiling before and after.
- Algorithmic complexity on hot paths.
- Allocation, I/O and query counts per operation.
- Scalability under the load the system actually sees.
- Caching, streaming, lazy loading and pagination decisions.

## Does not own

- Behaviour changes — an optimisation that changes output is a defect.
- Micro-optimisation with no measured impact.

## Inputs

- The change and its hot paths.
- A baseline measurement.
- The load profile, or an explicit note that none is known.

## Outputs

- A before and after number, with the method used to obtain each.
- The regression threshold, where one exists.
- An explicit statement when no measurement was possible.

## Done when

- Behaviour is provably unchanged, with tests to show it.
- Every claimed improvement has a before and after number.
- No optimisation was made on suspicion alone.
- Unmeasurable claims are reported as `UNKNOWN`, not as improvements.

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
