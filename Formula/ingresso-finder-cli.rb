class IngressoFinderCli < Formula
  desc "Use ingresso.com directly from the terminal"
  homepage "https://github.com/publi0/ingresso-finder-cli"
  COMMIT = "172d6bbca751190ba31bbae06ee508c0d4aba7a6".freeze
  url "https://github.com/publi0/ingresso-finder-cli/archive/#{COMMIT}.tar.gz"
  version "0.0.0-#{COMMIT[0, 7]}"
  sha256 "82c5693ae71ec1baa0346911efc8393561d2d03506ff17db3932016fe676ba79"

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
