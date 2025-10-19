{
  description = "DevDashboard - Repository management and dependency analysis tool";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs =
    {
      self,
      nixpkgs,
      flake-utils,
    }:
    flake-utils.lib.eachDefaultSystem (
      system:
      let
        pkgs = import nixpkgs { inherit system; };

        version = "0.2.0";

        vendorHash = "sha256-Opv+u4wM4QL00GjSkrYayb2aS3eZIhTOkfBZ5niGhzo=";

        devdashboard = pkgs.buildGoModule {
          pname = "devdashboard";
          inherit version vendorHash;

          src = ./core;

          ldflags = [
            "-s"
            "-w"
            "-X main.version=${version}"
          ];

          meta = with pkgs.lib; {
            description = "Repository management and dependency analysis tool for GitHub and GitLab";
            homepage = "https://github.com/greg-hellings/devdashboard";
            license = licenses.mit;
            maintainers = [ ];
            mainProgram = "devdashboard";
          };
        };
      in
      {
        packages = {
          default = devdashboard;
          devdashboard = devdashboard;
        };

        apps = {
          default = {
            type = "app";
            program = "${devdashboard}/bin/devdashboard";
            meta = {
              description = "Run the dashboard CLI, by default";
            };
          };
        };

        devShells.default = pkgs.mkShell {
          buildInputs =
            with pkgs;
            [
              # Go development
              go
              gotools
              gopls
              go-tools
              golint
              gosec
              delve
              golangci-lint

              # Git tools
              git

              # Code formatting and linting
              nixpkgs-fmt

              # Testing and coverage
              go-junit-report

              # Documentation
              nodePackages.markdown-link-check

              # Utilities
              gnumake
              jq
            ]
            ++ lib.optionals pkgs.stdenv.hostPlatform.isDarwin [
              pkgs.apple-sdk_15
            ];

          packages = with pkgs; [
            # Pre-commit hooks
            pre-commit
          ];

          env = {
            GOROOT = "${pkgs.go}/share/go";
            GO111MODULE = "on";
          };

          shellHook = ''
            echo "🚀 DevDashboard development environment"
            echo "======================================"
            echo ""
            echo "Go version: $(go version)"
            echo ""
            echo "Available commands:"
            echo "  make build          - Build the CLI tool"
            echo "  make test           - Run tests"
            echo "  make test-coverage  - Run tests with coverage"
            echo "  make fmt            - Format code"
            echo "  make check          - Run all checks"
            echo "  nix build           - Build with Nix"
            echo "  nix flake check     - Run all checks"
            echo ""
            echo "Run 'pre-commit run --all-files' to check all files."
            echo "Note: In CI, use 'nix develop --command pre-commit run --all-files'"
            echo ""
          '';
        };

        checks =
          let
            # Helper to run tests without re-fetching modules.
            # Reuses the vendored dependencies produced by the devdashboard build.
            commonTest =
              name: pattern:
              pkgs.runCommand name
                {
                  buildInputs = [
                    pkgs.gcc
                    pkgs.go
                    devdashboard
                  ];
                }
                ''
                  export HOME=$(mktemp -d)
                  # Need to create first, else the second copy makes a read-only dir
                  mkdir -p src
                  cp -r ${devdashboard.goModules.outPath}/ src/vendor || true
                  cp -r ${./.}/* ${./.}/.* src
                  cd src
                  export GOFLAGS="-mod=vendor"
                  go ${pattern} 2>&1 | tee $out
                '';
          in
          {
            # Build once
            build = devdashboard;

            # Aggregate test suites, all reuse the same vendored modules
            test = commonTest "devdashboard-tests" "test -v ./pkg/...";
            config-tests = commonTest "devdashboard-config-tests" "test -v ./pkg/config/...";
            dependencies-tests = commonTest "devdashboard-dependencies-tests" "test -v ./pkg/dependencies/...";
            repository-tests = commonTest "devdashboard-repository-tests" "test -v ./pkg/repository/...";

            # Vet using vendored modules
            vet = commonTest "devdashboard-vet" "vet ./...";

            # Formatting check (doesn't need vendored deps)
            fmt =
              pkgs.runCommand "devdashboard-fmt"
                {
                  buildInputs = [ pkgs.go ];
                }
                ''
                  unformatted=$(${pkgs.go}/bin/gofmt -l ${./.})
                  if [ -n "$unformatted" ]; then
                    echo "The following files are not formatted:" > $out
                    echo "$unformatted" >> $out
                    exit 1
                  fi
                  echo "All files are properly formatted" > $out
                '';
          };

        formatter = pkgs.nixpkgs-fmt;
      }
    );
}
