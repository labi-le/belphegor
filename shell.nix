{ pkgs ? import <nixpkgs> {} }:

pkgs.mkShell {
  packages = with pkgs; [
    go
    protobuf_33
  ];

  shellHook = ''
    unset GOROOT
    export GOTOOLCHAIN=local
    go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
    go install github.com/planetscale/vtprotobuf/cmd/protoc-gen-go-vtproto@latest
    go install golang.org/x/perf/cmd/benchstat@latest
  '';
}
