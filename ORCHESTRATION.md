<!-- generated: eos-ai-scaffold -->
# Orchestration Workflow

1. Planner receives request.
2. Planner creates execution plan.
3. Architect reviews design.
4. Specialized agents execute only their assigned work.
5. Reviewer validates output.
6. Documentation agent updates docs.
7. Return complete implementation.

Steps 3-6 are skipped when they add nothing — a one-line fix needs no design
review or changelog entry. Skipping is a decision to record in
[TASKS.md](./TASKS.md), not something to leave implicit.

## It is a loop, not a line

The numbered flow is the happy path. Real work returns:

- The reviewer rejects → back to the owning role, not to the planner.
- Verification fails → back to implementation, then the **whole** gate re-runs.
- Implementation finds the design wrong → back to the architect, and the
  original decision is corrected in [MEMORY.md](./MEMORY.md) rather than quietly
  replaced.
- A requirement turns out to be ambiguous → back to the planner, who rewrites
  the acceptance criteria explicitly and says they were wrong.

Rework is not failure. Reporting completion to avoid rework is.

## Scaling to the change

| Change | Roles worth involving |
|--------|----------------------|
| One-line fix, no behaviour change | Implementer + verification gate |
| Bug fix | Implementer, testing (regression test), reviewer |
| New feature in an existing surface | Planner, implementer, testing, reviewer, docs |
| New component or dependency | All of the above plus architect and security |
| Anything touching auth, secrets or user data | All of the above; security is not optional |
| Deployment | All of the above plus release |

One agent may hold several of these roles. The rule that does not bend is that
whoever wrote the code is not the one who approves it.

## Rules

- Keep context small.
- Load only required files.
- Never send the entire repository unless necessary.
- Reuse previous work instead of regenerating it.
- Verify before completing.

## Context and token management

When an agent approaches its context limit:

1. Summarize completed work.
2. Save decisions.
3. Hand off to the next agent.
4. Continue from the summary.
5. Do not restart work.
6. Preserve acceptance criteria.

Handing off does not by itself reduce token usage. Every handoff costs a
summary, and the receiving agent pays again to rebuild whatever the summary
omitted. Total cost falls through reusing context already loaded, selecting
files narrowly, and writing short handoffs — not through switching agents as
often as possible. Hand off when an agent is genuinely near its limit, or when
the next task needs a different set of files. Not on a schedule.

A summary that drops an acceptance criterion is worse than no handoff at all:
the work continues in the wrong direction and still looks finished. The
required contents of a handoff are in [HANDOFF.md](./HANDOFF.md).

## Definition of done

A task is complete only when:

- Requirements satisfied.
- Implementation finished — no placeholders, no unfinished files.
- Code compiles.
- Required tests pass, including tests written for this change.
- Code reviewed by someone other than its author.
- Documentation updated.
- No unresolved blockers remain.
- Acceptance criteria satisfied.

If any item is unmet, report the task as incomplete and say which item and why.
Report a check as passing only after running it; if it cannot be run here, mark
it `UNKNOWN` and state what is unverified. See [VERIFY.md](./VERIFY.md).

## Failure handling

If blocked: stop, explain why, identify the missing information, and recommend
the next action. Do not invent a solution to a gap in information, and do not
narrow the task silently to route around the blockage — finish everything that
is not blocked and say explicitly what was left out.
