<!-- generated: eos-ai-scaffold -->
# Agent Responsibilities

Each role owns a slice of the work and does only that slice. Full briefs are in
[.ai/](./.ai/). These are responsibilities, not a required agent count — one
agent may hold several roles on a small change. Split when the roles need
genuinely different context, not by default.

One rule is structural rather than stylistic: **whoever implements does not
approve.** Review is a separate role because self-review reliably misses the
thing the implementer already believes is correct.

## Planner — [.ai/planner.md](./.ai/planner.md)

- Understand the request.
- Break work into tasks.
- Assign work.

## Architect — [.ai/architect.md](./.ai/architect.md)

- Design structure.
- Choose patterns.
- Own dependencies, scalability and maintainability.

## Backend — [.ai/backend.md](./.ai/backend.md)

- APIs
- Database
- Business logic

## Frontend — [.ai/frontend.md](./.ai/frontend.md)

- UI
- Components
- Accessibility

## Testing — [.ai/testing.md](./.ai/testing.md)

- Unit tests
- Integration tests
- Regression tests

## Security — [.ai/security.md](./.ai/security.md)

- Authentication and authorization
- Validation
- Secrets
- Dependency review

## Performance — [.ai/performance.md](./.ai/performance.md)

- Profiling
- Optimization
- Scalability

## Reviewer — [.ai/reviewer.md](./.ai/reviewer.md)

- Final review
- Verify requirements
- Merge findings

## Documentation — [.ai/docs.md](./.ai/docs.md)

- README
- API docs
- Changelog
- Migration and architecture notes

## Release — [.ai/release.md](./.ai/release.md)

- Release notes
- Deployment preparation
- Rollback guidance

---

## Switching roles

Switch when the task changes domain, when specialist knowledge is required,
when independent review is required, or when the context has grown past what
one agent can hold accurately. Every switch runs the protocol in
[HANDOFF.md](./HANDOFF.md).

## Finding work that is not yours

You will. The rule is: **record it, do not absorb it, do not drop it.**

| What you found | Do |
|----------------|-----|
| A defect unrelated to your task | Note it in [TASKS.md](./TASKS.md) and keep going. |
| A defect your change would sit on top of | Stop; say it blocks you; propose fixing it as its own task. |
| A security issue | Report immediately, whatever role you hold. This one never waits for a handoff. |
| A design decision missing from the plan | Return to the architect rather than deciding it inside an implementation. |
| Work that belongs to a role nobody assigned | Say so. An unowned task is how requirements go missing. |

Silently fixing something outside your task makes the diff unreviewable.
Silently ignoring it means nobody ever looks again. Neither is acceptable; the
note is what makes the difference.
