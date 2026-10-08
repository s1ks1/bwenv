# Scripts and CI

[Documentation home](../README.md) · [Commands](commands.md) · [Security](security.md)

Automatic exports need an already authenticated provider. They do not prompt,
run an implicit Bitwarden sync, or create a persistent credential store.
Use an ephemeral runner and the smallest provider selection your task needs.

## Bash/Zsh scripts

Scripts do not necessarily load interactive RC files or the bwenv wrapper.
Capture the raw binary's output, check success, then evaluate it in the script's
shell. Example with an explicit Bash dialect:

```bash
#!/usr/bin/env bash
set -euo pipefail
set +x

if exports="$(SHELL=/bin/bash command bwenv export --project /path/to/project --quiet)"; then
  eval "$exports"
  unset exports
else
  printf '%s\n' 'Could not load project secrets' >&2
  exit 1
fi

./your-application
```

Replace `/path/to/project` and the application command. Setting `SHELL` here
selects Bash assignment syntax; it does not start a new shell. The application
runs in the script's working directory, not automatically the project path.
Quiet exports of a disabled project intentionally emit no assignments. Check the
variables required by your application before starting it if missing values
should fail the task. For direct selection use `--provider bitwarden --folder NAME --folder-id ID`
in place of `--project ...`, never both together.

> [!CAUTION]
> The captured string contains plaintext values. Do not print it, enable `set -x`,
> save it as an artifact, or write it to `.env`/`GITHUB_ENV`. Shell quoting protects
> evaluation syntax, not secrecy from processes running under the same user.

## Fish scripts

Ensure `SHELL` identifies the installed Fish executable when invoking the raw binary:

```fish
set -l fish_executable (command -s fish)
set -l exports "$(env SHELL=$fish_executable bwenv export --project /path/to/project --quiet)"
set -l export_status $status
if test $export_status -ne 0
    echo 'Could not load project secrets' >&2
    exit 1
end
eval "$exports"
set -e exports
./your-application
```

This calls the binary via `env`, bypassing an interactive wrapper. Lock cleanup
requires a different failure contract; prefer the installed wrapper for interactive
`bwenv lock`, which applies validated cleanup even on provider-lock failure.

## Provider authentication

| Provider | Headless authentication |
| --- | --- |
| Bitwarden | Existing valid `BW_SESSION` and CLI account state supplied securely to the runner |
| 1Password | `OP_SERVICE_ACCOUNT_TOKEN` with access to the selected vault, or an existing valid `OP_SESSION_*` |

A Bitwarden API login and vault unlock are separate provider operations. An API
credential alone does not replace the decrypted vault session. Expired sessions
fail the job; do not add interactive password prompts inside CI. Provision
credentials through your CI secret store or provider-supported workflow.

For local interactive 1Password use, prefer desktop integration. bwenv deliberately
discards `op signin` stdout; it does not parse session assignment text. For
non-desktop sessions, use the official provider setup before running bwenv.

## GitHub Actions example: 1Password

Install **bwenv v3** and `op` on the runner's PATH before this step. Pin the
bwenv release and provider CLI version in your organization's installation steps;
see [installation](../INSTALL.md) for available methods.
Store the service-account token in repository/environment secret `OP_SERVICE_ACCOUNT_TOKEN`.
Ensure the canonical project metadata selects a vault that account can access.

```yaml
- name: Run application with vault variables
  shell: bash
  env:
    OP_SERVICE_ACCOUNT_TOKEN: ${{ secrets.OP_SERVICE_ACCOUNT_TOKEN }}
  run: |
    set -euo pipefail
    set +x
    if exports="$(SHELL=/bin/bash command bwenv export --project . --quiet)"; then
      eval "$exports"
      unset exports
    else
      echo "Could not load project secrets" >&2
      exit 1
    fi
    ./your-application
```

Run the workload in the same step/process tree. Variables loaded this way do not
carry over to later steps, and dynamic vault values are not automatically masked
by GitHub. Keep application logs from exposing them. Provider access for untrusted
pull requests requires separate review; do not give those jobs vault credentials.

## Performance measurement

After logging in inside the selected project:

```sh
bwenv benchmark
python3 scripts/measure-performance.py /path/to/project --binary /path/to/bwenv --runs 5 --output /tmp/bwenv-performance.json
```

The helper runs from this repository and requires Python 3.11+. It records CLI
versions, counts and timings without values or session tokens. Use distinct
reports and binary paths for a before/after comparison. It measures authenticated
requests, not first-time provider login or unlocking a cold vault.
See [performance](performance.md) and [release readiness](release-readiness.md).
