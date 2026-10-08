# =============================================================================
# Homebrew formula for bwenv (reference template)
#
# NOTE: The official binary cask is published to the homebrew-bwenv tap
# after release artifacts are verified. This file is a
# reference for manual installations and development.
#
# Install:
#   brew tap s1ks1/bwenv
#   brew install bwenv
# =============================================================================

class Bwenv < Formula
  desc "Sync secrets from password managers (Bitwarden, 1Password) into your shell with native, direnv or mise hooks"
  homepage "https://github.com/s1ks1/bwenv"
  url "https://github.com/s1ks1/bwenv/archive/refs/tags/v3.0.0.tar.gz"
  sha256 "PLACEHOLDER"
  license "MIT"
  head "https://github.com/s1ks1/bwenv.git", branch: "main"

  # GoReleaser publishes pre-built binaries, but if building from source
  # we need Go installed as a build dependency.
  depends_on "go" => :build

  # direnv is an optional runtime dependency — bwenv generates .envrc files
  # that direnv loads, but users might install direnv separately.
  depends_on "direnv" => :optional

  def install
    # Inject version info at compile time via ldflags.
    ldflags = %W[
      -s -w
      -X main.Version=#{version}
    ]

    # Build the Go binary and install it to the Homebrew bin directory.
    system "go", "build", *std_go_args(ldflags: ldflags), "."
  end

  def caveats
    <<~EOS
      To get started, run:

        bwenv status

      Make sure you have at least one password manager CLI installed:
        - Bitwarden: brew install bitwarden-cli
        - 1Password: brew install --cask 1password-cli

      Run bwenv init and follow the shell setup steps.
      Native shell activation is the default; bwenv config selects direnv or mise.
    EOS
  end

  test do
    # Verify the binary runs and reports its version.
    assert_match "bwenv", shell_output("#{bin}/bwenv version")
  end
end
