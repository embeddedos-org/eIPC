<!-- generated: eos-ai-scaffold -->
# Handoff Protocol

Run this whenever work changes hands — between roles, between agents, or across
a context boundary. The receiving side should need nothing that is not here.

## Required contents

```markdown
## Handoff: <task id> -> <receiving role>

Current mode: <see MODES.md>

Completed work
: <what is finished and verified, with file paths>

Files modified
: <path — one line each, what changed and why>

Architecture decisions
: <choices made and the reason; anything that would otherwise be re-argued>

Open issues
: <known problems, in priority order>

Remaining work
: <what is left, in the order it should be done>

Known risks
: <what could break, and what would show it>

Acceptance criteria
: <copied verbatim from the task — never paraphrased, never trimmed>

Verification status
: | Check | Command | Result |
  |-------|---------|--------|
  | <name> | `<command>` | `PASS` / `FAIL` / `NOT RUN` / `UNKNOWN` |

Next recommended agent
: <role, and why that role>
```

## Rules

- **Acceptance criteria are copied, not summarised.** Every other field may be
  compressed. This one may not — a criterion lost in a summary is work that
  silently stops being done.
- **State what was not verified.** An omitted check reads as a passed check to
  whoever picks the work up.
- **Do not hand off mid-edit.** Leave the tree in a state that builds, or say
  clearly in *Open issues* that it does not and what is broken.
- **Do not restart.** The receiving side continues from the summary; it does not
  redo completed work to satisfy itself. If the summary is not enough to
  continue, that is a defect in the handoff — say so and ask for the gap.

## Receiving a handoff

Before doing any work, check:

1. Are the acceptance criteria present and specific? If they arrived as a
   paraphrase, ask for the original. Do not reconstruct them from the summary —
   that is how the requirement drifts.
2. Does the stated verification match reality? Re-run one cheap check. If the
   handoff says `PASS` and it fails, the rest of the report is now suspect.
3. Is anything in *Completed work* unverifiable from the repository? Treat it as
   `Inferred`, not `Verified`.
4. Does the tree build? If not, and *Open issues* did not say so, say so now.

Then continue. Do not redo completed work to satisfy yourself — if the summary
is not sufficient to continue, that is a defect in the handoff. Name the gap
and ask, rather than silently starting over.

## Cost

A handoff is not free. It costs the summary to write and the context to rebuild
on the other side. Hand off when an agent is genuinely near its limit or the
next task needs a different set of files — not on a schedule, and not because
switching feels like progress.

## Failure modes

- **The optimistic handoff** — everything reported green, nothing run. The
  receiver finds out an hour later.
- **The paraphrased criterion** — "make the endpoint faster" replacing "p99
  under 200ms at 500rps". The next agent hits 400ms and reports success.
- **The mid-edit handoff** — a tree that does not build, handed over without
  saying so.
- **The context dump** — the whole transcript pasted instead of a summary,
  which costs more than continuing would have.
- **The silent drop** — a remaining item that existed in the sender's head and
  in no document.
