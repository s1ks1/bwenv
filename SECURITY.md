# Reporting a security issue

Use GitHub's private [Report a vulnerability](https://github.com/s1ks1/bwenv/security/advisories/new)
form if private reporting is enabled for this repository. If it is unavailable,
open an issue asking the maintainer for a private reporting channel, without
publishing exploit details or sensitive data.

Describe the affected version/platform, the trust boundary involved and a minimal
reproduction using invented credentials. Never send real vault data, exported
variables, session tokens or raw provider responses. Coordinate disclosure with
the maintainer before making exploit details public.

The v3 line is under development; native shell and mise integration are experimental.
The implemented security baseline and outstanding work are documented in
[docs/security.md](docs/security.md). No response-time or security-audit guarantee
is implied by this policy.
