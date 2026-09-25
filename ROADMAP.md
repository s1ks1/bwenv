# bwenv Roadmap

> **Project:** `s1ks1/bwenv`
> **Current public baseline:** `v2.3.0`
> **Roadmap scope:** performance, architecture, security, activation model, DX, testing, CI/CD and future extensibility
> **Primary goal:** make bwenv feel instant in normal shell usage while keeping secret handling safe and the project maintainable as it grows.

## Implementation status

**v2.2.0 (2026-09-17):** Phase 0 is complete and released. The benchmark
command, injectable provider process runner, deterministic fake CLI tests,
cross-platform CI workflow, structural baseline, and release guardrails are
implemented. The next development line is v2.4.0, focused on migration and
diagnostics.

**v2.3.0 (2026-09-17):** FolderID persistence, optimistic non-interactive
authentication, Bitwarden batch item filtering, no-sync export behavior,
provider command timeouts, and cache write avoidance are implemented on the
main branch and released. A warm local Bitwarden measurement improved from
7182.4 ms with four provider processes to 3112.9 ms with one process, an
approximately 56.7% reduction.

See [CHANGELOG.md](CHANGELOG.md) and [docs/performance.md](docs/performance.md).

---

## 1. Why this roadmap exists

`bwenv` started with a focused idea: make secrets from password managers available as environment variables with a pleasant developer experience.

The project has since grown into a real cross-platform Go CLI with:

- Bitwarden support;
- 1Password support;
- interactive CLI/TUI flows;
- `.envrc` generation;
- `direnv` integration;
- shell integration;
- configuration and status flows;
- packaging for multiple operating systems and package managers;
- automated releases.

That growth is positive, but it also means that some early design decisions are now doing more work than they should.

The most visible symptom is performance. Entering a project directory can feel slow because `direnv` triggers `.envrc`, which starts `bwenv export`, which can in turn start several password-manager CLI processes before the environment is ready.

The long-term objective is therefore **not simply to replace `direnv`**.

The objective is to redesign the hot path so that:

1. entering a project is fast;
2. password-manager calls are minimized;
3. secrets are never unnecessarily persisted;
4. `direnv` becomes one activation backend rather than a hard architectural dependency;
5. providers can be added without touching unrelated code;
6. performance regressions become measurable in CI;
7. major refactors are delivered through small, reversible releases.

---

# 2. Current architecture and known pressure points

The current repository already has a reasonable starting separation:

```text
bwenv/
├── .github/
├── assets/
├── internal/
│   ├── config/
│   ├── envrc/
│   ├── provider/
│   └── ui/
├── packaging/
├── main.go
├── go.mod
├── Makefile
├── README.md
└── INSTALL.md
```

However, several files and packages now carry too many responsibilities.

Two notable examples are:

```text
main.go
internal/envrc/envrc.go
```

`main.go` contains a substantial amount of command routing and CLI behavior.

`internal/envrc/envrc.go` currently owns several different concerns:

- `.envrc` generation;
- project metadata parsing;
- `direnv allow`;
- `direnv deny`;
- secret export orchestration;
- interactive export;
- Bitwarden session propagation;
- shell RC detection;
- shell wrapper installation;
- environment cleanup;
- variable-name cache management;
- output formatting helpers.

That is a sign that the package has become a general application layer rather than an `.envrc` package.

The roadmap gradually separates those responsibilities instead of rewriting the entire application at once.

---

# 3. Current performance model

The current hot path can roughly look like this:

```text
cd project
   │
   ▼
direnv detects .envrc
   │
   ▼
.envrc
   │
   ▼
bwenv export
   │
   ├── provider lookup
   ├── provider CLI detection
   ├── session validation
   │     └── bw list folders
   │
   ├── authentication/session reuse
   │     └── session can be validated again
   │
   ├── folder lookup
   │     └── bw list folders
   │
   └── secret retrieval
         ├── bw list items
         └── OR multiple bw get item calls
```

The exact number of subprocesses depends on the selected provider and mode, but the important architectural fact is:

> A directory change can trigger multiple external CLI invocations and potentially multiple vault operations.

That is the real performance problem.

`direnv` is the trigger, but it is not necessarily the bottleneck.

The first performance milestone should therefore reduce the hot path to:

```text
cd project
   │
   ▼
direnv
   │
   ▼
bwenv export
   │
   ▼
one provider fetch
   │
   ▼
exports
```

Later releases can reduce even that provider fetch through a secure memory cache.

---

# 4. Roadmap principles

All releases in this roadmap should follow these rules.

## 4.1 Measure before optimizing

No performance change should be merged only because it “feels faster”.

Every important optimization should have:

- before measurement;
- after measurement;
- provider process count;
- regression test where possible.

Absolute latency can vary between machines and password-manager CLIs, so CI should primarily enforce structural performance properties such as:

```text
provider subprocesses per warm export <= 1
```

instead of fragile requirements such as:

```text
export must always finish under 200 ms
```

Local benchmark reports may still track real latency.

---

## 4.2 Keep the hot path small

Commands executed implicitly by shell hooks must do less work than commands explicitly started by the user.

For example:

```text
bwenv export
```

should be optimized for speed and non-interactive execution.

Commands such as:

```text
bwenv login
bwenv refresh
bwenv init
```

may perform network synchronization, authentication and richer validation because the user explicitly requested them.

---

## 4.3 No silent security tradeoffs for speed

Performance improvements must not introduce plaintext secret caches on disk.

Allowed long-term caching directions:

- in-memory process cache;
- dedicated local agent;
- operating-system secure credential storage where appropriate;
- metadata caches that do not contain secret values.

Avoid:

```text
.bwenv-cache.env
DATABASE_PASSWORD=...
API_KEY=...
```

---

## 4.4 Backward compatibility first, cleanup second

Existing `.envrc` files should continue working through the v2.x line.

When project configuration changes in v3, migration should be explicit and automated:

```bash
bwenv migrate
```

or:

```bash
bwenv init --upgrade
```

The CLI should be able to detect old project formats and explain what to do.

---

## 4.5 One major architectural change per release line

Do not combine all of the following in one release:

- provider redesign;
- shell hook redesign;
- new project config;
- agent process;
- package reorganization;
- cache rewrite.

