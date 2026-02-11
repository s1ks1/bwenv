class Bwenv < Formula
  desc "Bitwarden + direnv helper - sync secrets from Bitwarden into your shell environment"
  homepage "https://github.com/s1ks1/bwenv"
  url "https://github.com/s1ks1/bwenv/archive/refs/tags/v1.1.1.tar.gz"
  sha256 "PLACEHOLDER"
  license "MIT"
  head "https://github.com/s1ks1/bwenv.git", branch: "main"

  depends_on "direnv"
  depends_on "jq"

  def install
    bin.install "setup/bwenv"
    (share/"bwenv").install "setup/bitwarden_folders.sh"
  end

  def post_install
    # bwenv auto-copies the helper to ~/.config/direnv/lib/ on first use
    ohai "Run 'bwenv test' to verify your setup"
  end

  def caveats
    <<~EOS
      The helper script is installed to:
        #{share}/bwenv/bitwarden_folders.sh

      It will be auto-copied to ~/.config/direnv/lib/ on first use of
      'bwenv init' or 'bwenv interactive'.

      Ensure direnv is hooked into your shell:
        bash: eval "$(direnv hook bash)"
        zsh:  eval "$(direnv hook zsh)"
        fish: direnv hook fish | source

      Install Bitwarden CLI separately:
        npm install -g @bitwarden/cli
        # or download from https://bitwarden.com/help/cli/
    EOS
  end

  test do
    assert_match "Usage:", shell_output("#{bin}/bwenv")
  end
end
