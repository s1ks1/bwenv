# Native shell demo

[Documentation home](../README.md) · [Getting started](getting-started.md)

![Native shell workflow with fake vault data](../assets/workflow.gif)

## Accessible transcript

The animation uses the actual bwenv binary, native Bash hook and command wrapper
with the repository's deterministic fake Bitwarden CLI. The project is preconfigured
with a single test item. It demonstrates entry, authentication, presence checking
and environment restoration; it is not a recording of the interactive setup picker.

```text
$ cd project
  [!] bwenv · Session locked or expired · run bwenv login
$ bwenv login
  [OK] bwenv · 1 variable loaded · Bitwarden / Fixture
$ test -n "${API_KEY:-}" && echo "API_KEY is loaded"
API_KEY is loaded
$ cd ..
  [OK] bwenv · Project deactivated · 1 variables restored
$ test -z "${API_KEY+x}" && echo "API_KEY is cleared"
API_KEY is cleared
```

Only variable presence is displayed. The generator also checks the expected fake
value internally and fails if fake credentials appear in the captured output.
No real provider configuration, RC file or session environment is inherited by
the demonstration shell. Prompt events are invoked explicitly for reproducibility.

## Reproduce

From the repository, with Go, Bash, Python 3.11+ and Pillow available:

```sh
python3 scripts/render-demo.py
```

Pillow is an optional documentation-rendering dependency; it is not needed to
install, build or run bwenv. The script builds both executables in a temporary
directory, captures successful shell interactions and replaces `assets/workflow.gif`.
It leaves your real project and shell configuration alone. The GIF loops through
five frames; the static transcript above provides the same information without animation.
