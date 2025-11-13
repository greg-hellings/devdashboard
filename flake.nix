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
        lib = pkgs.lib;
        version = "0.2.0";

        # Common ldflags for reproducible, smaller binaries
        commonLdflags = [
          "-s"
          "-w"
          "-X main.version=${version}"
        ];

        # CLI package build (core module)
        cli = pkgs.buildGoModule {
          pname = "devdashboard";
          inherit version;

          # Build from repository root to allow multi-module sources
          src = ./.;

          # Point at module root containing go.mod
          modRoot = "core";

          # Packages (relative to modRoot) to build/install
          subPackages = [ "cmd/devdashboard" ];

          # Placeholder hash; run `nix build .#cli` to obtain real vendor hash and replace
          vendorHash = "sha256-6tFfPqyoohfhwLSldzAjhUEX5IqwkkIm2XB+YbZo3dQ=";

          ldflags = commonLdflags;

          meta = with lib; {
            description = "DevDashboard CLI - Repository management and dependency analysis tool";
            homepage = "https://github.com/greg-hellings/devdashboard";
            license = licenses.mit;
            maintainers = [ ];
            mainProgram = "devdashboard";
          };
        };

        # GUI desktop package build (gui/desktop module)
        gui = pkgs.buildGoModule {
          pname = "devdashboard-gui";
          inherit version;

          src = ./.;
          modRoot = "gui/desktop";
          subPackages = [ "cmd/devdashboard-gui" ];

          # Replace directive in gui/desktop/go.mod:
          #   replace github.com/greg-hellings/devdashboard/core => ../../core
          # Works because build context root is src=./., and modRoot limits module resolution.
          # Once core is version-tagged, drop the replace and require a version instead.

          vendorHash = "sha256-Q0yOTD3VjCksU1VC9bEM2/AMqyyKvtzLuauNTn4Swic=";

          # pkg-config is required to find libraries during build
          nativeBuildInputs = [ pkgs.pkg-config ];

          # X11 libraries required for Fyne GUI on Linux
          buildInputs = lib.optionals pkgs.stdenv.hostPlatform.isLinux (
            with pkgs;
            [
              xorg.libX11
              xorg.libXcursor
              xorg.libXrandr
              xorg.libXinerama
              xorg.libXi
              xorg.libXxf86vm
              libGL
              libglvnd
            ]
          );

          ldflags = commonLdflags;

          meta = with lib; {
            description = "DevDashboard GUI - Repository management and dependency analysis tool";
            homepage = "https://github.com/greg-hellings/devdashboard";
            license = licenses.mit;
            maintainers = [ ];
            mainProgram = "devdashboard-gui";
          };
        };
      in
      {
        packages = {
          inherit cli gui;
          default = cli;
        };

        # Convenience run target for the CLI
        apps = {
          default = {
            type = "app";
            program = "${cli}/bin/devdashboard";
            meta = {
              description = "Run the DevDashboard CLI";
            };
          };
        };

        # Developer shell with common tools
        devShells = {
          default = pkgs.mkShell {
            buildInputs =
              (with pkgs; [
                go
                gotools
                gopls
                go-tools
                golangci-lint
                golint
                gosec
                delve
                nixpkgs-fmt
                git
                go-junit-report
                nodePackages.markdown-link-check
                gnumake
                jq
                python3
                pkg-config
                shellcheck
              ])
              ++ lib.optionals pkgs.stdenv.hostPlatform.isDarwin [
                pkgs.apple-sdk_15
              ]
              ++ lib.optionals pkgs.stdenv.hostPlatform.isLinux (
                with pkgs;
                [
                  # X11 libraries required for Fyne GUI development on Linux
                  xorg.libX11
                  xorg.libXcursor
                  xorg.libXrandr
                  xorg.libXinerama
                  xorg.libXi
                  xorg.libXxf86vm
                  libGL
                  libglvnd
                ]
              );

            packages = with pkgs; [
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
              echo "Build targets:"
              echo "  nix build .#cli        (CLI)"
              echo "  nix build .#gui        (GUI)"
              echo ""
              echo "Common commands:"
              echo "  make build             - Build CLI"
              echo "  make test              - Run core tests"
              echo "  make fmt               - Format code"
              echo "  make check             - Run all checks"
              echo "  pre-commit run --all-files"
              echo ""
              echo "Replace vendorHash placeholders by running:"
              echo "  nix build .#cli"
              echo "  nix build .#gui"
              echo "and copying reported hashes back into flake.nix."
              echo ""
            '';
          };
        };

        # Expose both builds as checks
        checks = {
          inherit cli gui;
        };

        formatter = pkgs.nixpkgs-fmt;
      }
    );
}
