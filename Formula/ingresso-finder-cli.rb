class IngressoFinderCli < Formula
  desc "Use ingresso.com directly from the terminal"
  homepage "https://github.com/publi0/ingresso-finder-cli"
  commit = "93476c57696be633880473b0e2f6bea659c5f777"
  url "https://codeload.github.com/publi0/ingresso-finder-cli/tar.gz/#{commit}"
  sha256 "7a283842fd57c61cdb54a03535c89f90f0cc2b91a5d0507cfee8a1841e80d456"
  version "0.0.0-#{commit[0,7]}"

  depends_on "go" => :build

  def install
    ldflags = %W[
      -s -w
      -X main.version=#{version}
      -X main.commit=#{commit}
    ]

    system "go", "build", *std_go_args(ldflags: ldflags)
  end

  test do
    assert_match "ingresso-finder-cli", shell_output("#{bin}/ingresso-finder-cli --version")
  end
end
