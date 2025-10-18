{
  description = "DevDashboard - Repository management and dependency analysis tool";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
    pre-commit-hooks = {
      url = "github:cachix/pre-commit-hooks.nix";
      inputs.nixpkgs.follows = "nixpkgs";
    };
  };

  outputs =
    { self
    , nixpkgs
    , flake-utils
    , pre-commit-hooks
    ,
    }:
    flake-utils.lib.eachDefaultSystem (
      system:
      let
        pkgs = import nixpkgs { inherit system; };

        version = "0.2.0";

        vendorHash = "sha256-icE99gXkDYrKEmre+W+MWMFe+dfNLcjANUiVVrmBVWM=";

        devdashboard = pkgs.buildGoModule {
          pname = "devdashboard";
          inherit version vendorHash;

          src = ./.;

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

        pre-commit-check = pre-commit-hooks.lib.${system}.run {
          src = ./.;
          hooks = {
            # Go formatting
            gofmt = {
              enable = true;
              name = "gofmt";
              entry = "${pkgs.go}/bin/gofmt -w";
              files = "\\.go$";
            };

            # Go imports
            goimports = {
              enable = true;
              name = "goimports";
              entry = "${pkgs.gotools}/bin/goimports -w";
              files = "\\.go$";
            };

            # Go vet
            govet = {
              enable = true;
              name = "go vet";
              entry = "${pkgs.go}/bin/go vet ./...";
              files = "\\.go$";
              pass_filenames = false;
            };

            # Go mod tidy
            gomod-tidy = {
              enable = true;
              name = "go mod tidy";
              entry = "${pkgs.go}/bin/go mod tidy";
              files = "go\\.(mod|sum)$";
              pass_filenames = false;
            };

            # Nix formatting
            nixpkgs-fmt = {
              enable = true;
              entry = "${pkgs.nixpkgs-fmt}/bin/nixpkgs-fmt";
            };

            # Trailing whitespace
            trailing-whitespace = {
              enable = true;
              name = "trim trailing whitespace";
              entry = "${pkgs.python3Packages.pre-commit-hooks}/bin/trailing-whitespace-fixer";
              types = [ "text" ];
            };

            # End of file fixer
            end-of-file-fixer = {
              enable = true;
              name = "fix end of files";
              entry = "${pkgs.python3Packages.pre-commit-hooks}/bin/end-of-file-fixer";
              types = [ "text" ];
            };

            # YAML check
            check-yaml = {
              enable = true;
              name = "check yaml";
              entry = "${pkgs.python3Packages.pre-commit-hooks}/bin/check-yaml";
              files = "\\.ya?ml$";
            };

            # Markdown link check
            markdown-link-check = {
              enable = true;
              name = "markdown link check";
              entry = "${pkgs.nodePackages.markdown-link-check}/bin/markdown-link-check";
              files = "\\.md$";
            };
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
          buildInputs = with pkgs; [
            # Go development
            go
            gotools
            gopls
            go-tools
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
          ];

          packages = with pkgs; [
            # Pre-commit hooks
            pre-commit
          ];

          env = {
            GOROOT = "${pkgs.go}/share/go";
            GO111MODULE = "on";
          };

          shellHook = pre-commit-check.shellHook + ''
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
            echo "Pre-commit hooks are auto-installed in this shell."
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