The releases below intentionally stage those changes.

---

# 5. Release map

The proposed release sequence is:

| Release | Main objective | Risk |
|---|---|---:|
| **v2.2.0** | Benchmarking, diagnostics, CI guardrails | Low |
| **v2.3.0** | Fast export path and fewer provider calls | Medium |
| **v2.3.x** | Stabilization of performance changes | Low |
| **v2.4.0** | Migration tooling, doctor, project metadata cleanup | Medium |
| **v3.0.0** | Internal architecture redesign | High |
| **v3.1.0** | Activation abstraction; direnv becomes optional backend | Medium |
| **v3.2.0** | Optional secure local agent / warm cache | High |
| **v3.3.0** | Security hardening and release provenance | Medium |
| **v3.4.0+** | Provider ecosystem, advanced DX, future integrations | Variable |

The exact version numbers may change, but the dependency order should stay roughly the same.

---

# 6. Phase 0 — Measure and protect the current behavior

## Target release: v2.2.0

### Goal

Before changing the architecture, create a trustworthy baseline for:

- startup time;
- provider operations;
- current CLI behavior;
- generated `.envrc`;
- cross-platform builds;
- security-sensitive output.

This release should introduce very little user-facing behavior change.

Its purpose is to make all later refactors safer.

---

## 6.1 Add performance diagnostics

Introduce an internal diagnostics package:

```text
internal/diagnostics/
├── timing.go
├── counters.go
└── report.go
```

Possible API:

```go
timer := diagnostics.Start("provider.fetch")
defer timer.Stop()
```

or:

```go
ctx := diagnostics.WithRecorder(ctx, recorder)
```

The implementation should be effectively free when diagnostics are disabled.

### CLI

Add:

```bash
bwenv benchmark
```

and optionally:

```bash
bwenv export --debug-timing
```

Example output:

```text
bwenv benchmark

startup              2.8 ms
config                0.7 ms
session check       325.0 ms
folder resolution   291.4 ms
secret fetch        448.7 ms
render                1.1 ms
--------------------------------
total               1069.7 ms

provider subprocesses: 4
```

Secret values must never appear in timing/debug output.

---

## 6.2 Count external provider processes

Create a common command execution abstraction.

Instead of calling `exec.Command` directly throughout provider implementations, introduce something similar to:

```text
internal/process/
├── runner.go
└── exec_runner.go
```

Interface:

```go
type Runner interface {
    Run(ctx context.Context, name string, args ...string) ([]byte, error)
}
```

Provider clients receive a runner dependency.

Benefits:

- tests can use a fake runner;
- subprocess calls can be counted;
- timeouts become centralized;
- command output redaction becomes easier;
- providers are easier to test;
- later performance work can assert exact call counts.

This change should be kept small and should not yet redesign the full provider interface.

---

## 6.3 Build fake provider CLIs for integration tests

Create test fixtures such as:

```text
tests/
└── fixtures/
    ├── fake-bw/
    └── fake-op/
```

The fake CLI should be able to:

- simulate a valid session;
- simulate an expired session;
- return folders;
- return items;
- count invocations;
- inject latency;
- return malformed JSON;
- return provider errors.

This becomes one of the most valuable pieces of infrastructure in the repository.

Example test:

```go
func TestExportCurrentCallCount(t *testing.T) {
    // establish the baseline before optimization
}
```

Later:

```go
func TestWarmExportUsesAtMostOneBitwardenFetch(t *testing.T)
```

---

## 6.4 Introduce real CI, not only release automation

The project currently has release automation, but the development path should also have a dedicated CI workflow.

Create:

```text
.github/workflows/ci.yml
```

Suggested jobs:

### Build matrix

```text
ubuntu-latest
macos-latest
windows-latest
```

Run:

```bash
go build ./...
go test ./...
go vet ./...
```

### Linux quality job

Run heavier checks once:

```bash
go test -race ./...
golangci-lint run
govulncheck ./...
```

### Formatting

Fail CI when:

```bash
gofmt -l .
```

returns tracked Go files.

---

## 6.5 Establish performance baseline document

Create:

```text
docs/performance.md
```

Record:

- test machine;
- OS;
- CPU architecture;
- provider CLI version;
- number of secrets;
- number of selected items;
- cold load;
- warm load;
- subprocess count.

Example:

```text
Scenario: Bitwarden / 5 selected items
Platform: macOS arm64

v2.1.0
cold:  1.42 s
warm:  1.05 s
bw processes: 8

v2.3.0
cold:  0.61 s
warm:  0.24 s
bw processes: 1
```

Real numbers must be collected during implementation; the values above are only the desired report format.

---

## 6.6 Acceptance criteria for v2.2.0

v2.2.0 is complete when:

- [x] `bwenv benchmark` exists;
- [x] provider process execution can be instrumented;
- [x] unit tests still pass;
- [x] fake provider integration infrastructure exists;
- [x] CI runs on Linux, macOS and Windows;
- [x] formatting, vet and vulnerability checks exist;
- [x] benchmark output cannot reveal secret values in tested paths;
- [x] current structural behavior is documented before optimization;
- [ ] real cold and warm latency baseline is recorded with a vault.

---

# 7. Phase 1 — Fast export path

## Target release: v2.3.0

This is the highest-priority functional release.

### Goal

Make normal project activation substantially faster **without changing the user's workflow**.

A user should still be able to:

```bash
cd project
```

and get the environment automatically.

The internal work required to do that should be dramatically reduced.

---

# 7.1 Persist and use FolderID

The current project model already knows a provider-specific folder ID during initialization.

The runtime path should use it.

Current conceptual command:

```bash
bwenv export \
  --provider bitwarden \
  --folder "Production"
```

New generated command:

```bash
bwenv export \
  --provider bitwarden \
  --folder-id "provider-folder-id" \
  --folder "Production"
```

`--folder` remains useful for:

- display;
- diagnostics;
- backward compatibility.

`--folder-id` becomes the primary lookup key.

### Why

Folder names are user-facing metadata.

Folder IDs are machine identifiers.

If the project already knows the folder ID, `bwenv export` should not list every folder on every activation just to rediscover it.

---

## 7.2 Backward compatibility

Existing v2.1 `.envrc` files will not contain `--folder-id`.

Therefore:

