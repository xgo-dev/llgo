{
  description = "LLGo development environment";

  # Nix develop obtains Bash from the nixpkgs input. Intel macOS needs 26.05.
  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-26.05-darwin";
  inputs.nixpkgsUnstable.url = "github:NixOS/nixpkgs/nixpkgs-unstable";

  outputs = { nixpkgs, nixpkgsUnstable, ... }:
    let
      systems = [
        "x86_64-linux"
        "aarch64-linux"
        "x86_64-darwin"
        "aarch64-darwin"
      ];
      forAllSystems = f: nixpkgs.lib.genAttrs systems (system:
        f (import (if system == "x86_64-darwin" then nixpkgs else nixpkgsUnstable) { inherit system; }));
    in
    {
      devShells = forAllSystems (pkgs:
        let
          llvmPackages = pkgs.llvmPackages_22;
          llvm = llvmPackages.llvm;
        in
        {
          default = pkgs.mkShell {
            packages = [
              pkgs.git
              pkgs.go_1_27
              llvm
              llvm.dev
              llvm.lib
              llvmPackages.clang
              llvmPackages.lld
              pkgs.pkg-config
              pkgs.boehmgc
              pkgs.libffi
              pkgs.openssl
              pkgs.zlib
              pkgs.cjson
              pkgs.sqlite
              pkgs.libuv
            ];

            # The LLVM Go bindings otherwise use Homebrew/apt-specific paths.
            GOFLAGS = "-tags=byollvm";
            CGO_ENABLED = "1";

            shellHook = ''
              export CC=clang
              export CXX=clang++
              export LLVM_CONFIG="$(command -v llvm-config)"
              export CGO_CPPFLAGS="$($LLVM_CONFIG --cppflags)"
              export CGO_CXXFLAGS="-std=c++17"
              export CGO_LDFLAGS="$($LLVM_CONFIG --ldflags --link-shared --libs all --system-libs)"
              export LLGO_ROOT="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"
            '' + pkgs.lib.optionalString pkgs.stdenv.hostPlatform.isLinux ''
              # LLGo's generated programs can load libatomic through LLVM at runtime.
              export LD_LIBRARY_PATH="${pkgs.stdenv.cc.cc.lib}/lib''${LD_LIBRARY_PATH:+:$LD_LIBRARY_PATH}"
            '';
          };
        });
    };
}
