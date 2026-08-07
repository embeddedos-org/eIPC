<!-- generated: eos-ai-scaffold -->
# Security Standards

Secure engineering requirements for changes in this repository.

> Reporting a vulnerability is covered by `SECURITY.md`, which is the
> disclosure policy GitHub surfaces. This file is the engineering standard and
> does not replace it.

## Input

Validate every input at the boundary it enters: type, range, length, format and
allowed values. Reject what does not match rather than coercing it. Validation
performed only in the client is not validation.

## Output

Escape for the sink, not in general — HTML, SQL, shell, URL, JSON and log
output each need different treatment. Escaping once and reusing the result
across sinks is a defect.

## Injection

Every one of these is a place code becomes data or data becomes code:

- SQL — use parameterised queries. String interpolation into SQL is never
  acceptable, including for identifiers, which need an allowlist instead.
- Shell — pass argument arrays. Never build a shell string from input.
- Path — resolve and confirm the result stays inside the intended root.
- Template — never render an untrusted string as a template.
- Deserialization — never deserialize untrusted data into arbitrary types.

## Authentication and authorization

- Authenticate at the edge; authorize at every resource.
- Authorization is checked on the server for every request, including ones the
  UI does not offer. A hidden button is not an access control.
- Deny by default. A new endpoint without an explicit rule is closed, not open.

## Secrets

- Never commit a credential, token, key or connection string.
- Never log one, and never return one in an error message or stack trace.
- Never place one in a test fixture, a comment, or an example.
- Read secrets from the environment or a secret store. Do not modify
  environment files without instruction.

## Dependencies

Review every addition: what it does, its maintenance status, its transitive
weight, and its licence. Pin versions. Treat a new transitive dependency as a
new dependency.

## Logging

Log enough to diagnose an incident and nothing that would be damaging if the
log leaked. No credentials, tokens, personal data, or full request bodies from
authenticated endpoints.

## Before reviewing a change

Answer these four. If you cannot, you have not reviewed it yet.

1. What new input does this accept, and from whom?
2. What new thing can a caller now reach, and who is allowed to reach it?
3. What does it now trust that it did not trust before?
4. What would an attacker try first?

## Reporting a finding

State the path from untrusted input to impact:

> **Path:** `POST /api/report` → `body.filename` (unvalidated) → `path.join`
> → file read outside the intended directory.
> **Impact:** arbitrary file disclosure to an unauthenticated caller.
> **Fix:** resolve the path and reject anything outside the root.

A finding without a reachable path is a category name, not a finding. Say
"`path.join` on user input here, but the value is validated at the router two
frames up" rather than filing it — an inflated finding costs the next reviewer
the time to disprove it, and teaches everyone to skim.

Severity is impact times reachability. An unauthenticated remote path beats a
theoretical local one regardless of how the category sounds.

If nothing was found, say so explicitly. Silence reads as "not looked at", and
after enough silences nobody checks.
