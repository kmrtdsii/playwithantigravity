{
  description = "GitGym Dev Environment (Nix Flake)";

  inputs = {
    # 常に最新のパッケージを使いたい場合は unstable を指定
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs {
          inherit system;
          config.allowUnfree = true; # Allow proprietary binaries (e.g., Playwright)
        };
      in
      {
        # Formatter for 'nix fmt'
        formatter = pkgs.nixpkgs-fmt;

        # Checks for 'nix flake check'
        checks = {
          # formatted = pkgs.runCommand "check-format" { } "${pkgs.nixpkgs-fmt}/bin/nixpkgs-fmt --check ${self}";
        };

        devShells.default = pkgs.mkShell {
          # 1. Packages
          buildInputs = with pkgs; [
            go_1_25
            golangci-lint
            nodejs_22
            docker
            docker-compose
            git
            # Playwright binaries
            playwright-driver.browsers
            nixpkgs-fmt # Available in shell too
          ];

          # 2. Environment Setup
          shellHook = ''
            echo "--- GitGym Dev Environment ---"
            echo "Go: $(go version)"
            echo "Node: $(node -v) (Expected: v22.x)"
            
            # Platform detection for node_modules
            CURRENT_PLATFORM="$(uname -s)-$(uname -m)"
            PLATFORM_FILE="frontend/node_modules/.platform"
            
            if [ -d "frontend/node_modules" ]; then
              if [ -f "$PLATFORM_FILE" ]; then
                STORED_PLATFORM="$(cat $PLATFORM_FILE)"
                if [ "$CURRENT_PLATFORM" != "$STORED_PLATFORM" ]; then
                  echo ""
                  echo "⚠️  node_modules was created on a different platform ($STORED_PLATFORM)"
                  echo "   Current platform: $CURRENT_PLATFORM"
                  echo "   Please reinstall: cd frontend && rm -rf node_modules && npm install"
                  echo ""
                fi
              else
                echo ""
                echo "⚠️  node_modules exists but platform info is missing."
                echo "   Recommend: cd frontend && rm -rf node_modules && npm install"
                echo ""
              fi
            else
              echo ""
              echo "📦 node_modules not found. Run: cd frontend && npm install"
              echo ""
            fi

            # Check Git Config
            if [ -f "scripts/check-git-config.sh" ]; then
              /bin/bash scripts/check-git-config.sh
            fi

            # Ensure git man pages are accessible
            export MANPATH="${pkgs.git}/share/man:$MANPATH"
            export PLAYWRIGHT_BROWSERS_PATH="${pkgs.playwright-driver.browsers}"

            # Optional Zsh completion
            export SHELL=$(which zsh)
          '';
        };
      });
}

