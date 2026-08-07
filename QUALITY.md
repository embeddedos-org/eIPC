<!-- generated: eos-ai-scaffold -->
# Quality Standards

Code produced here is readable, maintainable, modular, secure, testable,
documented, and consistent with the architecture already in place.

## Structure

- Single responsibility per unit. A function that needs "and" to describe it is
  usually two functions.
- Keep functions small enough to hold in your head at once.
- Depend on interfaces, not concrete implementations, where the seam is real.
  Do not add a seam that has one implementation and no prospect of a second.
- Prefer reuse over replacement. Search for an existing utility before writing
  a new one.
- Avoid duplication — but duplication is cheaper than the wrong abstraction.
  Wait until the third occurrence before generalising.

## Naming and style

- Follow the style already in the file. Consistency beats personal preference.
- Names say what the thing is, not what type it has.
- Comments explain why, not what. Code that needs a comment to say what it does
  is usually asking to be rewritten.

## Documentation

Every public function carries a description, its parameters, its return value,
and an example. The example must be one that was actually run.

## Errors

- Return clear, specific errors. A caller must be able to act on one.
- Distinguish recoverable from unrecoverable.
- Log enough to diagnose, never enough to leak. See
  [SECURITY-STANDARDS.md](./SECURITY-STANDARDS.md).
- Never fail silently. A swallowed error is a defect that will be found later
  and cost more.

## Performance

Prefer better complexity to micro-optimisation. Reach for caching, streaming,
lazy loading and pagination when the data justifies them.

Do not optimise on suspicion. Measure, change, measure again, and report both
numbers. An optimisation without a before and after is an assertion, not a
result.

## Dependencies

- Minimise them. Every dependency is a maintenance and security commitment.
- Justify each addition: what it does, why the standard library will not,
  and what its maintenance status is.
- Never add one to avoid writing twenty lines.

## Changes

The smallest safe change is the one where every line can be traced to the
requirement. Concretely:

- Touch a file only if the requirement needs it touched.
- Do not reformat unrelated code — it buries the real change in the diff and
  makes review a search problem.
- Do not fix an unrelated defect in the same change. Note it, finish the task,
  report it separately.
- Do not add abstraction for a second case that does not exist yet.
- Do not rename things you are merely passing through.
- Do not upgrade a dependency as a side effect of a feature.

"Smallest" is not "least effort". Deleting a needed check is smaller and less
safe. When the smallest safe change and the cleanest design disagree, say so
and let the reviewer decide, rather than deciding by silently doing the larger
one.

## Working on unfamiliar code

Read enough to know what depends on what you are changing before changing it.
The cost of reading is bounded; the cost of a change made without understanding
is not.

If the existing code does something that looks wrong, assume there was a reason
until you find otherwise. Record what you find in [MEMORY.md](./MEMORY.md) — the
next person will have the same suspicion.
