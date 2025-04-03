class Lil < Formula
  desc "Lil - A lightweight systray app to manage Linear issues"
  homepage "https://github.com/pzurek/lil"
  url "https://github.com/pzurek/lil/archive/refs/tags/v0.1.0.tar.gz"
  sha256 "0019dfc4b32d63c1392aa264aed2253c1e0c2fb09216f8e2cc269bbfb8bb49b5"
  license "MIT"
  head "https://github.com/pzurek/lil.git", branch: "main"

  depends_on "go" => :build
  
  # Dependencies for systray on macOS
  if OS.mac?
    depends_on "xcode-build-tools" => :build
  # Dependencies for Linux
  elsif OS.linux?
    depends_on "pkg-config" => :build
    depends_on "gtk+3"
    depends_on "libayatana-appindicator"
  end

  def install
    system "go", "build", *std_go_args(ldflags: "-s -w -X main.version=#{version} -X main.buildTime=#{time.now.utc.iso8601}")
  end

  def caveats
    <<~EOS
      Lil requires a Linear API key to function.
      You can set this with:
        export LINEAR_API_KEY=your_api_key

      For persistent setup, add this to your shell profile (~/.bashrc, ~/.zshrc, etc.).
    EOS
  end

  test do
    assert_match "Error: LINEAR_API_KEY environment variable not set", shell_output("#{bin}/lil 2>&1", 1)
  end
end 