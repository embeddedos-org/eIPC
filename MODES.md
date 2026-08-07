<!-- generated: eos-ai-scaffold -->
# Modes

A mode is the current objective. Announce the mode you are in, and leave it only
when its exit gate is met. Modes may repeat; the sequence is not fixed.

| Mode | Objective | Leave when |
|------|-----------|------------|
| Discovery | Understand the problem and the affected code. | The requirement can be restated precisely, and the unknowns are named. |
| Planning | Produce an implementation strategy. | Tasks exist in [TASKS.md](./TASKS.md) with acceptance criteria and verification commands. |
| Research | Gather technical information that Discovery could not settle. | The open question is answered, or is recorded as unanswerable with its impact. |
| Architecture | Validate the design against the existing system. | Components, interfaces and non-obvious choices are written down. |
| Implementation | Write code. | The acceptance criteria are met with no placeholders. |
| Verification | Validate the change against evidence. | Every check in [VERIFY.md](./VERIFY.md) carries `PASS`, `FAIL`, `NOT RUN` or `UNKNOWN`. |
| Optimization | Improve quality without changing behaviour. | Behaviour is provably unchanged and the improvement is measured, not asserted. |
| Documentation | Bring the written record back in line with the code. | Docs match the change and every example has been run. |
| Release | Prepare deployment. | Release notes, deployment steps and rollback guidance exist. |
| Maintenance | Resolve defects and technical debt. | The defect is fixed with a regression test, or the debt item is closed. |

## Forbidden in each mode

| Mode | Do not |
|------|--------|
| Discovery | Write code. Propose a solution before the problem is stated. |
| Planning | Write code. Set an acceptance criterion you cannot check. |
| Research | Present a plausible recollection as a finding — cite or mark it `Inferred`. |
| Architecture | Redesign parts the task does not touch. |
| Implementation | Widen scope. Fix unrelated defects — note them instead. |
| Verification | Change code to make a check pass. That is Implementation again. |
| Optimization | Change observable behaviour. Optimise without a baseline number. |
| Documentation | Document intended behaviour as though it shipped. |
| Release | Ship over a `FAIL` or a `NOT RUN`. Omit rollback steps. |
| Maintenance | Fix a defect without a regression test that would have caught it. |

## Loops

The sequence is not linear. Verification failing sends you back to
Implementation; Implementation discovering a bad assumption sends you back to
Architecture or Discovery. That is the system working, not a setback.

Going backwards is cheap. Going forwards on a broken assumption is not.

## Breaking a gate

Leaving a mode with its gate unmet is allowed exactly once per task, and only
when you state:

- Which gate is unmet.
- Why proceeding is safer or cheaper than finishing it.
- What is now unverified as a result.

Record it in [TASKS.md](./TASKS.md). Twice on the same gate means the plan is
wrong — stop and say so rather than continuing to push past it.

## Reporting the mode

Every status report names the current mode. A report that cannot name its mode
usually means the work has no objective, which is worth stopping to fix.
