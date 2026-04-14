{
  description = "5G Core deploy devshell with Ansible";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-25.11";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
      in
      {
        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [
            kubectl
            k9s
            (python3.withPackages (ps: with ps; [ pip ]))
          ];

          shellHook = ''
            export VIRTUAL_ENV="$PWD/.venv"
            if [ ! -d "$VIRTUAL_ENV" ]; then
              python -m venv "$VIRTUAL_ENV"
            fi
            source "$VIRTUAL_ENV/bin/activate"
            pip install -q ansible
            
            export KUBECONFIG="$PWD/kubeconfig"
          '';
        };
      }
    );
}
