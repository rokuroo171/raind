# Releasing raind

How a release flows from tag to downstream packages, including the packaging
targets that publish after the GitHub Releases page. nixpkgs is the first
downstream target, AUR is second, and the procedure is written so more targets
(Homebrew, etc.) can be appended.

## Priming a new release target

A new target needs three things before the first release hits it:

1. Package metadata checked into this repo (`nix/` for nixpkgs, `packaging/`
   for AUR), so test builds are reproducible from a tag.
2. A step in this document under "After the release".
3. Someone to own the publish. rokuroo171 is the maintainer for all targets.

## The release itself

The existing flow stays unchanged: tag `v0.2.0` on `main`, GitHub Actions runs
GoReleaser (`.goreleaser.yml`), and artifacts land on the GitHub Releases page.
Installers pull from there.

## After the release

Run these in order. Each step is a test of the previous one, so a broken
nixpkgs hash or AUR source is caught before the next target is touched.

### 1. nixpkgs (first)

1. Bump `version` in `nix/raind.nix` to the new tag without the `v`.
2. Update `src.hash`:
   ```bash
   nix-prefetch-url --unpack "https://github.com/rokuroo171/raind/archive/v0.2.0.tar.gz"
   nix hash convert --hash-algo sha256 --to sri <printed hash>
   ```
   Paste the SRI hash into `nix/raind.nix`.
3. Update `vendorHash`: set it to `lib.fakeHash`, run the local build below,
   and paste the hash nix prints into the file.
4. Test locally:
   ```bash
   nix flake check
   nix build .#raind
   nix run .#raind
   ```
   Every contributor on NixOS runs this before opening the package PR.
5. Open the nixpkgs PR: copy `nix/raind.nix` to
   `pkgs/by-name/ra/raind/default.nix`, add rokuroo171 to
   `pkgs/maintainers/maintainer-list.nix`, and add the package to
   `pkgs/top-level/all-packages.nix`.

### 2. AUR (second)

1. Bump `pkgver` in `packaging/aur/PKGBUILD` to the new tag (keep `pkgrel`
   unless the package metadata changed).
2. Refresh checksums:
   ```bash
   cd packaging/aur && updpkgsums
   ```
   Replace the `SKIP` placeholder with the real hash (AUR rejects `SKIP` for
   release builds).
3. Test locally:
   ```bash
   makepkg -si
   namcap PKGBUILD
   ```
4. Publish the AUR package. The AUR repo is
   `ssh://aur@aur.archlinux.org/raind.git`; push the new PKGBUILD there.
   `pkgver` must match the released tag exactly.

### 3. Future targets

Append here when adding Homebrew, a snap, or another channel. Keep the rule:
nixpkgs always publishes first, because it exercises the source tarball and
build flags earliest.

## Maintainers

- rokuroo171: maintainer, all targets.