```text
if folder-id exists:
    use fast path
else:
    use legacy folder-name resolution
```

The legacy path should emit no scary warning.

Optionally:

```bash
bwenv doctor
```

may later recommend regenerating the project configuration.

---

# 7.3 Remove redundant Bitwarden session preflight

A non-interactive export does not need to prove that a Bitwarden session works multiple times before attempting the actual operation.

Instead of:

```text
IsAuthenticated()
↓
bw list folders

Authenticate()
↓
session validation

ListFolders()
↓
bw list folders

FetchSecrets()
```

the fast path should behave closer to:

```text
BW_SESSION present?
   │
   ├── no → authentication-required error
   │
   └── yes
         │
         ▼
actual requested vault operation
         │
         ├── success → continue
         └── auth error → tell user to run bwenv login
```

The actual provider operation itself becomes the validation.

This is a classic optimistic fast path.

---

## 7.4 Separate interactive and non-interactive authentication semantics

The distinction should become explicit.

### Interactive commands

Examples:

```text
bwenv login
bwenv init
bwenv refresh
```

May:

- prompt;
- synchronize the vault;
- validate sessions;
- repair configuration.

### Implicit/hot-path commands

Example:

```text
bwenv export
```

Must:

- never prompt;
- never perform unnecessary sync;
- avoid validation round-trips;
- fail fast if a session is invalid.

This rule should later become part of the provider contract.

---

# 7.5 Replace N item fetches with one list operation

When selected item IDs are configured, avoid:

```text
bw get item A
bw get item B
bw get item C
bw get item D
```

Prefer:

```text
bw list items --folderid FOLDER
```

once.

Then filter in Go:

```go
selected := map[string]struct{}{
    "A": {},
    "B": {},
    "C": {},
    "D": {},
}

for _, item := range items {
    if _, ok := selected[item.ID]; ok {
        // collect fields
    }
}
```

Complexity changes from:

```text
N external processes
```

to:

```text
1 external process + O(number of returned items) local filtering
```

For small and medium vault folders, local Go filtering is extremely cheap compared with spawning external CLIs.

---

# 7.6 Avoid automatic vault sync in the hot path

`bw sync` should not be a routine part of every directory activation.

Recommended semantics:

```text
bwenv login
    → authentication
    → optional/required sync

bwenv refresh
    → explicit sync
    → invalidate caches
    → reload

bwenv export
    → no sync
```

If automatic sync remains a user option, it should be redesigned around a stale interval:

```text
sync only when last sync > configured TTL
```

and should preferably happen outside the latency-sensitive shell activation path.

---

# 7.7 Reduce unnecessary disk writes

The `.bwenv_vars` cache is useful for knowing which variables must later be unset.

However, avoid rewriting it when nothing changed.

Possible approach:

```text
new variable-name set
      │
      ▼
compare with current cache
      │
   changed?
   /     \
 no       yes
 │         │
skip     atomic write
```

Use atomic replacement:

```text
write temporary file
fsync if appropriate
rename
```

This is both safer and avoids needless filesystem churn.

---

# 7.8 Add context and timeouts to provider commands

All external CLI commands should receive a `context.Context`.

Example:

```go
ctx, cancel := context.WithTimeout(parent, 5*time.Second)
defer cancel()
```

Timeouts should differ by operation:

- local status check: short;
- secret fetch: moderate;
- interactive login/sync: longer.

Do not allow an implicit shell activation to hang indefinitely.

---

# 7.9 Performance acceptance criteria for v2.3.0

Structural criteria:

- [x] generated project config contains FolderID;
- [x] old `.envrc` files still work;
- [x] warm Bitwarden export does not list folders when FolderID is available;
- [x] warm Bitwarden export performs at most one primary vault fetch;
- [x] selected items no longer require one process per item;
- [x] `bwenv export` never runs `bw sync`;
- [x] `bwenv export` never prompts;
- [x] provider errors are translated into useful bwenv errors;
- [x] subprocess-count tests exist;
- [x] benchmark comparison is added to `docs/performance.md`.

Performance target:

> On the reference development machine, median warm export should improve by at least 50–60% compared with the v2.1 baseline.

The exact wall-clock requirement should not be enforced in CI because provider CLI and machine performance vary.

The subprocess-count requirement **should** be enforced.

Verified local result for v2.3.0:

- v2.2.0 legacy path: 7182.4 ms, four provider processes;
- v2.3.0 FolderID path: 3112.9 ms, one provider process;
- observed improvement: approximately 56.7%.

---

# 8. v2.3.x — stabilization releases

Do not immediately start the v3 refactor after shipping the fast path.

Reserve one or more patch releases for real-world issues.

Focus areas:

- different Bitwarden CLI versions;
- large folders;
- duplicate variable names;
- expired sessions;
- 1Password behavior;
- macOS shell differences;
- Linux shells;
- Windows packaging;
- fish shell quoting;
- spaces and special characters in folder names.

Possible releases:

```text
v2.3.1
v2.3.2
```

No major architecture changes should enter these patches.

---

# 9. Phase 1.5 — migration and diagnostics

## Target release: v2.4.0

### Goal

Prepare the repository and users for the v3 architecture without requiring the full v3 migration yet.

---

## 9.1 Introduce `bwenv doctor`

Add:

```bash
bwenv doctor
```

Example:

```text
bwenv doctor

System
✓ bwenv 2.4.0
✓ darwin/arm64
✓ zsh detected

Provider
✓ Bitwarden CLI installed
✓ session available
✓ vault reachable

Project
✓ bwenv configuration detected
✓ folder ID configured
✓ 6 selected variables

Activation
✓ direnv installed
✓ shell hook detected
✓ .envrc allowed

Performance
✓ fast path available
✓ expected provider calls: 1

Security
✓ project files have restrictive permissions
✓ no plaintext secret cache detected
```

`doctor` should not print:

- session tokens;
- secret values;
- full provider payloads.

---

## 9.2 Add `bwenv refresh`

The command should have clear semantics:

```bash
bwenv refresh
```

Possible behavior:

1. sync provider if supported;
2. re-fetch secrets;
3. refresh runtime cache if one exists;
4. update environment through shell integration where possible.

This command becomes the explicit alternative to automatic synchronization during every `cd`.

---

