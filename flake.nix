{
  description = "Keep it handy";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    devenv.url = "github:cachix/devenv";
  };

  outputs = { self, nixpkgs, devenv, ... } @ inputs:
    let
      systems = [ "x86_64-linux" "i686-linux" "x86_64-darwin" "aarch64-linux" "aarch64-darwin" ];
      forAllSystems = f: builtins.listToAttrs (map (name: { inherit name; value = f name; }) systems);
    in
      {
      packages.x86_64-linux.devenv-up = self.devShells.x86_64-linux.default.config.procfileScript;
      devShells = forAllSystems
        (system: let
          pkgs = import nixpkgs { inherit system; };
        in {
          default = devenv.lib.mkShell {
            inherit inputs pkgs;

            modules = [{
              languages.go.enable = true;

              packages = with pkgs; [
                bacon
                operator-sdk
                cloc
                just
                linuxKernel.packages.linux_6_6.perf
                perf-tools
                plantuml

                kubectl
                k9s
                kubectx
                skopeo

                stockfish
              ];
            }];
          };
        });
    };
}
