# =============================================================================
# Homebrew formula for bwenv
#
# This is a template formula. When using GoReleaser, the actual formula
# is auto-generated and pushed to the homebrew-bwenv tap repository.
# This file serves as a reference and for manual installations.
#
# Install:
#   brew tap s1ks1/bwenv
#   brew install bwenv
# =============================================================================

class Bwenv < Formula
  desc "Sync secrets from password managers (Bitwarden, 1Password) into your shell via direnv"
  homepage "https://github.com/s1ks1/bwenv"
  url "https://github.com/s1ks1/bwenv/archive/refs/tags/v2.0.0.tar.gz"
  sha256 "PLACEHOLDER"
  license "MIT"
  head "https://github.com/s1ks1/bwenv.git", branch: "go-rewrite"

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
    system "go", "build", *std_go_args(ldflags:), "."
  end

  def caveats
    <<~EOS
      To get started, run:

        bwenv test

      Make sure you have at least one password manager CLI installed:
        - Bitwarden: brew install bitwarden-cli
        - 1Password: brew install --cask 1password-cli

      And hook direnv into your shell:
        bash: eval "$(direnv hook bash)"
        zsh:  eval "$(direnv hook zsh)"
        fish: direnv hook fish | source
    EOS
  end

  test do
    # Verify the binary runs and reports its version.
    assert_match "bwenv", shell_output("#{bin}/bwenv version")

    # Verify the test command runs without crashing (it will report
    # missing dependencies, but should not error out).
    output = shell_output("#{bin}/bwenv test")
    assert_match "System Information", output
  end
end