## 9.3 Define canonical project metadata format

Prepare a dedicated project config that can eventually replace metadata embedded in `.envrc`.

Recommended direction:

```text
.bwenv.toml
```

Example:

```toml
version = 1
provider = "bitwarden"

[project]
folder_id = "abc123"
folder_name = "Production"

items = [
  "item-id-a",
  "item-id-b"
]

[activation]
mode = "direnv"
```

Important:

> `.bwenv.toml` must contain references and configuration, not secret values.

For v2.4 it may be optional/experimental.

When migrating an older `.envrc` that has no stable folder ID, `folder_id` may be omitted and
export must retain the existing folder-name lookup fallback.

For v3 it can become canonical.

---

## 9.4 Migration command

Introduce:

```bash
bwenv migrate
```

Responsibilities:

- detect legacy `.envrc`;
- parse provider/folder/items;
- create `.bwenv.toml`;
- regenerate a thin `.envrc`;
- preserve behavior;
- create a restrictive backup before replacing `.envrc`.

Support:

```bash
bwenv migrate --dry-run
```

Output example:

```text
Legacy project detected.

Planned changes:
  + create .bwenv.toml
  ~ regenerate .envrc
  = retain BW_SESSION in .envrc for v2 compatibility

No secret values will be written to .bwenv.toml.

Run:
  bwenv migrate

or preview:
  bwenv migrate --dry-run
```

---

# 10. Phase 2 — architecture redesign

## Target release: v3.0.0

v3.0 should be primarily an **internal architecture release**.

It should not attempt to simultaneously introduce the agent and replace `direnv`.

The major-version bump gives freedom to clean up project configuration and CLI internals while preserving recognizable user workflows.

---

# 10.1 Target repository structure

Proposed structure:

```text
bwenv/
├── cmd/
│   └── bwenv/
│       └── main.go
│
├── internal/
│   ├── app/
│   │   ├── app.go
│   │   ├── init.go
│   │   ├── export.go
│   │   ├── login.go
│   │   ├── refresh.go
│   │   ├── status.go
│   │   └── migrate.go
│   │
│   ├── cli/
│   │   ├── root.go
│   │   ├── init.go
│   │   ├── export.go
│   │   ├── login.go
│   │   ├── refresh.go
│   │   ├── config.go
│   │   ├── status.go
│   │   ├── doctor.go
│   │   └── benchmark.go
│   │
│   ├── provider/
│   │   ├── provider.go
│   │   ├── registry.go
│   │   ├── bitwarden/
│   │   │   ├── provider.go
│   │   │   ├── client.go
│   │   │   ├── model.go
│   │   │   └── errors.go
│   │   └── onepassword/
│   │       ├── provider.go
│   │       ├── client.go
│   │       ├── model.go
│   │       └── errors.go
│   │
│   ├── activation/
│   │   ├── activator.go
│   │   └── direnv/
│   │       ├── activator.go
│   │       ├── generator.go
│   │       └── detector.go
│   │
│   ├── project/
│   │   ├── config.go
│   │   ├── loader.go
│   │   ├── migration.go
│   │   └── validation.go
│   │
│   ├── session/
│   │   ├── manager.go
│   │   └── state.go
│   │
│   ├── shell/
│   │   ├── detect.go
│   │   ├── quote.go
│   │   └── wrapper.go
│   │
│   ├── process/
│   │   ├── runner.go
│   │   └── exec.go
│   │
│   ├── diagnostics/
│   ├── config/
│   └── ui/
│
├── tests/
│   ├── integration/
│   ├── e2e/
│   └── fixtures/
│
├── docs/
│   ├── architecture.md
│   ├── performance.md
│   ├── security.md
│   └── providers.md
│
├── packaging/
├── .github/
├── go.mod
├── Makefile
├── README.md
├── CONTRIBUTING.md
├── SECURITY.md
├── CHANGELOG.md
└── ROADMAP.md
```

---

# 10.2 Make `main.go` boring

Desired `cmd/bwenv/main.go`:

```go
func main() {
    os.Exit(cli.Execute())
}
```

or equivalent.

`main.go` should not know about:

- Bitwarden;
- `.envrc`;
- TUI screens;
- shell wrappers;
- folder parsing;
- provider-specific behavior.

That logic belongs below the CLI layer.

---

# 10.3 Application layer

The application layer coordinates use cases.

Example:

```go
type ExportService struct {
    Providers ProviderRegistry
    Projects  ProjectStore
    Sessions  SessionManager
}
```

Responsibilities:

```text
CLI request
    ↓
application service
    ↓
project config
provider
session
activation
```

The provider should not decide how the CLI looks.

The UI should not implement secret retrieval.

The activation backend should not perform authentication.

---

# 10.4 Redesign provider boundaries

Avoid one oversized provider interface that knows every possible provider feature.

Prefer a compact core contract plus optional capabilities.

Conceptually:

```go
type Provider interface {
    Name() string
    Slug() string
    Available(context.Context) error
    Fetch(context.Context, FetchRequest) ([]Secret, error)
}
```

Optional interfaces:

```go
type Authenticator interface {
    Login(context.Context) (Session, error)
}

type Synchronizer interface {
    Sync(context.Context) error
}

type FolderLister interface {
    ListFolders(context.Context, Session) ([]Folder, error)
}
```

Why:

- 1Password and Bitwarden do not need identical authentication behavior;
- future providers may not have folders;
- some providers may use service accounts;
- some providers may not support sync;
- testing becomes simpler.

---

# 10.5 Introduce typed provider errors

Avoid parsing human-readable strings everywhere.

Create typed errors such as:

```go
ErrNotAuthenticated
ErrSessionExpired
ErrProviderUnavailable
ErrFolderNotFound
ErrItemNotFound
ErrMalformedProviderResponse
ErrProviderTimeout
```

The application layer maps these to user-friendly messages.

Example:

```text
provider.ErrSessionExpired
        ↓
app
        ↓
"Bitwarden session expired. Run `bwenv login`."
```

---

# 10.6 Canonical project config

In v3:

```text
.bwenv.toml
```

becomes the project definition.

Example:

```toml
version = 1
provider = "bitwarden"

[secret_source]
folder_id = "abc"
folder_name = "Production"
items = ["id1", "id2"]

[activation]
mode = "direnv"
```

