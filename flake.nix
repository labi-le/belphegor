{
  description = "Belphegor clipboard manager flake";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    let
      version = "2.3.0";
      pname = "belphegor";
      supportedSystems = [ "x86_64-linux" "aarch64-linux" ];

      systemConfigs = {
        x86_64-linux = {
          arch = "linux_amd64";
          hash = "sha256-FAIr2jTkPgN3zYJqOH7vyTzrPeZ6y7F5Nx9D/m7p45M="; # x86_64-linux
        };
        aarch64-linux = {
          arch = "linux_armv6";
          hash = "sha256-hrDcfqUqrI79kZi926H8zjF+HvhQNIMNlmUXdfL/TxE="; # aarch64-linux
        };
      };
    in
    flake-utils.lib.eachSystem supportedSystems (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
        config = systemConfigs.${system};
      in
      {
        packages.default = pkgs.stdenv.mkDerivation {
          inherit pname version;

          src = pkgs.fetchurl {
            url = "https://github.com/labi-le/belphegor/releases/download/v${version}/${pname}_${version}_${config.arch}";
            hash = config.hash;
          };

          dontUnpack = true;

          installPhase = ''
            mkdir -p $out/bin
            cp $src $out/bin/${pname}
            chmod +x $out/bin/${pname}
          '';

          meta = with pkgs.lib; {
            description = "Belphegor is a clipboard manager that allows you to share your clipboard with other devices on the network";
            homepage = "https://github.com/labi-le/belphegor";
            license = licenses.mit;
            platforms = supportedSystems;
          };
        };
      }
    );
}