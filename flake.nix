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

        version = "0.1.0";

        devdashboard = pkgs.buildGoModule {
          pname = "devdashboard";
          inherit version;

          src = ./.;

          vendorHash = "sha256-SFjQWmj2qztwsGOtGVBSxbaxztYU16PaU8aS/H0L/hk=";

          subPackages = [ "cmd/devdashboard" ];

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
              entry = "${pkgs.go}/bin/go vet ./pkg/... ./cmd/...";
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
            echo "Pre-commit hooks are installed. Run 'pre-commit run --all-files' to check."
            echo ""
          '';
        };

        checks = {
          # Build check
          build = devdashboard;

          # Test check
          test =
            pkgs.runCommand "devdashboard-tests"
              {
                buildInputs = [ pkgs.go ];
              }
              ''
                cd ${./.}
                export HOME=$(mktemp -d)
                export GOCACHE=$HOME/.cache/go-build
                export GOMODCACHE=$HOME/go/pkg/mod
                ${pkgs.go}/bin/go test -v ./pkg/... > $out
              '';

          # Go vet check
          vet =
            pkgs.runCommand "devdashboard-vet"
              {
                buildInputs = [ pkgs.go ];
              }
              ''
                cd ${./.}
                export HOME=$(mktemp -d)
                ${pkgs.go}/bin/go vet ./pkg/... ./cmd/... 2>&1 | tee $out
              '';

          # Formatting check
          fmt =
            pkgs.runCommand "devdashboard-fmt"
              {
                buildInputs = [ pkgs.go ];
              }
              ''
                cd ${./.}
                unformatted=$(${pkgs.go}/bin/gofmt -l .)
                if [ -n "$unformatted" ]; then
                  echo "The following files are not formatted:" > $out
                  echo "$unformatted" >> $out
                  exit 1
                fi
                echo "All files are properly formatted" > $out
              '';

          # Dependencies check
          config-tests =
            pkgs.runCommand "devdashboard-config-tests"
              {
                buildInputs = [ pkgs.go ];
              }
              ''
                cd ${./.}
                export HOME=$(mktemp -d)
                export GOCACHE=$HOME/.cache/go-build
                export GOMODCACHE=$HOME/go/pkg/mod
                ${pkgs.go}/bin/go test -v ./pkg/config/... > $out
              '';

          dependencies-tests =
            pkgs.runCommand "devdashboard-dependencies-tests"
              {
                buildInputs = [ pkgs.go ];
              }
              ''
                cd ${./.}
                export HOME=$(mktemp -d)
                export GOCACHE=$HOME/.cache/go-build
                export GOMODCACHE=$HOME/go/pkg/mod
                ${pkgs.go}/bin/go test -v ./pkg/dependencies/... > $out
              '';

          repository-tests =
            pkgs.runCommand "devdashboard-repository-tests"
              {
                buildInputs = [ pkgs.go ];
              }
              ''
                cd ${./.}
                export HOME=$(mktemp -d)
                export GOCACHE=$HOME/.cache/go-build
                export GOMODCACHE=$HOME/go/pkg/mod
                ${pkgs.go}/bin/go test -v ./pkg/repository/... > $out
              '';
        };

        formatter = pkgs.nixpkgs-fmt;
      }
    );
}
