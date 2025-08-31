{
  description = "Flake to manage SendGrid provider workspace.";
  inputs.nixpkgs.url = "nixpkgs/nixpkgs-unstable";
  inputs.flake-parts.url = "github:hercules-ci/flake-parts";
  outputs =
    inputs@{
      self,
      nixpkgs,
      flake-parts,
    }:
    flake-parts.lib.mkFlake { inherit inputs; } {
      systems = [
        "x86_64-linux"
        "aarch64-linux"
        "aarch64-darwin"
        "x86_64-darwin"
      ];
      perSystem =
        { pkgs, system, ... }:
        let
          pkgs = import nixpkgs {
            inherit system;
            config.allowUnfreePredicate =
              pkg:
              pkgs.lib.getName [
                pkgs.terraform
              ];
          };

          sharedBuildInputs = with pkgs; [
            git
            pinact
            sops
            go
            tenv
            proxychains-ng
          ];
        in
        {
          devShells = {
            default = pkgs.mkShell {
              shellHook = ''
                export PS1='\n\[\033[1;34m\][:\w]\$\[\033[0m\] '
                export TENV_AUTO_INSTALL=true;
              '';
              buildInputs = sharedBuildInputs;
            };
          };
        };
    };
}
