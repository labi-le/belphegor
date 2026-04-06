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
  '';
}
