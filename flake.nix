{
  description = "vmTUI for local QEMU virtual machines";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
  };

  outputs =
    { self, nixpkgs, ... }:
    let
      system = "x86_64-linux";
      pkgs = import nixpkgs { inherit system; };
      lib = pkgs.lib;

      vmtuiSrc = lib.cleanSourceWith {
        src = self;
        filter =
          path: type:
          let
            base = baseNameOf path;
          in
          !(base == "result" || base == ".cache");
      };

      vmtui-bin = pkgs.buildGoModule {
        pname = "vmtui";
        version = "0.1.0";
        src = vmtuiSrc;
        vendorHash = "sha256-pDWrBm5KANLz5GFEaJZupBM3d1hd9F6Vh0fHOmd1p5k=";
        env.CGO_ENABLED = 0;
        enableParallelBuilding = false;
        ldflags = [
          "-s"
          "-w"
        ];
      };

      vmtui = pkgs.symlinkJoin {
        name = "vmtui";
        paths = [ vmtui-bin ];
        nativeBuildInputs = [ pkgs.makeWrapper ];
        postBuild = ''
          wrapProgram "$out/bin/vmtui" \
            --prefix PATH : ${
              lib.makeBinPath [
                pkgs.aria2
                pkgs.qemu_kvm
              ]
            }
        '';
      };

    in
    {
      formatter.${system} = pkgs.nixfmt;

      packages.${system} = {
        inherit vmtui;
        default = vmtui;
      };

      apps.${system} = {
        default = {
          type = "app";
          program = "${vmtui}/bin/vmtui";
        };

        vmtui = {
          type = "app";
          program = "${vmtui}/bin/vmtui";
        };
      };

      devShells.${system}.default = pkgs.mkShell {
        packages = [
          pkgs.aria2
          pkgs.go
          pkgs.qemu_kvm
        ];

        shellHook = ''
          mkdir -p .cache/go-build .cache/go-mod
          echo "vmtui dev shell"
          echo "  run: go run ."
          echo "  fmt: nix fmt"
        '';
      };
    };
}
