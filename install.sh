#!/bin/sh

set -eu
(set -o pipefail) 2>/dev/null && set -o pipefail

VERSION="${VERSION:-0.1.1}"
REPO="rokuroo171/raind"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
FROM_SOURCE="${FROM_SOURCE:-0}"

usage() {
    cat <<EOF
raind installer

  ./install.sh                 install v${VERSION} from GitHub (linux/darwin amd64/arm64)
  ./install.sh --from-source     build with Go and install
  ./install.sh --help

Environment:
  VERSION       release version (default: ${VERSION})
  INSTALL_DIR   install path (default: ${INSTALL_DIR})
  FROM_SOURCE=1 build from source instead of downloading
EOF
}

need_cmd() {
    if ! command -v "$1" >/dev/null 2>&1; then
        echo "error: required command not found: $1" >&2
        exit 1
    fi
}

install_binary() {
    src=$1
    dest="${INSTALL_DIR}/raind"
    if [ -w "${INSTALL_DIR}" ] 2>/dev/null || [ ! -e "${INSTALL_DIR}" ]; then
        mkdir -p "${INSTALL_DIR}"
        install -m755 "$src" "$dest"
    else
        need_cmd sudo
        sudo install -Dm755 "$src" "$dest"
    fi
    echo "installed to ${dest}"
}

install_from_source() {
    need_cmd go
    echo "building raind from source..."
    CGO_ENABLED=0 go build -o raind .
    install_binary "./raind"
    rm -f ./raind
}

detect_arch() {
    arch=$(uname -m)
    case "$arch" in
        x86_64|amd64)
            echo "amd64"
            ;;
        aarch64|arm64)
            echo "arm64"
            ;;
        *)
            echo "error: unsupported architecture for release install: $arch" >&2
            echo "       supported: x86_64 (amd64), aarch64/arm64" >&2
            echo "       try: ./install.sh --from-source" >&2
            exit 1
            ;;
    esac
}

install_from_release() {
    need_cmd curl
    need_cmd tar

    os=$(uname -s)
    case "$os" in
        Linux)  goos="linux"  ;;
        Darwin) goos="darwin" ;;
        *)
            echo "error: release install supports linux and darwin only" >&2
            echo "       on this system use: ./install.sh --from-source" >&2
            exit 1
            ;;
    esac

    goarch=$(detect_arch)
    tarball="raind_${VERSION}_${goos}_${goarch}.tar.gz"
    url="https://github.com/${REPO}/releases/download/v${VERSION}/${tarball}"

    tmp=$(mktemp -d)
    trap 'rm -rf "$tmp"' EXIT INT HUP

    echo "detected: ${goos}/${goarch}"
    echo "downloading raind v${VERSION}..."
    curl -fsSL -o "${tmp}/${tarball}" "$url"

    tar -xzf "${tmp}/${tarball}" -C "$tmp"
    if [ ! -f "${tmp}/raind" ]; then
        echo "error: expected binary raind not found in ${tarball}" >&2
        exit 1
    fi

    install_binary "${tmp}/raind"
    echo "run: raind"
}

for arg in "$@"; do
    case "$arg" in
        --from-source) FROM_SOURCE=1 ;;
        -h|--help)
            usage
            exit 0
            ;;
        *)
            echo "error: unknown option: $arg" >&2
            usage >&2
            exit 1
            ;;
    esac
done

if [ "$FROM_SOURCE" = "1" ]; then
    install_from_source
else
    install_from_release
fi
