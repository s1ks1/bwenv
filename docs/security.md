# Security and threat model

## Scope and trusted components

bwenv intentionally exports decrypted values into the invoking shell. Its goal is
to keep those values out of project files and diagnostics and to generate shell
code that cannot execute a value as a command. It is not a sandbox for programs
launched from that shell. The OS, user account, bwenv binary, password-manager CLI,
shell RC and installed hook integrations must be trusted.

A repository author, a malformed provider response or modified runtime metadata
must not be able to turn an export/unset into arbitrary shell syntax. A process
running as the same user, an administrator, or a compromised provider CLI is
outside this confidentiality boundary: such a process can already read or change
runtime credentials. Project/folder/item names and IDs are metadata, not secret
storage, and may appear in diagnostics.

## Boundaries and controls

| Surface / threat | Implemented control and practical limit |
| --- | --- |
| Project files and version control | `.bwenv.toml` contains references and activation settings, not sessions or decrypted exports. Generated canonical metadata uses mode 0600. `.bwenv_vars` stores validated names only. Legacy `.envrc` may contain a session; migration preserves it in a private backup rather than silently deleting it. Remove legacy credentials from tracked files and rotate any exposed token. |
| Shell expressions and forged state | Keys become valid POSIX identifiers; reserved hook-control names are rejected. Values are quoted as literals for Bash/Zsh or Fish. All values are checked before emitting assignments; NUL bytes are rejected. Disk-derived unset names and decoded restoration state are validated before execution. Invalid state never becomes arbitrary shell syntax. |
| Executable stdout | `export`, native activation/login/refresh and cleanup commands emit assignments/unsets only. Human messages use stderr. Wrappers apply successful output; lock/logout deliberately apply validated cleanup even when provider locking fails, retaining the failure status. Help, version, root, hook and benchmark have separate output formats and must not be treated as secret exports. |
| Provider responses and logs | Raw non-interactive stdout/stderr and JSON bodies are never printed on failure. Runner/decoder errors are rendered through a fixed safe message; typed causes remain available to `errors.Is`/`errors.As`. Measurements record operation timings and executable counts, not arguments, environment or payloads. Do not add raw payload logging or format unwrapped causes in user diagnostics. |
| Interactive authentication | Prompts from the trusted CLI may use stdin/stderr. Bitwarden unlock captures its token in memory. 1Password signin stdout is discarded and never copied into shell-evaluated output; desktop integration manages authentication. Existing valid `OP_SESSION_*` or service-account credentials are supported through the environment. bwenv does not parse/eval arbitrary signin output. |
| Shell history and terminal capture | Use the installed wrapper (`bwenv login`) instead of typing a literal token or password into a command. Do not print `BW_SESSION`, `OP_SESSION_*`, `_BWENV_STATE`, full `env`, shell traces (`set -x`) or generated assignments in shared logs. Direct `bwenv export` intentionally writes secret assignments; redirect/eval it only into a trusted destination. bwenv cannot remove terminal scrollback, recorded sessions or existing shell history. |
| Malicious `.envrc`, mise script or RC | Trusting/sourcing arbitrary shell files executes arbitrary code. bwenv quoting does not make a custom activation file safe. Review repository scripts before `direnv allow`/`mise trust`, including later changes. bwenv refuses unsupported custom code during migration rather than evaluating it. Native activation reads canonical metadata, but metadata can request a different provider source; review those references too. |
| Files, symlinks and temporary data | Generated mise scripts fetch into process memory; no decrypted `.env` cache is written. Known legacy mise caches are removed on installation/reload. Migration temp files/backups use restrictive permissions and contain legacy content, which may include old tokens. Do not run setup/migration in a directory writable by an untrusted user: current project-file operations are not a universal symlink/race defense. File modes do not stop same-user access and do not substitute for Windows ACL verification. |
| Sessions and subprocesses | Tokens travel in child environment, not argv. Automatic provider commands have deadlines; cancellation bounds inherited pipe waits. Native original-value state is base64-encoded in `_BWENV_STATE`; this is encoding, not encryption. Values and session copies remain in process memory and child environments. |
| Lock and project disablement | Lock restores/clears the invoking shell's managed variables, removes local provider session variables and blocks automatic loads there until successful login. Cleanup runs even if provider locking fails. Other terminals and existing children keep their own copies. Removing a service-account token locally does not revoke it remotely. Native/mise disablement is persisted in project config; direnv trust is local to direnv. |
| IPC and future agent | No bwenv daemon/socket or shared cache currently exists. PER-22/PER-23 must authenticate the local user, restrict socket/pipe access, bound requests, avoid secret logs/disk caches, and implement TTL/invalidation/purge before claiming agent confidentiality or zero-call warm loads. |
| Crash dumps, debuggers and swap | Go strings, provider subprocess buffers and shell environments can retain plaintext; guaranteed zeroization is not provided. Core dumps, debugger sessions, OS crash collection or swap can contain values. Use appropriate OS controls for the deployment environment; do not attach raw dumps to public issues. Lock cannot retroactively erase these copies. |

The [1Password signin reference](https://www.1password.dev/cli/reference/commands/signin)
explains desktop integration and token-bearing output. bwenv keeps that output
outside its export stream; configure the desktop integration or supply an already
valid session/service-account environment for non-desktop use.

## Regression evidence and release checks

Tests exercise literal quoting with real shells when available, malicious
variable-name metadata through Bash eval, forged restoration state, reserved
control names, NUL rejection before partial output, provider payload/error
redaction, signin output isolation, approval failures, quiet authentication exit
status, lock failure cleanup and compatibility of both executable entry points.
Fake CLIs keep these tests independent of real vault contents.

Local tests do not prove every shell/OS/provider version. Fish is skipped when
unavailable; Windows ACL behavior, current remote CI, real-vault compatibility
and release provenance require their own release checks. Security work on this
branch does not imply the optional agent or all later security milestones exist.
See the [workflow review](workflow-review.md) and
[SECURITY.md](../SECURITY.md) for delivery gaps and vulnerability reporting.
