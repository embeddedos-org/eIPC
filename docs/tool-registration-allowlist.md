# Tool registration: the command-allowlist requirement

**Status:** requirement (2026-10-08). Any eIPC-side agent tool
registration MUST declare executable commands against an explicit
allowlist. This is not guidance; it is a precondition for registration.

## Why: the Langflow class

**CVE-2026-105697 (CVSS 9.9, disclosed 2026-10-05):** Langflow's MCP
server configuration took the user-typed `command`, wrapped it in
`bash -c`, and executed it -- with **no allowlist**. A configuration
file became arbitrary OS command execution on the host. Patch: upgrade
to 1.10.3.

The defect is a *class*, not an incident: any surface where a tool
registration names something to execute, and the runtime executes it
without checking the name against an allowlist, is the same
vulnerability. eIPC's tool registration is such a surface.

## The requirement

1. **Every tool registration declares its executable surface.**
   If a tool can cause command execution -- a shell command, a binary
   path, a script interpreter invocation -- the registration lists the
   exact commands/paths allowed. No declaration, no registration.
2. **Config-supplied commands are untrusted input.** A command arriving
   via configuration is not "configuration"; it is attacker-controlled
   data until it matches the allowlist. It is never interpolated into a
   shell string -- no `bash -c` wrapping, ever. Execution uses argv
   arrays against allowlisted absolute paths.
3. **Registration is versioned and reviewed.** Changing the allowlist is
   a reviewed change, not a config edit. The allowlist ships with the
   tool definition, not beside it.

## The companion: fetch-style tools and SSRF

**CVE-2026-104120 (published 2026-10-02):** `mcp-server-fetch`
<= 2026.6.4 allowed SSRF via `fetch_url`, with a publicly disclosed
exploit. Fetch-style tools are network egress: they get allowlisted
destinations (or an egress proxy) and the same "untrusted input"
treatment as commands. Being "just a tool" grants no implicit trust.

## The protocol-level principle

These CVEs sit under the hostile-boundary principle from yesterday's
design work: **never trust data crossing the IPC boundary because it
came from inside**. For the zero-copy tensor transport this means
descriptors are validated at the protocol level -- a tensor header from
a peer process is untrusted input even when the peer is "one of ours."

## Cross-references

- eSec `docs/mcp-hostile-protocol-hardening.md` -- the org-wide hostile-protocol posture.
- eosllm `docs/threat_model.md` -- the same CVEs mapped to the inference-service surface.
- eVera `docs/mcp-tool-use-policy.md` -- the agent-side policy these requirements mirror.
