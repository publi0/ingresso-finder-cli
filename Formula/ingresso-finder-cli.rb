class IngressoFinderCli < Formula
  desc "Use ingresso.com directly from the terminal"
  homepage "https://github.com/publi0/ingresso-finder-cli"
  COMMIT = "a700e641a9671e7f09c8b6f07d13a176ddb9f188".freeze
  url "https://github.com/publi0/ingresso-finder-cli/archive/#{COMMIT}.tar.gz"
  version "0.0.0-#{COMMIT[0, 7]}"
  sha256 "a22fa5808351d6e347a2cd614ec98a5a6625f92c2877808cb325156ba1749533"

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
