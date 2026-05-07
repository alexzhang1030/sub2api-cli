#!/usr/bin/env sh
set -eu

REPO="${SUB2API_CLI_REPO:-alexzhang1030/sub2api-cli}"
VERSION="${SUB2API_CLI_VERSION:-latest}"
INSTALL_DIR="${SUB2API_CLI_INSTALL_DIR:-/usr/local/bin}"
BIN_NAME="${SUB2API_CLI_BIN:-sub2api}"

detect_os() {
  os="$(uname -s | tr '[:upper:]' '[:lower:]')"
  case "$os" in
    darwin|linux) printf '%s' "$os" ;;
    msys*|mingw*|cygwin*) printf 'windows' ;;
    *) echo "unsupported OS: $os" >&2; exit 1 ;;
  esac
}

detect_arch() {
  arch="$(uname -m)"
  case "$arch" in
    x86_64|amd64) printf 'amd64' ;;
    arm64|aarch64) printf 'arm64' ;;
    *) echo "unsupported arch: $arch" >&2; exit 1 ;;
  esac
}

need() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "missing required command: $1" >&2
    exit 1
  fi
}

download() {
  url="$1"
  dest="$2"
  if command -v curl >/dev/null 2>&1; then
    curl -fsSL "$url" -o "$dest"
  elif command -v wget >/dev/null 2>&1; then
    wget -qO "$dest" "$url"
  else
    echo "missing required command: curl or wget" >&2
    exit 1
  fi
}

release_tag() {
  if [ "$VERSION" = "latest" ]; then
    download "https://api.github.com/repos/$REPO/releases/latest" "$tmp/latest.json"
    sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' "$tmp/latest.json" | head -n 1
  else
    printf '%s' "$VERSION"
  fi
}

verify_checksum() {
  archive="$1"
  checksums="$2"
  if command -v shasum >/dev/null 2>&1; then
    (cd "$tmp" && grep "  $(basename "$archive")$" "$checksums" | shasum -a 256 -c -)
  elif command -v sha256sum >/dev/null 2>&1; then
    (cd "$tmp" && grep "  $(basename "$archive")$" "$checksums" | sha256sum -c -)
  else
    echo "checksum verification skipped: shasum or sha256sum unavailable" >&2
  fi
}

install_binary() {
  src="$1"
  dest="$INSTALL_DIR/$BIN_NAME"
  mkdir -p "$INSTALL_DIR"
  if [ -w "$INSTALL_DIR" ]; then
    cp "$src" "$dest"
    chmod 0755 "$dest"
  else
    sudo cp "$src" "$dest"
    sudo chmod 0755 "$dest"
  fi
  echo "installed $BIN_NAME to $dest"
}

need uname
need sed
need tar

os="$(detect_os)"
arch="$(detect_arch)"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT INT TERM

tag="$(release_tag)"
if [ -z "$tag" ]; then
  echo "failed to resolve release tag" >&2
  exit 1
fi

asset="sub2api_${tag}_${os}_${arch}.tar.gz"
if [ "$os" = "windows" ]; then
  asset="sub2api_${tag}_${os}_${arch}.zip"
  need unzip
fi

base_url="https://github.com/$REPO/releases/download/$tag"
archive="$tmp/$asset"
checksums="$tmp/checksums.txt"

download "$base_url/$asset" "$archive"
download "$base_url/checksums.txt" "$checksums"
verify_checksum "$archive" "$checksums"

case "$asset" in
  *.zip) unzip -q "$archive" -d "$tmp/extract" ;;
  *.tar.gz) mkdir -p "$tmp/extract"; tar -xzf "$archive" -C "$tmp/extract" ;;
esac

binary="$(find "$tmp/extract" -type f \( -name sub2api -o -name sub2api.exe \) | head -n 1)"
if [ -z "$binary" ]; then
  echo "binary missing in archive" >&2
  exit 1
fi

install_binary "$binary"