The generated `.envrc` should become intentionally boring.

Example:

```bash
# generated by bwenv
eval "$(bwenv export --project .)"
```

No provider-specific data needs to be duplicated inside `.envrc`.

---

# 10.7 Remove sessions from project files

v3 should stop writing:

```bash
export BW_SESSION=...
```

into project `.envrc`.

Sessions belong to runtime state, not project configuration.

Possible v3.0 runtime model:

```text
current shell environment
        +
explicit bwenv login
```

The later v3.2 agent improves this further.

---

# 10.8 Split `internal/envrc`

The old package should disappear or become a tiny direnv-specific package.

Responsibilities move to:

```text
project/
activation/direnv/
shell/
app/
session/
```

This is one of the key architectural milestones.

---

# 10.9 v3.0 acceptance criteria

- [ ] root `main.go` is replaced by `cmd/bwenv/main.go`;
- [ ] CLI routing is separated from application logic;
- [ ] provider implementations are isolated in subpackages;
- [ ] external process execution is dependency-injected;
- [ ] project config is canonical and versioned;
- [ ] legacy config migration is supported;
- [ ] `.envrc` no longer stores provider session tokens;
- [ ] direnv logic is isolated behind an activation package;
- [ ] typed errors exist;
- [ ] all external operations support context cancellation/timeouts;
- [ ] CI passes on supported platforms;
- [ ] v2 migration documentation exists.

---

# 11. Phase 3 — Activation abstraction

## Target release: v3.1.0

### Goal

Make `direnv` the default integration, not the identity of the project.

Target architecture:

```text
                 ┌────────────────┐
                 │ password vault │
                 └───────┬────────┘
                         │
                         ▼
                   ┌───────────┐
                   │   bwenv   │
                   └─────┬─────┘
                         │
                activation layer
              ┌──────────┼──────────┐
              │          │          │
              ▼          ▼          ▼
           direnv    native shell   mise
```

---

# 11.1 Activator interface

Possible model:

```go
type Activator interface {
    Name() string
    Detect(context.Context) Status
    Install(context.Context, Project) error
    Remove(context.Context, Project) error
    Render(context.Context, Project) ([]byte, error)
}
```

Initial implementation:

```text
activation/direnv
```

Later:

```text
activation/shell
activation/mise
```

---

# 11.2 Explicit activation mode

Project config:

```toml
[activation]
mode = "direnv"
```

CLI:

```bash
bwenv init --activation direnv
```

Later:

```bash
bwenv init --activation shell
bwenv init --activation mise
```

---

# 11.3 Native shell experiment

Add:

```bash
bwenv hook zsh
bwenv hook bash
bwenv hook fish
```

and:

```bash
bwenv activate
```

The purpose is not immediately to replace direnv.

The purpose is to prove that bwenv's secret retrieval and project model work independently from direnv.

Possible shell setup:

```bash
eval "$(bwenv hook zsh)"
```

The hook may detect directory changes and ask bwenv whether the current project needs activation.

This must remain experimental until:

- performance is known;
- nested directories work;
- deactivation works;
- shell behavior is stable;
- security properties are understood.

---

# 11.4 Optional mise integration

If useful, implement a small adapter for users already using mise.

Do not move provider logic into mise.

The integration should remain:

```text
mise
  ↓
bwenv
  ↓
provider
```

not:

```text
mise-specific Bitwarden implementation
```

One source of secret retrieval logic should remain inside bwenv.

---

# 11.5 v3.1 acceptance criteria

- [ ] direnv behavior is implemented through `Activator`;
- [ ] project config explicitly identifies activation mode;
- [ ] core secret retrieval works without importing direnv-specific packages;
- [ ] `bwenv activate` exists;
- [ ] `bwenv hook` can be tested independently;
- [ ] direnv remains the stable default;
- [ ] alternative activators are marked experimental until proven.

---

# 12. Phase 4 — Secure warm cache / bwenv agent

## Target release: v3.2.0

This is a high-value feature, but it should come only after the hot path and architecture are clean.

### Goal

Make repeated project activation close to instant without writing plaintext secrets to disk.

---

# 12.1 Why a local agent

A normal CLI process exits after every command.

Therefore an in-process cache such as:

```go
var cache map[string]string
```

does not help the next `cd`.

A persistent local process can.

Architecture:

```text
shell / direnv
      │
      ▼
    bwenv
      │
      │ local IPC
      ▼
┌─────────────────┐
│   bwenv-agent   │
│                 │
│ session cache   │
│ secret cache    │
│ TTL metadata    │
└────────┬────────┘
         │
         ▼
 password manager
```

---

# 12.2 IPC

Unix-like systems:

```text
Unix domain socket
```

Windows:

```text
Named Pipe
```

The transport must be local-only.

The agent must validate that requests originate from the expected local user where the OS permits it.

---

# 12.3 Cache rules

Cache key should include at least:

```text
provider
account/profile if applicable
folder ID
selected item IDs
project config version/hash
```

Cache entry:

```text
secret values
created_at
expires_at
provider metadata
```

---

# 12.4 TTL

Configurable defaults:

```text
5m
15m
30m
until lock
```

Recommended default should favor safety over maximum persistence.

CLI:

```bash
bwenv config set cache.ttl 15m
```

or equivalent.

---

# 12.5 Invalidation

Invalidate when:

- project config changes;
- user runs `bwenv refresh`;
- user runs `bwenv lock`;
- provider session changes;
- TTL expires;
- provider indicates auth failure;
- agent restarts.

---

# 12.6 Agent commands

Possible commands:

```bash
bwenv agent status
bwenv agent start
bwenv agent stop
bwenv agent restart
bwenv lock
```

`bwenv lock` should:

1. purge cached secret values;
2. invalidate cached sessions;
3. ask provider to lock where supported;
4. leave no secret values in debug output.

---

# 12.7 No plaintext disk cache

The agent may persist non-sensitive metadata such as:

```text
agent PID
socket path
last refresh timestamp
project hash
```

It should not persist decrypted secret values in a regular file.

If future releases support persistent secure storage, it must use an operating-system secure facility and be a separate design decision.

---

# 12.8 Fast path with agent

Target:

