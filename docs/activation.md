# Shell activation: native, direnv and mise

[Documentation home](../README.md) · [Getting started](getting-started.md) · [Configuration](configuration.md)

An activation backend loads variables as you enter a project and removes/restores
them when you leave. Provider authentication is separate: the hook uses an
existing session and does not ask for a password on every directory change.

## Choose once

```sh
bwenv config
```

Select **Default Activation Hook**, Enter to cycle, **S** to save. The default is
`shell`. The preference applies only to new projects; `.bwenv.toml` selects the
backend for each existing project.

| Backend | External CLI | One-time shell setup | Project approval |
| --- | --- | --- | --- |
| `shell` (experimental) | None | bwenv native hook + wrapper | `bwenv allow` / `bwenv disallow` |
| `direnv` | `direnv` | direnv hook + bwenv wrapper | direnv's local allow/deny |
| `mise` (experimental) | `mise` | mise activation + bwenv wrapper | `mise trust`, plus bwenv enable/disable |

To override one project, run `bwenv init --activation shell`, `direnv`, or `mise`.
Setup reports missing backend dependencies and prints the next steps.

## Native shell: default

`bwenv init` installs the native hook and wrapper in the detected shell RC.
Source the displayed file or open a new terminal, then run `bwenv login` in the project.

For manual **native hook** setup, use the matching command once in your RC file:

| Shell | Hook command |
| --- | --- |
| Zsh | `eval "$(bwenv hook zsh)"` |
| Bash | `eval "$(bwenv hook bash)"` |
| Fish | `bwenv hook fish \| source` |

The `hook` command prints executable shell integration only, so its output can
be evaluated. Do not evaluate `help`, `version`, `status` or `doctor` output.
The hook command alone is not a substitute for init's command wrapper; init
installs the wrapper that makes plain `bwenv login` update the current shell.

Native activation discovers the nearest `.bwenv.toml` from the current directory
or its parents. It restores the previous project's values before switching.
Subdirectories and repeated prompt hooks do not refetch unchanged selections.
A project configuration change can trigger reactivation without leaving the directory.
Use `bwenv refresh` to fetch changed provider values explicitly.

## Direnv

Install direnv, then add its hook to your RC file **once**:

| Shell | Command |
| --- | --- |
| Zsh | `eval "$(direnv hook zsh)"` |
| Bash | `eval "$(direnv hook bash)"` |
| Fish | `direnv hook fish \| source` |

```sh
bwenv init --activation direnv
```

Source the RC file printed by init or open a new terminal. The project receives
`.bwenv.toml` and a generated `.envrc` delegating retrieval to bwenv.
Init approves its generated file; review subsequent changes and run `direnv allow`
when direnv asks for approval. `bwenv login` also approves and loads the project.

Trust is local to direnv. Keep unrelated `.envrc` commands under review; approving
an `.envrc` authorizes all its shell code. bwenv hides direnv's routine messages by
default; turn on **Show Direnv Output** in `bwenv config` for diagnosis.

## Mise

Install mise and enable activation **once** in your RC file:

| Shell | Command |
| --- | --- |
| Zsh | `eval "$(mise activate zsh)"` |
| Bash | `eval "$(mise activate bash)"` |
| Fish | `mise activate fish \| source` |

```sh
bwenv init --activation mise
```

After sourcing the printed RC file or opening a new terminal, return to the
project, review its scripts and run:

```sh
mise trust
bwenv login
```

bwenv generates `.bwenv.mise.sh` and adds this block to `mise.toml` while preserving
unrelated settings:

```toml
[env]
# bwenv activation (experimental)
_.source = { path = ".bwenv.mise.sh", tools = true }
```

`tools = true` matters when the Bitwarden CLI uses a mise-managed Node runtime.
It makes active tools available during evaluation and avoids recursion through
mise shims. Keep this setting when reviewing/upgrading older generated scripts.
If `env._.source` already belongs to another script, setup reports the conflict;
review and integrate the sources manually rather than overwriting that setting.

Mise may evaluate its source after unrelated commands. bwenv's automatic source
is quiet: it prints no successful export summary each time. When authentication
is missing, the parent shell displays one login hint per project entry. The hook
must not turn that into repeated password prompts. Use `bwenv export` through the
wrapper or `bwenv doctor` to investigate a failure explicitly.

## Switching an existing project

1. Keep a copy of `.bwenv.toml` and review any custom `.envrc` or mise configuration.
2. If active, run `bwenv disallow` in the current shell.
3. Run `bwenv init --activation TARGET` and choose the same provider/folder/items.
4. Source the RC file printed by setup; enable the new external hook if needed.
5. Review and remove obsolete generated activation artifacts from the old backend.
   In `mise.toml`, remove only bwenv's old source block; preserve your tool settings.
6. Complete direnv/mise trust if required, then run `bwenv login` and verify a test field.

Changing `activation.mode` alone does not install the target backend or remove
old artifacts. Changing the global default does not migrate existing projects.
Avoid having two backends load the same project accidentally.

## Session, lock and disablement

- Bitwarden's session belongs to the current shell. A new shell normally requires
  login. bwenv does not save `BW_SESSION` in files to avoid that step.
- 1Password desktop integration or an existing valid session/service-account token
  can provide authentication; its provider decides when another prompt is needed.
- `bwenv lock` clears provider credentials and managed variables in the invoking
  shell and blocks automatic loads there until successful login/allow.
- `bwenv disallow` persists `activation.disabled = true` for native/mise projects.
  This travels with `.bwenv.toml`; direnv keeps its own local trust mechanism.
- Leaving a project unloads its variables without disabling it or logging out.

See [troubleshooting](troubleshooting.md) for startup errors, repeated messages,
missing wrappers and older mise shim recursion.
