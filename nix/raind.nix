{ lib, buildGoModule, fetchFromGitHub }:

# nixpkgs-ready package expression.
# For a real nixpkgs PR this file becomes pkgs/by-name/ra/raind/default.nix,
# and the maintainer needs an entry in pkgs/maintainers/maintainer-list.nix.
buildGoModule rec {
  pname = "raind";
  version = "0.2.0";

  src = fetchFromGitHub {
    owner = "rokuroo171";
    repo = "raind";
    rev = "v${version}";
    hash = "sha256-zH5RP1mCxLdyqcDLlMG8zwodsEsez2XePtiWO+m/ST4=";
  };

  vendorHash = "sha256-8UprJXRLFO3giWAm8k+vbNz7HPYwKW7cD36qc3hEkzE=";

  meta = with lib; {
    description = "Terminal weather screensaver with four modes: rain, thunder, snow, meteor";
    homepage = "https://github.com/rokuroo171/raind";
    license = licenses.mit;
    platforms = platforms.linux ++ platforms.darwin;
    maintainers = with maintainers; [
      # maintainer: rokuroo171
    ];
    mainProgram = "raind";
  };
}
