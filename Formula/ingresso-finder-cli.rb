class IngressoFinderCli < Formula
  desc "Use ingresso.com directly from the terminal"
  homepage "https://github.com/publi0/ingresso-finder-cli"
  COMMIT = "eed75a56e08da9cb774d34b6ee6252e230f9e8fd"
  url "https://codeload.github.com/publi0/ingresso-finder-cli/tar.gz/#{COMMIT}"
  sha256 "c88a4e37893937dd03df29d78747ef15d85129a0ca8e64b83b2885fee8b3716a"
  version "0.0.0-#{COMMIT[0,7]}"

  depends_on "go" => :build

  def install
    ldflags = %W[
      -s -w
      -X main.version=#{version}
      -X main.commit=#{COMMIT}
    ]

    system "go", "build", *std_go_args(output: bin/"ingresso", ldflags: ldflags)
  end

  test do
    assert_match "ingresso", shell_output("#{bin}/ingresso --version")
  end
end