```text
cd project
  ↓
direnv / shell hook
  ↓
bwenv export
  ↓
local agent
  ↓
cache hit
  ↓
exports
```

No password-manager process on a cache hit.

This is the point where repeated activation can realistically become nearly instant.

---

# 12.9 v3.2 acceptance criteria

- [ ] agent is optional;
- [ ] bwenv works without the agent;
- [ ] local IPC is authenticated/restricted;
- [ ] secrets are cached only in memory;
- [ ] TTL is enforced;
- [ ] `bwenv lock` reliably purges cache;
- [ ] config changes invalidate cache;
- [ ] agent crashes fail safely;
- [ ] cache hit performs zero provider CLI calls;
- [ ] integration tests cover restart, expiry and invalidation.

---

# 13. Phase 5 — Security hardening

## Target release: v3.3.0

Security improvements should happen throughout the roadmap, but this release performs a focused audit.

---

# 13.1 Threat model document

Create:

```text
docs/security.md
```

Document what bwenv protects against and what it does not.

Threats to discuss:

- secrets accidentally committed to Git;
- secret values printed to logs;
- world-readable project files;
- shell history exposure;
- malicious project `.envrc`;
- local IPC abuse;
- stale sessions;
- command injection through variable names or folder names;
- CI log leakage;
- crash dumps;
- temporary files.

---

# 13.2 No secret values in logs

Create a rule:

> Secret values must never be included in application errors.

Errors may contain:

```text
variable name
provider
folder display name
item ID
command name
exit code
```

Errors must not contain:

```text
secret value
session token
raw provider payload when it may include secrets
```

---

# 13.3 Output channel contract

For commands used by `eval`:

```text
stdout = executable shell export/unset code only
stderr = human-readable diagnostics
```

Add tests that guarantee this.

A debug log must never accidentally enter stdout.

---

# 13.4 Validate environment variable names

Continue or strengthen key sanitization.

Reject or safely transform values that could generate invalid or dangerous shell expressions.

Tests should include:

```text
NORMAL_KEY
key-with-dash
"quoted key"
$(command)
KEY;rm
multiline input
unicode
```

---

# 13.5 Atomic and restrictive file handling

Sensitive/runtime files:

```text
0600 on Unix where appropriate
```

Use atomic writes where configuration corruption would be harmful.

Check Windows ACL behavior separately instead of pretending Unix modes provide the same guarantees.

---

# 13.6 Supply-chain security

Add:

- dependency vulnerability scanning;
- release checksums;
- SBOM generation;
- signed/provenance-aware releases where practical;
- pinned GitHub Action versions;
- minimal workflow permissions.

Potential release assets:

```text
checksums.txt
sbom.spdx.json
provenance metadata
```

---

# 13.7 SECURITY.md

Add instructions for responsible vulnerability reporting.

Do not ask users to file public issues for sensitive security reports.

---

# 14. Phase 6 — GitHub and OSS maturity

This work begins in v2.2 and continues through v3.

Target repository metadata:

```text
.github/
├── ISSUE_TEMPLATE/
│   ├── bug.yml
│   ├── feature.yml
│   └── config.yml
├── workflows/
│   ├── ci.yml
│   ├── security.yml
│   └── release.yml
├── dependabot.yml
└── pull_request_template.md

CONTRIBUTING.md
SECURITY.md
CHANGELOG.md
ROADMAP.md
```

---

# 14.1 Pull request policy

Every non-trivial PR should state:

```text
Problem
Solution
Behavior change
Performance impact
Security impact
Tests
Migration impact
```

Performance-sensitive PRs additionally include:

```text
Before
After
Provider subprocess count
```

---

# 14.2 Release checklist

Before every release:

- [ ] tests pass;
- [ ] supported OS builds pass;
- [ ] race tests pass where supported;
- [ ] `go vet` passes;
- [ ] linter passes;
- [ ] vulnerability scan passes or findings are reviewed;
- [ ] changelog updated;
- [ ] migration notes written;
- [ ] README commands verified;
- [ ] package manager metadata updated;
- [ ] generated artifacts use the correct version;
- [ ] benchmark report updated for performance releases.

---

# 14.3 Branch strategy

Keep it simple:

```text
main
feature/*
fix/*
refactor/*
```

Use short-lived branches.

Avoid a permanent `develop` branch unless release management genuinely requires it.

`main` should remain releasable.

---

# 15. Phase 7 — Developer experience

Some DX features begin in v2.4 and mature in v3.x.

---

# 15.1 `bwenv status`

Provide a quick view:

```text
Project: Production API
Provider: Bitwarden
Activation: direnv
Session: active
Cache: warm (8m remaining)
Secrets: 7 variables
Last refresh: 2m ago
```

No values.

---

# 15.2 `bwenv doctor`

Detailed diagnostics and repair guidance.

Possible optional:

```bash
bwenv doctor --fix
```

Only safe, reversible repairs should be automatic.

Potential fixes:

- regenerate `.envrc`;
- fix project file permissions;
- install missing shell integration;
- re-run `direnv allow`.

Do not silently modify unrelated user shell configuration.

---

# 15.3 `bwenv benchmark`

Keep benchmark useful beyond v2.2.

Future report:

```text
Activation benchmark

project config      0.2 ms
client startup      1.8 ms
agent IPC           2.4 ms
cache lookup        0.4 ms
shell render        0.8 ms
--------------------------
total               5.6 ms

provider calls:     0
cache:              HIT
```

---

# 15.4 `bwenv refresh`

Explicitly refresh provider and cached state.

Possible flags:

```bash
bwenv refresh
bwenv refresh --force
bwenv refresh --project .
```

---

# 15.5 `bwenv lock`

Single mental model:

```text
lock = remove live secret/session state
```

It should work whether activation uses:

- direnv;
- native shell;
- agent.

---

# 16. Provider extensibility after v3

Once v3 provider boundaries are stable, adding a provider should not require editing:

```text
CLI command router
direnv generator
TUI implementation
main.go
```

A new provider should mostly require:

```text
internal/provider/newprovider/
├── provider.go
├── client.go
├── model.go
├── errors.go
└── provider_test.go
```

plus registry/config metadata.

Documentation:

```text
docs/providers.md
```

should define:

- required interfaces;
- optional capabilities;
- secret field mapping;
- authentication expectations;
- error taxonomy;
- testing contract.

