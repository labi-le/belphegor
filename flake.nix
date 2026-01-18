{
  description = "Belphegor clipboard manager flake";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs =
    {
      self,
      nixpkgs,
      flake-utils,
    }:
    let
      version = "3.6.0";
      pname = "belphegor";
      supportedSystems = [
        "x86_64-linux"
        "aarch64-linux"
      ];

      systemConfigs = {
        x86_64-linux = {
          arch = "linux_amd64";
          hash = "sha256-hnQ7uza0wOiOGnuWB7eP/lUd5G02aN7EIRdoyqXPvNE="; # x86_64-linux
        };
        aarch64-linux = {
          arch = "linux_armv6";
          hash = "sha256-2fMUumVEC7Cw+db6RLPe2NI3WOJfMr1iNXGpO0fTMUU="; # aarch64-linux
        };
      };
    in
    flake-utils.lib.eachSystem supportedSystems (
      system:
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
            description = "Belphegor clipboard manager";
            homepage = "https://github.com/labi-le/belphegor";
            license = licenses.mit;
            platforms = supportedSystems;
          };
        };
      }
    )
    // {
      nixosModules.default =
        {
          config,
          lib,
          pkgs,
          ...
        }:
        let
          cfg = config.services.belphegor;
          defaultPackage = self.packages.${pkgs.stdenv.hostPlatform.system}.default;

          boolStr = val: if val then "true" else "false";
        in
        {
          options.services.belphegor = with lib; {
            enable = mkEnableOption "Belphegor clipboard manager";

            package = mkOption {
              type = types.package;
              default = defaultPackage;
              description = "The belphegor package to use";
            };

            verbose = mkOption {
              type = types.bool;
              default = false;
              description = "Enable verbose logging";
            };

            connect = mkOption {
              type = types.nullOr types.str;
              default = null;
              example = "192.168.1.5:8090";
              description = "Address in ip:port format to connect to the node";
            };

            port = mkOption {
              type = types.nullOr types.port;
              default = null;
              description = "Port to use. By default: random";
            };

            secret = mkOption {
              type = types.nullOr types.str;
              default = null;
              description = "Key to connect between nodes. Warning: visible in process list";
              example = "paprika";
            };

            maxPeers = mkOption {
              type = types.nullOr types.int;
              default = null;
              description = "Maximum number of discovered peers";
            };

            nodeDiscover = mkOption {
              type = types.nullOr types.bool;
              default = null;
              description = "Find local nodes on the network";
            };

            discoverDelay = mkOption {
              type = types.nullOr types.str;
              default = null;
              example = "5m";
              description = "Delay between node discovery";
            };

            keepAlive = mkOption {
              type = types.nullOr types.str;
              default = null;
              example = "1m";
              description = "Interval for checking connections";
            };

            writeTimeout = mkOption {
              type = types.nullOr types.str;
              default = null;
              example = "1m";
              description = "Write timeout";
            };

            readTimeout = mkOption {
              type = types.nullOr types.str;
              default = null;
              example = "1m";
              description = "Read timeout";
            };

            maxFileSize = mkOption {
              type = types.nullOr types.str;
              default = null;
              example = "500MiB";
              description = "Maximum file size to receive";
            };

            fileSavePath = mkOption {
              type = types.nullOr types.path;
              default = null;
              description = "Folder where received files will be saved";
            };

            notify = mkOption {
              type = types.nullOr types.bool;
              default = null;
              description = "Enable notifications. App default: true";
            };
          };

          config = lib.mkIf cfg.enable {
            environment.systemPackages = [ cfg.package ];

            systemd.user.services.belphegor = {
              description = "Belphegor Clipboard Manager";
              documentation = [ "https://github.com/labi-le/belphegor" ];

              partOf = [ "graphical-session.target" ];
              after = [
                "graphical-session.target"
                "network.target"
              ];
              wants = [ "network-online.target" ];
              wantedBy = [ "graphical-session.target" ];

              path = [ pkgs.libnotify ];

              unitConfig = {
                ConditionEnvironment = [
                  "|DISPLAY"
                  "|WAYLAND_DISPLAY"
                ];
              };

              serviceConfig = {
                Type = "simple";
                Restart = "on-failure";
                RestartSec = "10";

                ExecStart =
                  let
                    args = lib.flatten [
                      (lib.optional cfg.verbose "--verbose")

                      (lib.optional (cfg.connect != null) [
                        "--connect"
                        cfg.connect
                      ])
                      (lib.optional (cfg.port != null) [
                        "--port"
                        (toString cfg.port)
                      ])
                      (lib.optional (cfg.secret != null) [
                        "--secret"
                        cfg.secret
                      ])
                      (lib.optional (cfg.maxPeers != null) [
                        "--max_peers"
                        (toString cfg.maxPeers)
                      ])
                      (lib.optional (cfg.maxFileSize != null) [
                        "--max_file_size"
                        cfg.maxFileSize
                      ])
                      (lib.optional (cfg.fileSavePath != null) [
                        "--file_save_path"
                        (toString cfg.fileSavePath)
                      ])

                      (lib.optional (cfg.discoverDelay != null) [
                        "--discover_delay"
                        cfg.discoverDelay
                      ])
                      (lib.optional (cfg.keepAlive != null) [
                        "--keep_alive"
                        cfg.keepAlive
                      ])
                      (lib.optional (cfg.writeTimeout != null) [
                        "--write_timeout"
                        cfg.writeTimeout
                      ])
                      (lib.optional (cfg.readTimeout != null) [
                        "--read_timeout"
                        cfg.readTimeout
                      ])

                      (lib.optional (cfg.nodeDiscover != null) "--node_discover=${boolStr cfg.nodeDiscover}")
                      (lib.optional (cfg.notify != null) "--notify=${boolStr cfg.notify}")
                    ];
                  in
                  "${cfg.package}/bin/belphegor ${lib.escapeShellArgs args}";
              };
            };
          };
        };
    };
}