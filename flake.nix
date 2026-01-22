{
  description = "Microforge";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-24.11";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs { inherit system; };
      in
      {
        packages.default = pkgs.buildGoModule {
          pname = "microforge";
          version = "0.1.0";
          src = ./.;
          vendorHash = null;
          subPackages = [ "cmd/mforge" ];
        };

        devShells.default = pkgs.mkShell {
          buildInputs = [ pkgs.go pkgs.tmux pkgs.git ];
        };
      });
}
