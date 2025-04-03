class Lil < Formula
  desc "Lightweight systray app that displays your Linear issues directly in your system tray/menu bar"
  homepage "https://github.com/pzurek/lil"
  url "https://github.com/pzurek/lil/archive/refs/tags/v0.1.0.tar.gz"
  sha256 "REPLACE_WITH_ACTUAL_SHA256_AFTER_RELEASE"
  license "MIT"
  head "https://github.com/pzurek/lil.git", branch: "main"

  depends_on "go" => :build

  def install
    system "go", "build", *std_go_args(ldflags: "-s -w -X main.version=#{version} -X main.buildTime=#{time.iso8601}")
  end

  test do
    assert_match "Lil version", shell_output("#{bin}/lil -version")
  end
end 