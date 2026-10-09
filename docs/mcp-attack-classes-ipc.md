# MCP attack classes on the eIPC attack surface

The Agentics' *Enterprise MCP Guide 2026* names six attack classes for
agent tool protocols (68 MCP server CVEs in one month). Every one of
them has an eIPC analogue: the IPC boundary is where a tool registry,
a tool's declared schema, and a peer's privileges meet. This document
maps each class to the eIPC surface and to the existing controls —
the guide is the external validation of the posture, not a new
requirement.

## The six classes, mapped

| # | Attack class | eIPC analogue | eIPC control |
|---|---|---|---|
| 1 | **Tool poisoning** — a tool's description is altered to smuggle malicious instructions | A compromised or malicious peer re-registers an existing tool with a poisoned description, so agents calling it get steered | Tool registration is allowlisted (see `tool-registration-allowlist.md`); descriptions are pinned at registration and re-registration requires re-approval |
| 2 | **Schema poisoning** — the declared parameter schema is widened beyond what the tool honestly accepts | A peer advertises a permissive schema (e.g. extra fields, broader types) that the eIPC schema validator must reject | Schema validation at the boundary; descriptors from peers are validated against the registered schema, never trusted |
| 3 | **Tool shadowing** — a lower-privilege tool overrides a higher-privilege tool's identity | A lower-privilege peer registers a tool under a name that shadows a privileged tool, intercepting calls meant for it | Name registration is privilege-scoped: a peer cannot register a name owned by a higher-privilege namespace; collisions fail closed |
| 4 | **Command injection** — tool arguments become shell commands | Any eIPC-side agent tool that shells out (the Langflow class: CVE-2026-105697, CVSS 9.9) | Executable commands must be declared against an explicit allowlist at registration (`tool-registration-allowlist.md`); no `bash -c` on user-supplied strings |
| 5 | **Shadow servers** — rogue MCP servers the agent discovers and trusts | A rogue IPC endpoint advertising the eIPC protocol to intercept or inject traffic | Endpoint identity binding: IPC endpoints authenticate (capabilities, attestation) before the registry lists them |
| 6 | **Context oversharing** — tools receive more context than they need | A tensor or message payload carries adjacent-process data across the boundary | Zero-copy tensor transport treats descriptors as untrusted; payloads are capability-scoped (Track 1: capability-scoped inference) |

## The guide's control set, as eIPC requirements

The guide's recommended controls map 1:1 onto eIPC's day-one posture:

- **Per-agent allowlists** → the tool-registration allowlist.
- **Identity binding** → endpoint authentication before registry listing.
- **Centralized MCP gateways** → the Strata gateway evaluation
  (agent-fabric track): one policy-enforcement point for all tool traffic.
- **Human approval for destructive actions** → the confirm/deny policy
  hooks on privileged tool invocations.

## Cross-references

- `tool-registration-allowlist.md` — the allowlist requirement (the
  concrete control behind classes 1 and 4).
- `embeddedos-org/eSec` `docs/mcp-hostile-protocol-hardening.md` —
  the org-wide hostile-protocol posture.
- `embeddedos-org/eVera` `docs/mcp-tool-use-policy.md` — the agent-side
  policy (credential isolation, pinned versions, approval on config change).