---

# 17. Testing strategy

Testing should be treated as part of architecture, not post-release cleanup.

---

## 17.1 Unit tests

Focus on pure logic:

- shell quoting;
- variable sanitization;
- config parsing;
- migration;
- cache keys;
- provider response parsing;
- error mapping.

---

## 17.2 Provider contract tests

Every provider should pass a common suite where applicable.

Example contract:

```text
Available
Authentication failure
Fetch by source
Selected items
Empty result
Malformed response
Timeout
Lock
```

---

## 17.3 Fake CLI integration tests

Use fake `bw` and `op` executables.

Test real process boundaries without contacting live vaults.

This is where subprocess-count assertions belong.

---

## 17.4 End-to-end tests

Where practical:

```text
bwenv init
→ project config
→ generated activation file
→ export
→ disallow
→ remove
```

Include:

- zsh;
- bash;
- fish;
- Linux;
- macOS.

Windows testing should focus on supported shell/runtime paths rather than forcing Unix assumptions.

---

## 17.5 Performance regression tests

Prefer deterministic assertions:

```text
warm export provider calls == 1
cache hit provider calls == 0
```

Do not fail CI because one hosted runner was 80 ms slower.

Store real timings as informational benchmark artifacts when useful.

---

## 17.6 Security regression tests

Explicit tests should verify:

- session token not written to v3 project config;
- secret values never appear in stderr;
- secret values never appear in benchmark output;
- errors redact provider payloads;
- generated shell output is properly quoted;
- cached values disappear after `bwenv lock`.

---

# 18. Detailed PR sequence

The roadmap should be implemented through incremental PRs.

Suggested order:

---

## PR 001 — Benchmark foundation

Scope:

- diagnostics timer;
- benchmark command;
- no behavior change.

Deliverables:

```text
internal/diagnostics/
bwenv benchmark
docs/performance.md
```

---

## PR 002 — Process runner abstraction

Scope:

- centralize `exec.Command`;
- add fake runner support;
- preserve behavior.

Goal:

Make later subprocess optimizations testable.

---

## PR 003 — CI baseline

Scope:

```text
ci.yml
format check
go test
go vet
race
lint
govulncheck
```

Do this before major runtime refactors.

---

## PR 004 — Fake Bitwarden CLI

Scope:

- deterministic integration fixture;
- invocation counter;
- expired-session scenarios.

---

## PR 005 — FolderID fast path

Scope:

- generated `.envrc` includes folder ID;
- CLI parses it;
- export skips folder listing when available;
- legacy fallback retained.

---

## PR 006 — Authentication fast path

Scope:

- remove duplicate non-interactive validation;
- actual provider fetch becomes validation;
- typed temporary errors if necessary.

---

## PR 007 — Single-call selected-item fetch

Scope:

- one `bw list items`;
- in-memory ID filtering;
- tests assert process count.

---

## PR 008 — Remove sync from export path

Scope:

- `bwenv export` never syncs;
- sync remains in explicit commands;
- docs updated.

---

## PR 009 — Disk-write cleanup

Scope:

- `.bwenv_vars` compare-before-write;
- atomic writes;
- permissions test.

---

## PR 010 — `bwenv doctor`

Scope:

- diagnostics;
- fast-path detection;
- shell/provider/project health.

---

## PR 011 — Project config format

Scope:

- `.bwenv.toml`;
- parser;
- validation;
- no secret values.

Initially optional.

---

## PR 012 — Migration tooling

Scope:

```text
bwenv migrate
bwenv migrate --dry-run
```

---

## PR 013 — Move entrypoint

Scope:

```text
cmd/bwenv/main.go
internal/cli
```

No major behavior changes.

---

## PR 014 — Introduce application services

Scope:

```text
internal/app
```

Move orchestration from CLI and envrc package.

---

## PR 015 — Split providers

Scope:

```text
provider/bitwarden
provider/onepassword
```

Introduce typed errors and context.

---

## PR 016 — Split project / activation / shell responsibilities

Scope:

remove oversized `internal/envrc` responsibility set.

---

## PR 017 — v3 project config default

Scope:

- `.bwenv.toml` canonical;
- thin generated `.envrc`;
- no BW session token persisted.

This is an appropriate v3 release boundary.

---

## PR 018 — Activation interface

Scope:

- direnv adapter;
- no user-visible change initially.

---

## PR 019 — Native shell activation experiment

Scope:

```text
bwenv hook
bwenv activate
```

Experimental.

---

## PR 020 — Agent prototype

Scope:

- local IPC;
- memory cache;
- status/start/stop;
- no automatic enabling.

---

## PR 021 — Agent security hardening

Scope:

- user-scoped socket/pipe;
- TTL;
- lock;
- invalidation;
- crash recovery.

Only after this should the agent be considered stable.

---

# 19. Dependency graph

High-level dependencies:

```text
Benchmarking
     │
     ▼
Process runner
     │
     ├──────────────► CI regression protection
     │
     ▼
Fast provider path
     │
     ▼
v2.3 stabilization
     │
     ▼
Project metadata/migration
     │
     ▼
v3 architecture
     │
     ▼
Activation abstraction
     │
     ▼
Agent / cache
```

Do not implement the agent before the provider hot path is cleaned up.

The agent should be an optimization on top of a good architecture, not a workaround for a slow one.

---

# 20. Migration strategy

## v2.1 → v2.3

No manual migration required.

Existing projects continue using folder-name fallback.

Regenerating with:

```bash
bwenv init
```

enables the FolderID fast path.

---

## v2.x → v3

Recommended path:

```bash
bwenv migrate --dry-run
bwenv migrate
```

Expected migration:

```text
OLD

.envrc
  provider metadata
  folder
  item IDs
  possibly BW_SESSION
  bwenv export command


NEW

.bwenv.toml
  provider metadata
  folder ID
  folder display name
  selected items
  activation mode

.envrc
  tiny generated activation command
  BW_SESSION retained until v3 runtime session management
```

The migration should be reversible until validation succeeds. Although v2.4 moves provider,
folder, and item metadata out of `.envrc`, it must retain `BW_SESSION` to preserve current
authentication behavior. Session removal is deferred until the v3 runtime session manager is
available, as recorded in the risk register.

