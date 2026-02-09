class IngressoFinderCli < Formula
  desc "Use ingresso.com directly from the terminal"
  homepage "https://github.com/publi0/ingresso-finder-cli"
  COMMIT = "9d47b7413fda76d64515196283583042d0186e97".freeze
  url "https://github.com/publi0/ingresso-finder-cli/archive/#{COMMIT}.tar.gz"
  version "0.0.0-#{COMMIT[0, 7]}"
  sha256 "3c99a5785140a15047e35d6aa2076076daf50c109d3e32a3870e5ac09226f6a0"

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
