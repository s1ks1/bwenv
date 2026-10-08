# Release verification and publication

The release workflow validates a version tag, builds a draft and attests its
artifacts. Publish the verified draft as latest only after all jobs pass and
provenance verification succeeds. Existing releases must never be replaced.
V3 is the official release line; V2 downloads remain available and possible
patches use the `v2` maintenance branch without changing the latest V3 release.

## Reproduce the checks

Use Go 1.25+ for source compatibility. Release/quality jobs pin Go 1.27.1 and the
analysis tools; action references are pinned to verified commits. The minimum Go
version increased when fixing the Windows `x/sys` vulnerability.

```sh
go mod verify
go test -v -count=1 -race ./...
go vet ./...
staticcheck ./...
govulncheck -show verbose ./...
actionlint
goreleaser check
goreleaser release --snapshot --clean --skip=publish
python3 scripts/verify-release.py dist
```

The release rehearsal requires Syft v1.54.0 on PATH; CI installs that pinned
version. It produces five platform archives, four Linux packages, five CycloneDX
SBOMs and SHA256 checksums. The verifier checks archive contents, required target
coverage, SBOM presence and every recorded digest without executing binaries.
Snapshot versions use `3.0.0-dev`, independent of the latest v2 tag.

CI executes minimum-version builds/tests/vet on Ubuntu, macOS and Windows.
Linux additionally installs Bash/Zsh/Fish, direnv, mise and Node, and runs race,
static, vulnerability and workflow checks. It packages only after those jobs
pass. A manual Release workflow dispatch calls the same verification workflow
and uploads unpublished artifacts; it cannot create/push a tag or publish.

The tag-only release job requires the protected `release` environment, refuses
an existing release and writes a draft with Homebrew/Scoop publishing skipped.
The maintainer reviews that environment deployment before building. Artifact
provenance uses GitHub attestations with job-scoped OIDC/attestation permissions.
Attestation issuance must still be exercised in an authorized release; a local
snapshot is not signed provenance. Verify released assets with
`gh attestation verify <archive> --repo s1ks1/bwenv` before distribution.

Both installers now require an exact, unique valid checksum and fail if it cannot
be fetched or verified. This detects tampering against that checksum file; it is
not independent authentication of a compromised repository. GitHub provenance
adds build identity; it does not certify that the source itself is harmless.

## Publish a verified release

1. Merge the release source and documentation to `main` after CI passes.
2. Create an annotated `v3.x.y` tag on that merged commit and push it.
3. Review the protected environment deployment; wait for validation, packaging
   and provenance attestation to succeed.
4. Download the draft assets, run `python3 scripts/verify-release.py DIRECTORY`,
   and verify archive/package provenance with `gh attestation verify`.
5. Publish the draft and mark it latest. Update the Homebrew and Scoop manifests
   with the exact archive URLs and the verified release checksums.
6. Check a downloaded binary's version and help output and confirm the previous
   V2 release still has its assets. Preserve release tags and downloads.

Real-provider authentication and target-OS installation are separate from the
isolated CI fixtures; report that distinction when publishing.

## Compatibility evidence

| Area | Local evidence / remaining release check |
| --- | --- |
| Bash/Zsh/Fish | Native entry/exit, same-directory selection changes, literal quoting, wrapper upgrade and failed-lock cleanup run against real shells with fake providers. Fish activation preserves failed command status before evaluating output. |
| mise + Node | Actual mise evaluation uses an isolated installed Node runtime and shim, including a locked-vault case; no network install or real vault is needed. |
| direnv | Actual approval/login reload behavior is tested with fake providers. |
| Large Bitwarden folders | A 1000-item fixture and selected first/last items retain one provider call and secret-free measurements. |
| Windows | Home-directory fixtures now set USERPROFILE as well as HOME. POSIX executable fixtures skip on Windows; Windows CLI/provider/build and PowerShell checksum tests run there. Native PowerShell hooks are not implemented. |
| Packages | Five archive/SBOM pairs and four DEB/RPM files verified locally. Target-OS execution/install remains a separate CI/manual check. |
| Real vault/version matrix | Requires an explicitly selected development project and authenticated session; fake fixtures do not prove all provider CLI versions. |

Run the safe measurement harness only against the selected development project:

```sh
python3 scripts/measure-performance.py /path/to/test-project \
  --binary /absolute/path/to/bwenv --runs 5 --output /tmp/bwenv-performance.json
```

It records OS/architecture, CLI version, selection count, first authenticated
request, warm runs/median and provider process counts. It does not export values,
record tokens, or dump provider payloads. “First” is not a freshly locked vault or
purged OS cache. Compare the same project/selection/provider version across
before/after binaries; timing is a local observation, not a CI threshold.