---

# 21. Risk register

| Risk | Impact | Mitigation |
|---|---|---|
| Performance refactor changes authentication behavior | High | fake provider integration tests + v2.3 patch stabilization |
| Provider CLI output changes between versions | Medium | typed parser tests + defensive error handling |
| Session removed from `.envrc` breaks current UX | High | migrate only in v3 + explicit login/session manager |
| Native shell hook behaves differently across shells | High | keep direnv default until experimental hook matures |
| Agent increases attack surface | High | optional feature, local-only IPC, threat model, memory-only secrets |
| Absolute benchmarks are flaky | Medium | CI asserts process counts, not strict wall-clock latency |
| Large refactor causes merge conflicts | Medium | small PRs and moving responsibilities before rewriting behavior |
| Windows assumptions differ from Unix | Medium | dedicated Windows CI and platform-specific code |
| Debug output leaks secret data | Critical | output-contract tests and redaction layer |
| Config format becomes hard to evolve | Medium | version `.bwenv.toml` from day one |

---

# 22. Success metrics

The project should be considered successfully modernized when the following are true.

## Performance

### v2.3 target

```text
warm provider calls: <= 1
selected-item fetches: 1 provider list operation
folder discovery during warm export: 0
automatic sync during export: 0
```

### v3.2 target with warm agent cache

```text
provider calls on cache hit: 0
```

---

## Architecture

- `main.go` contains no business logic;
- provider code does not import UI code;
- activation code does not implement authentication;
- project config contains no secret values;
- direnv-specific behavior lives inside the direnv adapter;
- external commands use one shared runner abstraction.

---

## Security

- no persistent plaintext secret cache;
- no session token in v3 project files;
- no secret values in logs/errors;
- `bwenv lock` clears runtime cache;
- output used by `eval` is strictly separated from diagnostics;
- release pipeline includes security checks.

---

## Developer experience

A healthy project should eventually feel like:

```bash
cd project
```

and the environment is available almost immediately.

When something is wrong:

```bash
bwenv doctor
```

explains why.

When credentials change:

```bash
bwenv refresh
```

updates them.

When the user wants everything cleared:

```bash
bwenv lock
```

clears runtime state.

That is the target mental model.

---

# 23. What should NOT be done immediately

The following ideas are intentionally postponed:

## Do not replace direnv first

Replacing direnv does not solve repeated provider calls.

Optimize the retrieval path before replacing the trigger.

---

## Do not build the agent first

An agent can hide a slow architecture, but it also adds:

- IPC;
- background process lifecycle;
- cache invalidation;
- new security concerns;
- Windows-specific behavior.

First make one uncached export efficient.

Then cache it.

---

## Do not add many providers before v3 boundaries stabilize

More providers would lock the current interface into place and increase refactor cost.

Stabilize the provider contract first.

---

## Do not persist decrypted `.env` caches for performance

The project is a secret-management tool.

A performance shortcut that leaves decrypted secrets on disk would conflict with its core purpose.

---

# 24. Recommended immediate work

If development starts now, the next milestone should contain only these items:

### Milestone: v2.2.0

1. `bwenv benchmark`;
2. command runner abstraction;
3. fake Bitwarden fixture;
4. CI workflow;
5. initial performance document.

Then:

### Milestone: v2.3.0

1. FolderID hot path;
2. remove repeated auth/folder checks;
3. one Bitwarden list operation for selected items;
4. remove sync from implicit export;
5. reduce disk writes;
6. publish before/after benchmark.

Only after v2.3 is stable should work begin on the v3 architecture.

---

# 25. Long-term product direction

The best long-term identity for bwenv is not:

> “A wrapper around direnv.”

It should become:

> **A fast, provider-agnostic secret activation layer for developer environments.**

Password managers supply the secrets.

bwenv decides:

- how they are resolved;
- how sessions are managed;
- how projects reference them;
- how they are safely activated;
- how they are cached;
- how they are diagnosed.

Activation backends such as direnv, shell hooks or mise become replaceable adapters.

That architecture leaves room for future capabilities without turning the codebase into a collection of shell-specific special cases.

---

# 26. Release summary

## v2.2.0 — Measure

**Theme:** visibility before optimization.

Deliver:

- benchmark tooling;
- process instrumentation;
- fake provider tests;
- CI;
- performance baseline.

---

## v2.3.0 — Accelerate

**Theme:** make normal activation fast.

Deliver:

- FolderID fast path;
- single provider fetch;
- remove redundant validation;
- remove automatic sync from export;
- reduce disk writes.

---

## v2.4.0 — Prepare

**Theme:** migration and diagnostics.

Deliver:

- `bwenv doctor`;
- `bwenv refresh`;
- optional `.bwenv.toml`;
- `bwenv migrate`;
- v3 migration path.

---

## v3.0.0 — Restructure

**Theme:** clean internal architecture.

Deliver:

- `cmd/bwenv`;
- application layer;
- provider subpackages;
- project package;
- session package;
- typed errors;
- canonical `.bwenv.toml`;
- thin `.envrc`;
- no session token in project config.

---

## v3.1.0 — Decouple

**Theme:** bwenv no longer depends conceptually on direnv.

Deliver:

- activation interface;
- direnv adapter;
- `bwenv activate`;
- experimental shell hook;
- optional integration adapters.

---

## v3.2.0 — Instant

**Theme:** secure warm activation.

Deliver:

- optional local agent;
- memory-only secret cache;
- TTL;
- IPC;
- zero provider calls on cache hit;
- `bwenv lock`.

---

## v3.3.0 — Harden

**Theme:** dedicated security pass.

Deliver:

- threat model;
- output/redaction guarantees;
- release provenance;
- SBOM;
- security workflow;
- stricter file handling.

---

## v3.4.0+ — Expand

**Theme:** ecosystem and advanced DX.

Possible work:

- additional providers;
- provider SDK/documentation;
- mature native shell activation;
- mise integration;
- richer diagnostics;
- optional secure OS credential-store integrations.

---

# 27. Final engineering rule

When deciding whether a new feature belongs in the activation path, ask:

> **Does this operation need to happen every time the user enters a directory?**

If the answer is no, it should probably not be on the hot path.

That single rule should keep bwenv fast as the project grows.
