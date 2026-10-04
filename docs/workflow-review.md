# v3 workflow review — 2026-10-03

Reviewed the `v3` branch against the Personal/bwenv Linear project. This is a
local implementation review, not a release approval or a real-vault audit.

## Expected daily flow

1. Install the chosen provider CLI and sign in to it once.
2. Use `bwenv config` to save the default hook for new projects.
3. Run `bwenv init` and select the provider, folder and items.
4. Enable the printed shell integration once. Mise projects also need `mise trust`.
5. Run `bwenv login`. The shell wrapper applies the session and selected variables.
6. Enter the project normally. Mise shows a login reminder once per entry if locked.
7. Run `bwenv refresh` after remote changes, and `bwenv doctor` for diagnostics.
8. Use `bwenv disallow` to persist project disablement; `allow`/`login` enables it.
9. Use `bwenv lock` to clear the current shell and block automatic loads until login.

New projects default to native shell activation, following the user's product
choice. Existing project modes are retained. Native shell and mise remain experimental;
direnv remains supported and is the fallback for legacy `.envrc` projects.

## Verified or corrected in this pass

| Area | Result | Linear task |
| --- | --- | --- |
| Failed direnv approval | Non-zero exit, no success summary or shell exports; wrappers retain the original environment | PER-37 |
| Stalled direnv control commands | Allow, deny and reload share a 30-second limit and actionable failures | PER-37 |
| Project commands in subdirectories | Login, allow, refresh, disallow and remove resolve the nearest project root | Flow review |
| Native cleanup after disallow | Repeated disallow and subsequent remove preserve restored original values; cached names do not own inactive native variables | Flow review |
| Provider selection | Canonical FolderID and selected item IDs remain authoritative | PER-38 |
| Cached unset names | Both variable-name cache and legacy exports reject injected shell syntax; real Bash eval test covers both | PER-39 |
| Canonical activation source | Native shell and mise share one resolver | PER-51 |
| Mise secret handling | Removed generated plaintext `.env` cache; legacy cache is removed on adapter installation/reload | Related to PER-23/PER-25/PER-30 |
| Documentation | Added current architecture, provider, security and contribution references | PER-33 |
| Persistent native/mise disable | Stored in canonical activation metadata; re-entry and fresh shells do not fetch until allow/login | PER-72 |
| Current-shell lock/logout | Applies safe cleanup even on provider failure, clears session variables and blocks all automatic retrieval until successful login | PER-72, prerequisite for PER-23 |
| Secret-safe output / threat model | Signin stdout isolated, runner/decoder errors redacted with typed causes preserved, NUL values rejected before output; documented trust boundaries | PER-25 |
| CLI/application separation | Thin canonical and compatible entry points, strict syntax parsing, typed workflow coordination and shared release metadata | PER-15 |
| Mise script lifecycle | Refresh upgrades generated scripts while preserving custom scripts; name metadata belongs to the referenced project | PER-72 |

## Remaining gaps and delivery order

1. **PER-22/PER-23:** optional authenticated local agent and memory-only cache.
   Mise can still refetch on prompt evaluation; removing the disk cache restores
   the security contract but does not deliver zero-call warm prompts.
2. **PER-24:** release provenance. The threat model and output hardening are
   implemented under PER-25; platform checks and future agent protections remain
   required before claiming the entire security milestone complete.
3. Complete platform/provider validation. Local tests exercise Bash/Zsh and real
   mise/direnv with fake provider data; Fish is skipped when unavailable. No claim
   is made here about current remote CI or real-vault performance.

Lock cleanup is scoped to the invoking shell. Other terminals/children retain
their own environments; there is no shared agent to purge yet. Removing a
service-account token locally does not revoke it remotely. Native/mise disablement
is a project setting, while direnv approval remains local to direnv.

The project description and roadmap now record the native-default decision;
older issue proposals that mention a direnv default should follow that decision.
Several old backlog issues already have fixes in `v3`; status alone is not evidence
that work remains. PER-33 was marked Done although its requested documents were
absent from this checkout and the local reference branches.
