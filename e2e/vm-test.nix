# Real multi-VM end-to-end test for belphegor.
#
# Boots FIVE NixOS QEMU guests on a virtual LAN in a star topology (node1 is the
# hub, node2..node5 connect to it), runs the real belphegor binary (built with
# -tags null, the display-less headless backend) as a systemd service on each,
# drives a single clipboard copy on the hub and asserts that:
#   1. all nodes hand-shake and connect,
#   2. the payload fans out and is injected into every leaf's clipboard,
#   3. every business-logic step is present in the debug (journald) logs,
#   4. the hub does not echo its own copy back to itself.
#
# The belphegor binary is injected as a prebuilt static (CGO_ENABLED=0) artifact
# so the test needs no vendorHash/network Go build. Build it first with:
#
#   CGO_ENABLED=0 go build -tags null -ldflags='-s -w' -o /tmp/belphegor-null-static ./cmd/cli
#
# then run the VM test:
#
#   nix build --impure -f e2e/vm-test.nix -L
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

  secret = "vm-e2e-secret";
  port = 19001;
  payload = "VM5-PAYLOAD-mesh";

  mkNode =
    {
      id,
      connectTo ? null,
    }:
    { lib, pkgs, ... }:
    {
      networking.firewall.enable = false;
      virtualisation.memorySize = 512;
      systemd.services.belphegor = {
        description = "belphegor e2e node";
        serviceConfig = {
          ExecStartPre = "${pkgs.coreutils}/bin/touch /tmp/clip_in /tmp/clip_out";
          ExecStart = lib.concatStringsSep " " (
            [
              "${belphegor}/bin/belphegor"
              "-p ${toString port}"
              "--transport tcp"
              "--node_discover=false"
              "--secret ${secret}"
              "--verbose"
            ]
            ++ lib.optional (connectTo != null) "-c ${connectTo}:${toString port}"
          );
          Environment = [
            "BELPHEGOR_NODE_ID=${toString id}"
            "BELPHEGOR_HEADLESS_IN=/tmp/clip_in"
            "BELPHEGOR_HEADLESS_OUT=/tmp/clip_out"
            "HOME=/root"
          ];
          Restart = "no";
        };
      };
    };
in
pkgs.testers.runNixOSTest {
  name = "belphegor-5node-clipboard-e2e";

  nodes = {
    node1 = mkNode { id = 1; };
    node2 = mkNode {
      id = 2;
      connectTo = "node1";
    };
    node3 = mkNode {
      id = 3;
      connectTo = "node1";
    };
    node4 = mkNode {
      id = 4;
      connectTo = "node1";
    };
    node5 = mkNode {
      id = 5;
      connectTo = "node1";
    };
  };

  testScript = ''
    hub = node1
    leaves = [node2, node3, node4, node5]

    start_all()

    # hub listens first; leaves dial it (belphegor exits if the dial fails).
    hub.succeed("systemctl start belphegor")
    hub.wait_for_open_port(${toString port})
    for leaf in leaves:
        leaf.succeed("systemctl start belphegor")

    # business logic #1: every node connects.
    for m in [hub] + leaves:
        m.wait_until_succeeds("journalctl -u belphegor | grep -q connected")

    # drive a single clipboard copy on the hub.
    hub.succeed("echo -n '${payload}' > /tmp/clip_in")

    # end-to-end: it must fan out and be injected into EVERY leaf's clipboard,
    # and each leaf must have pulled + injected it (business logic in the logs).
    for leaf in leaves:
        leaf.wait_until_succeeds("grep -q '${payload}' /tmp/clip_out")
        leaf.wait_until_succeeds("journalctl -u belphegor | grep -q 'requesting message'")
        leaf.wait_until_succeeds("journalctl -u belphegor | grep -q 'set clipboard data'")

    # business logic #2: the hub detected the copy and broadcast it.
    hub.wait_until_succeeds("journalctl -u belphegor | grep -q 'new update'")
    hub.wait_until_succeeds("journalctl -u belphegor | grep -q announced")

    # business logic #3: the hub must not echo its own copy back to itself.
    hub.fail("grep -q '${payload}' /tmp/clip_out")

    print("==== hub (node1) fan-out log ====")
    print(hub.succeed("journalctl -u belphegor | grep -oE '(connected|new update|announced|received request|sending)' | sort | uniq -c"))
    for leaf in leaves:
        name = leaf.name
        print(f"==== {name} receive log ====")
        print(leaf.succeed("journalctl -u belphegor | grep -oE '(connected|requesting message|set clipboard data)' | sort -u"))
  '';
}
