class IngressoFinderCli < Formula
  desc "Use ingresso.com directly from the terminal"
  homepage "https://github.com/publi0/ingresso-finder-cli"
  COMMIT = "0d36fd59441a0bf151f416f8048ba6716fb5088a".freeze
  url "https://github.com/publi0/ingresso-finder-cli/archive/#{COMMIT}.tar.gz"
  version "0.0.0-#{COMMIT[0, 7]}"
  sha256 "96dac4a536641d895ced1c0f8bedd0a2c0c642f8b590d820f2f3243b220bd580"

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
