{
  description = "CEC-MQTT";

  inputs.nixpkgs.url = "nixpkgs/nixos-25.05";
  # Use older version of nixos for libcec
  inputs.nixpkgs_2411.url = "nixpkgs/nixos-24.11";

  outputs = {
    self,
    nixpkgs,
    nixpkgs_2411,
  }: let
    lastModifiedDate = self.lastModifiedDate or self.lastModified or "19700101";
    version = builtins.substring 0 8 lastModifiedDate;
    supportedSystems = ["x86_64-linux" "x86_64-darwin" "aarch64-linux" "aarch64-darwin"];
    forAllSystems = nixpkgs.lib.genAttrs supportedSystems;
    nixpkgsFor = forAllSystems (
      system: let
        pkgs2411 = import nixpkgs_2411 {inherit system;};
      in
        import nixpkgs {
          inherit system;
          overlays = [
            (final: prev: {
              libcec = pkgs2411.libcec;
              libcec_platform = pkgs2411.libcec_platform;
            })
          ];
        }
    );
  in {
    packages = forAllSystems (system: let
      pkgs = nixpkgsFor.${system};
    in {
      cec-mqtt = pkgs.buildGoModule {
        pname = "cec-mqtt";
        inherit version;
        src = ./.;
        nativeBuildInputs = [pkgs.pkg-config];
        buildInputs = [pkgs.libcec pkgs.libcec_platform];
        vendorHash = "sha256-uGhfR93Tj3EK0vqwgxGrAFW8XTLC8eQci0wPpSZ0b7U=";
      };
    });

    devShells = forAllSystems (system: let
      pkgs = nixpkgsFor.${system};
    in {
      default = pkgs.mkShell {
        buildInputs = with pkgs; [
          go
          gopls
          gotools
          go-tools
          go-outline
          godef
          delve
          mqttui
          libcec
          libcec_platform
          pkg-config
          glibc.static
        ];
      };
    });

    defaultPackage = forAllSystems (system: self.packages.${system}.cec-mqtt);
  };
}
