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

      vmctlSrc = lib.cleanSourceWith {
        src = self;
        filter =
          path: type:
          let
            base = baseNameOf path;
          in
          !(base == "result" || base == ".cache");
      };

      vmctl-bin = pkgs.buildGoModule {
        pname = "vmctl";
        version = "0.1.0";
        src = vmctlSrc;
        vendorHash = "sha256-pDWrBm5KANLz5GFEaJZupBM3d1hd9F6Vh0fHOmd1p5k=";
        env.CGO_ENABLED = 0;
        enableParallelBuilding = false;
        ldflags = [
          "-s"
          "-w"
        ];
      };

      vmctl = pkgs.symlinkJoin {
        name = "vmctl";
        paths = [ vmctl-bin ];
        nativeBuildInputs = [ pkgs.makeWrapper ];
        postBuild = ''
          wrapProgram "$out/bin/vmctl" \
            --prefix PATH : ${
              lib.makeBinPath [
                pkgs.aria2
                pkgs.qemu_kvm
              ]
            }
        '';
      };

      vmctl-run = pkgs.writeShellApplication {
        name = "vmctl";
        runtimeInputs = [
          pkgs.aria2
          pkgs.babashka
          pkgs.go
          pkgs.qemu_kvm
        ];
        text = ''
          VMCTL_SRC="${vmctlSrc}" bb ${./vmctl-run.bb} "$@"
        '';
      };
    in
    {
      formatter.${system} = pkgs.nixfmt;

      packages.${system} = {
        inherit vmctl;
        default = vmctl;
      };

      apps.${system} = {
        default = {
          type = "app";
          program = "${vmctl-run}/bin/vmctl";
        };

        vmctl = {
          type = "app";
          program = "${vmctl-run}/bin/vmctl";
        };
      };

      devShells.${system}.default = pkgs.mkShell {
        packages = [
          pkgs.aria2
          pkgs.babashka
          pkgs.go
          pkgs.gopls
          pkgs.nixfmt
          pkgs.qemu_kvm
        ];

        shellHook = ''
          bb ${./vmctl-shellhook.bb}
        '';
      };
    };
}
