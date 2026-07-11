# Real five-VM FULL-MESH end-to-end test for belphegor via mDNS discovery.
#
# Unlike vm-test.nix (an explicit star with -c), this boots five NixOS QEMU
# guests with peer discovery enabled (--node_discover) and NO static peers.
# The nodes find each other over the virtual LAN's multicast and form a full
# mesh, so a clipboard copy on ANY node reaches every other node (which is not
# possible in a star, where the architecture only propagates to the origin's
# direct peers). A copy is driven on a NON-hub node (node3) and must land on
# all four others.
#
# Build the static binary first, then run:
#
#   CGO_ENABLED=0 go build -tags null -ldflags='-s -w' -o /tmp/belphegor-null-static ./cmd/cli
#   nix build --impure -f e2e/vm-mesh.nix -L
#
{
  pkgs ? import <nixpkgs> { },
}:
let
  belphegor = pkgs.runCommandLocal "belphegor-null" { } ''
    mkdir -p $out/bin
    cp ${
      builtins.path {
        path = "/tmp/belphegor-null-static";
        name = "belphegor-null-static";
      }
    } $out/bin/belphegor
    chmod +x $out/bin/belphegor
  '';

  secret = "vm-mesh-secret";
  port = 19001;
  payload = "MESH-PAYLOAD-any-origin";

  mkNode =
    { id }:
    { pkgs, ... }:
    {
      networking.firewall.enable = false;
      virtualisation.memorySize = 512;
      systemd.services.belphegor = {
        description = "belphegor mesh node";
        wantedBy = [ "multi-user.target" ];
        after = [ "network-online.target" ];
        wants = [ "network-online.target" ];
        serviceConfig = {
          ExecStartPre = "${pkgs.coreutils}/bin/touch /tmp/clip_in /tmp/clip_out";
          ExecStart = builtins.concatStringsSep " " [
            "${belphegor}/bin/belphegor"
            "-p ${toString port}"
            "--transport tcp"
            "--node_discover=true"
            "--discover_delay 2s"
            "--secret ${secret}"
            "--verbose"
          ];
          Environment = [
            "BELPHEGOR_NODE_ID=${toString id}"
            "BELPHEGOR_HEADLESS_IN=/tmp/clip_in"
            "BELPHEGOR_HEADLESS_OUT=/tmp/clip_out"
            "HOME=/root"
          ];
          Restart = "on-failure";
        };
      };
    };
in
pkgs.testers.runNixOSTest {
  name = "belphegor-5node-mesh-e2e";

  nodes = {
    node1 = mkNode { id = 1; };
    node2 = mkNode { id = 2; };
    node3 = mkNode { id = 3; };
    node4 = mkNode { id = 4; };
    node5 = mkNode { id = 5; };
  };

  testScript = ''
    everyone = [node1, node2, node3, node4, node5]

    start_all()
    for m in everyone:
        m.wait_for_unit("belphegor.service")

    # business logic #1: discovery forms a full mesh - each node must connect
    # to the other four peers.
    for m in everyone:
        m.wait_until_succeeds("[ $(journalctl -u belphegor | grep -c connected) -ge 4 ]", timeout=120)

    # drive a clipboard copy on a NON-hub node.
    node3.succeed("echo -n '${payload}' > /tmp/clip_in")

    # end-to-end: it must reach and be injected into every OTHER node.
    for m in [node1, node2, node4, node5]:
        m.wait_until_succeeds("grep -q '${payload}' /tmp/clip_out", timeout=60)
        m.wait_until_succeeds("journalctl -u belphegor | grep -q 'set clipboard data'", timeout=60)

    # business logic #2: the origin detected + broadcast, and did not self-echo.
    node3.wait_until_succeeds("journalctl -u belphegor | grep -q 'new update'")
    node3.wait_until_succeeds("journalctl -u belphegor | grep -q announced")
    node3.fail("grep -q '${payload}' /tmp/clip_out")

    print("==== node3 (origin) log ====")
    print(node3.succeed("journalctl -u belphegor | grep -oE '(discovered|connected|new update|announced|received request|sending)' | sort | uniq -c"))
    for m in [node1, node2, node4, node5]:
        print(f"==== {m.name} received via mesh ====")
        print(m.succeed("journalctl -u belphegor | grep -oE '(discovered|connected|requesting message|set clipboard data)' | sort -u"))
  '';
}
