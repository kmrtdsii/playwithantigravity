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
          config.allowUnfree = true; # Playwrightなどの商用バイナリを許可
        };
      in
      {
        devShells.default = pkgs.mkShell {
          # 1. パッケージの定義 (Features に相当)
          buildInputs = with pkgs; [
            go_1_25
            nodejs_22
            docker
            docker-compose
            git
            # Playwright の実行に必要なバイナリ
            playwright-driver.browsers
          ];

          # 2. 環境変数の設定 (Settings に相当)
          shellHook = ''
            # Force Nix Node version to take precedence over local tools (like Volta)
            export IN_NIX_SHELL=1
            
            # Explicitly alias binaries to ensure the correct version is used even if PATH is mangled
            # alias node=${pkgs.nodejs_22}/bin/node
            # alias npm=${pkgs.nodejs_22}/bin/npm
            # alias npx=${pkgs.nodejs_22}/bin/npx

            # Playwright Config:
            # Nix (playwright-driver) と NPM パッケージのバージョン不一致を避けるため、
            # ブラウザ管理は Playwright 標準の仕組み (npx playwright install) に任せます。
            # export PLAYWRIGHT_BROWSERS_PATH=${pkgs.playwright-driver.browsers}
            # export PLAYWRIGHT_SKIP_BROWSER_DOWNLOAD=1

            echo "--- GitGym Dev Environment ---"
            echo "Go: $(go version)"
            echo "Node: $(node -v) (Expected: v22.x)"

            # postCreateCommand の再現 (direnv の高速化のため自動実行を抑制)
            echo "To install/update dependencies, run: (cd backend && go mod download) && (cd frontend && npm install)"
            # (cd backend && go mod download)
            # (cd frontend && npm install)
            
            # Git 設定のチェック
            if [ -f "scripts/check-git-config.sh" ]; then
              /bin/bash scripts/check-git-config.sh
            fi

            # Zsh を使っている場合の補完設定（任意）
            export SHELL=$(which zsh)
          '';
        };
      });
}

