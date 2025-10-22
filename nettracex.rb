class Nettracex < Formula
  desc "A comprehensive network diagnostic toolkit built with Go, featuring a beautiful terminal user interface"
  homepage "https://nettracex.net"
  url "https://github.com/nettracex/nettracex-tui/archive/refs/tags/v0.1.0.tar.gz"
  sha256 "PLACEHOLDER_REPLACE_WITH_ACTUAL_SHA256"
  license "MIT"
  head "https://github.com/nettracex/nettracex-tui.git", branch: "main"

  depends_on "go" => :build

  def install
    # Set build variables
    ldflags = %W[
      -s -w
      -X main.version=#{version}
      -X main.gitCommit=#{tap.git_head || "unknown"}
      -X main.buildTime=#{Time.now.utc.iso8601}
    ]

    # Build the application
    system "go", "build", *std_go_args(ldflags: ldflags), "."
  end

  test do

    # Test version output
    version_output = shell_output("#{bin}/nettracex -version 2>&1")
    assert_match "NetTraceX", version_output
    
    # Test help output
    help_output = shell_output("#{bin}/nettracex -help 2>&1")
    assert_match "network diagnostic toolkit", help_output.downcase
    assert_match "Interactive Mode", help_output
  end
end