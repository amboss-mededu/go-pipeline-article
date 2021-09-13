{
  description = "Demonstrating The Pipelines Pattern In Golang";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    flake-utils.url = "github:numtide/flake-utils";
    golang.url = "https://dl.google.com/go/go1.17.src.tar.gz";
    golang.flake = false;
  };

  outputs = { self, nixpkgs, golang, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let pkgs = import nixpkgs {
        inherit system;
        overlays = [
          (self: super: {
            go = super.go.overrideAttrs
              (old: rec {
                version = "1.17";
                src = golang;
              });
          })
        ];
      };
      in
      {
        devShell = pkgs.mkShell {
          buildInputs = [
            pkgs.go
            pkgs.gotools
            pkgs.golangci-lint
            pkgs.gopls
            pkgs.go-outline
            pkgs.gopkgs
          ];
        };
      });
}
