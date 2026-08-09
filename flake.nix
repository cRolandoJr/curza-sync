{
  description = "curza-sync — baja el material de PEDCO y lo deja en un archivo por unidad";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

  outputs = { self, nixpkgs }:
    let
      forAllSystems = nixpkgs.lib.genAttrs [ "x86_64-linux" "aarch64-linux" ];
    in
    {
      devShells = forAllSystems (system:
        let pkgs = nixpkgs.legacyPackages.${system};
        in {
          default = pkgs.mkShell {
            # pandoc y poppler-utils son dependencias de RUNTIME: convert los invoca.
            # Van acá para no tocar la config del sistema.
            packages = with pkgs; [ go gopls pandoc poppler-utils ];
            # libreoffice NO: ya está en el sistema (soffice en PATH) y son ~1 GB.
          };
        });
    };
}
