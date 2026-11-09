#!/bin/sh
# Install a clablings release binary on Linux.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/tianyuxue/clablings/main/scripts/install.sh | sh
#   ./scripts/install.sh --version v0.1.0
#   ./scripts/install.sh --bindir "$HOME/bin"
#   ./scripts/install.sh --uninstall
#
# Environment:
#   CLABLINGS_REPO     GitHub owner/repo (default: tianyuxue/clablings)
#   CLABLINGS_VERSION  Release tag, or "latest" (default: latest)
#   CLABLINGS_BASE_URL Override the download base URL (mirrors/testing)
#   PREFIX             Install prefix (default: ~/.local)
#   BINDIR             Exact install directory (default: $PREFIX/bin)

set -eu

REPO="${CLABLINGS_REPO:-tianyuxue/clablings}"
VERSION="${CLABLINGS_VERSION:-latest}"
BASE_URL="${CLABLINGS_BASE_URL:-}"
PREFIX="${PREFIX:-$HOME/.local}"
BINDIR="${BINDIR:-$PREFIX/bin}"

usage() {
	cat <<'EOF'
Install clablings from GitHub Releases.

Usage: install.sh [options]

Options:
  --version VERSION  Release tag (for example v0.1.0) or "latest"
  --bindir DIR       Install directory (default: $PREFIX/bin)
  --uninstall        Remove the installed clablings binary
  -h, --help         Show this help

Environment:
  CLABLINGS_REPO     GitHub owner/repo (default: tianyuxue/clablings)
  CLABLINGS_VERSION  Release tag, or "latest" (default: latest)
  CLABLINGS_BASE_URL Override the download base URL (mirrors/testing)
  PREFIX             Install prefix (default: ~/.local)
  BINDIR             Exact install directory (default: $PREFIX/bin)
EOF
}

while [ "$#" -gt 0 ]; do
	case "$1" in
		--version)
			[ "$#" -ge 2 ] || { echo "install.sh: --version needs a value" >&2; exit 2; }
			VERSION="$2"
			shift 2
			;;
		--bindir)
			[ "$#" -ge 2 ] || { echo "install.sh: --bindir needs a value" >&2; exit 2; }
			BINDIR="$2"
			shift 2
			;;
		--uninstall)
			rm -f "$BINDIR/clablings"
			echo "removed $BINDIR/clablings"
			exit 0
			;;
		-h|--help)
			usage
			exit 0
			;;
		*)
			echo "install.sh: unknown option: $1" >&2
			usage >&2
			exit 2
			;;
	esac
done

case "$VERSION" in
	latest|v*) ;;
	*) VERSION="v$VERSION" ;;
esac

need() {
	command -v "$1" >/dev/null 2>&1 || {
		echo "install.sh: required command not found: $1" >&2
		exit 1
	}
}

need uname
need tar
need install
need grep

os="$(uname -s)"
if [ "$os" != "Linux" ]; then
	echo "install.sh: clablings currently ships Linux binaries only (found $os)" >&2
	exit 1
fi

case "$(uname -m)" in
	x86_64|amd64) arch="amd64" ;;
	aarch64|arm64) arch="arm64" ;;
	*)
		echo "install.sh: unsupported architecture: $(uname -m)" >&2
		exit 1
		;;
esac

asset="clablings_linux_${arch}.tar.gz"
if [ -n "$BASE_URL" ]; then
	base="${BASE_URL%/}"
elif [ "$VERSION" = "latest" ]; then
	base="https://github.com/${REPO}/releases/latest/download"
else
	base="https://github.com/${REPO}/releases/download/${VERSION}"
fi

tmp="$(mktemp -d "${TMPDIR:-/tmp}/clablings-install.XXXXXX")"
trap 'rm -rf "$tmp"' EXIT

download() {
	if command -v curl >/dev/null 2>&1; then
		curl -fsSL "$1" -o "$2"
	elif command -v wget >/dev/null 2>&1; then
		wget -qO "$2" "$1"
	else
		echo "install.sh: curl or wget is required" >&2
		exit 1
	fi
}

echo "downloading ${asset} (${VERSION}) from ${REPO}"
download "${base}/${asset}" "${tmp}/${asset}"
download "${base}/checksums.txt" "${tmp}/checksums.txt"

if command -v sha256sum >/dev/null 2>&1; then
	(cd "$tmp" && grep "  ${asset}$" checksums.txt | sha256sum -c -)
elif command -v shasum >/dev/null 2>&1; then
	(cd "$tmp" && grep "  ${asset}$" checksums.txt | shasum -a 256 -c -)
else
	echo "install.sh: sha256sum or shasum is required to verify the download" >&2
	exit 1
fi

tar -xzf "${tmp}/${asset}" -C "$tmp"
mkdir -p "$BINDIR"
install -m 0755 "${tmp}/clablings_linux_${arch}/clablings" "${BINDIR}/clablings"

echo "installed ${BINDIR}/clablings"
case ":${PATH}:" in
	*":${BINDIR}:"*) ;;
	*) echo "note: add ${BINDIR} to PATH to run clablings from any directory" ;;
esac
case "${BINDIR}" in
	"${HOME}"/*)
		echo "note: sudo resets PATH; for root runs use 'clablings --sudo' or --bindir /usr/local/bin" ;;
esac
