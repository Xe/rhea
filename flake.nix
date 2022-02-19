{
  description = "The rhea gemini server";

  # Nixpkgs / NixOS version to use.
  inputs.nixpkgs.url = "nixpkgs/nixos-unstable";

  outputs = { self, nixpkgs }:
    let

      # Generate a user-friendly version number.
      version = builtins.substring 0 8 self.lastModifiedDate;

      # System types to support.
      supportedSystems =
        [ "x86_64-linux" "x86_64-darwin" "aarch64-linux" "aarch64-darwin" ];

      # Helper function to generate an attrset '{ x86_64-linux = f "x86_64-linux"; ... }'.
      forAllSystems = nixpkgs.lib.genAttrs supportedSystems;

      # Nixpkgs instantiated for supported system types.
      nixpkgsFor = forAllSystems (system: import nixpkgs { inherit system; });
    in {

      # Provide some binary packages for selected system types.
      packages = forAllSystems (system:
        let pkgs = nixpkgsFor.${system};
        in {
          rhea = pkgs.buildGoModule {
            pname = "rhea";
            inherit version;
            src = ./.;
            vendorSha256 = null;
          };
        });

      # The default package for 'nix build'. This makes sense if the
      # flake provides only one package or there is a clear "main"
      # package.
      defaultPackage = forAllSystems (system: self.packages.${system}.rhea);

      nixosModule = forAllSystems (system:
        let pkgs = nixpkgsFor.${system};
        in { config, lib, pkgs, ... }:

        with lib;
        let
          cfg = config.within.rhea;

          files = types.submodule {
            options = {
              root = mkOption {
                type = types.str;
                example = "/srv/gemini/cetacean.club";
                description = "gemini root";
              };

              userPaths =
                mkEnableOption "Enables ~user for ~user/public_gemini";
              autoIndex = mkEnableOption "Enables automatic index creation";
            };
          };

          filesToJSON = files: {
            root = files.root;
            user_paths = files.userPaths;
            auto_index = files.autoIndex;
          };

          reverseProxy = types.submodule {
            options = {
              domain = mkOption {
                type = types.str;
                example = "cetacean.club";
                description =
                  "Domain to use with the remote host after proxying the request";
              };

              to = mkOption {
                type = types.listOf types.str;
                example = ''[ "cetacean.club:1965"]'';
                description =
                  "list of host:port sets of backend servers for this site";
              };
            };
          };

          reverseProxyToJSON = rp: {
            domain = rp.domain;
            to = rp.to;
          };

          site = types.submodule {
            options = {
              domain = mkOption {
                type = types.str;
                example = "cetacean.club";
                description = "domain name for this site";
              };

              certPath = mkOption {
                type = types.str;
                example = "/srv/within/certs/cetacean.club/cert.pem";
                description = "certificate path";
              };

              keyPath = mkOption {
                type = types.str;
                example = "/srv/within/keys/cetacean.club/key.pem";
                description = "key path";
              };

              files = mkOption {
                type = types.nullOr files;
                default = null;
                description = "files to serve for this site";
              };

              reverseProxy = mkOption {
                type = types.nullOr reverseProxy;
                default = null;
                description = "reverse proxy target";
              };
            };
          };

          siteToJSON = site: {
            domain = site.domain;
            cert_path = site.certPath;
            key_path = site.keyPath;
            files =
              if site.files != null then (filesToJSON site.files) else null;
            reverse_proxy = if site.reverseProxy != null then
              (reverseProxyToJSON site.reverseProxy)
            else
              null;
          };

          configToJSON = cfg: {
            port = cfg.port;
            http_port = cfg.httpPort;
            sites = map siteToJSON cfg.sites;
          };

        in {
          options.within.rhea = {
            enable = mkEnableOption "Rhea gemini server";

            package = mkOption {
              type = types.package;
              default = self.packages.${system}.rhea;
              description = "rhea package to use";
            };

            port = mkOption {
              type = types.port;
              default = 1965;
              description = "port to serve Gemini on";
            };

            httpPort = mkOption {
              type = types.port;
              default = 23818;
              description = "port to serve prometheus metrics (http) on";
            };

            sites = mkOption {
              type = types.listOf site;
              description = "gemini sites to serve on this machine";
            };
          };

          config = mkIf cfg.enable {
            systemd.services.rhea = {
              description = "Rhea Gemini server";
              wantedBy = [ "multi-user.target" ];

              serviceConfig = {
                ExecStart = "${cfg.package}/bin/rhea -config ${
                    builtins.toFile "config.json"
                    (builtins.toJSON (configToJSON cfg))
                  }";
                ProtectHome = "read-only";
                Restart = "on-failure";
                Type = "notify";
              };
            };
          };
        });

      devShell = forAllSystems (system:
        let pkgs = nixpkgsFor.${system};
        in with pkgs;
        mkShell {
          buildInputs =
            [ go goimports gopls sqliteInteractive pkg-config minica ];
        });
    };
}
