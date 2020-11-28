{ pkgs ? import <nixpkgs> { } }:

pkgs.mkShell {
  buildInputs = with pkgs; [ go goimports gopls sqliteInteractive pkg-config minica ];
}
