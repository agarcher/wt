# typed: false
# frozen_string_literal: true

class Wt < Formula
  desc "Git worktree manager with lifecycle hooks"
  homepage "https://github.com/agarcher/wt"
  version "0.1.0"
  license "MIT"

  on_macos do
    on_intel do
      url "https://github.com/agarcher/wt/releases/download/v#{version}/wt-darwin-amd64.tar.gz"
      sha256 "PLACEHOLDER_DARWIN_AMD64_SHA256"
    end

    on_arm do
      url "https://github.com/agarcher/wt/releases/download/v#{version}/wt-darwin-arm64.tar.gz"
      sha256 "PLACEHOLDER_DARWIN_ARM64_SHA256"
    end
  end

  on_linux do
    on_intel do
      url "https://github.com/agarcher/wt/releases/download/v#{version}/wt-linux-amd64.tar.gz"
      sha256 "PLACEHOLDER_LINUX_AMD64_SHA256"
    end

    on_arm do
      url "https://github.com/agarcher/wt/releases/download/v#{version}/wt-linux-arm64.tar.gz"
      sha256 "PLACEHOLDER_LINUX_ARM64_SHA256"
    end
  end

  def install
    # Determine binary name based on OS and architecture
    binary_name = "wt-#{OS.kernel_name.downcase}-#{Hardware::CPU.arch == :arm64 ? "arm64" : "amd64"}"
    bin.install binary_name => "wt"

    # Install shell completions
    generate_completions_from_executable(bin/"wt", "completion")
  end

  def caveats
    <<~EOS
      To enable wt shell integration, add the following to your shell config:

      For zsh (~/.zshrc):
        eval "$(wt init zsh)"

      For bash (~/.bashrc):
        eval "$(wt init bash)"

      For fish (~/.config/fish/config.fish):
        wt init fish | source
    EOS
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/wt --version")
  end
end
