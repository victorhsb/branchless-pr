#!/usr/bin/env sh
set -eu

REPO="victorhsb/branchless-pr"
BINARY="bpr"
SYMLINK="stack-pr"

BOLD="$(tput bold 2>/dev/null || printf '')"
RED="$(tput setaf 1 2>/dev/null || printf '')"
GREEN="$(tput setaf 2 2>/dev/null || printf '')"
YELLOW="$(tput setaf 3 2>/dev/null || printf '')"
NO_COLOR="$(tput sgr0 2>/dev/null || printf '')"

usage() {
	cat <<EOF
${BOLD}install.sh${NO_COLOR} — Install ${BINARY} (and ${SYMLINK} symlink) from GitHub releases.

Usage: install.sh [options]

Options:
  --dir DIR       Install directory (default: \$HOME/.local/bin)
  --version TAG   Install a specific version (default: latest release)
  -y, --yes       Skip confirmation prompt
  -h, --help      Show this help

Environment variables:
  INSTALL_DIR     Same as --dir
  INSTALL_VERSION  Same as --version
EOF
}

err() {
	printf '%s%s%s\n' "${RED}${BOLD}" "error: ${1}" "${NO_COLOR}" >&2
	exit 1
}

info() {
	printf '%s\n' "${1}"
}

success() {
	printf '%s%s%s\n' "${GREEN}${BOLD}" "${1}" "${NO_COLOR}"
}

warn() {
	printf '%s%s%s\n' "${YELLOW}${BOLD}" "warning: ${1}" "${NO_COLOR}" >&2
}

need_cmd() {
	if ! command -v "$1" >/dev/null 2>&1; then
		err "need '$1' (command not found)"
	fi
}

detect_os() {
	_os="$(uname -s | tr '[:upper:]' '[:lower:]')"
	case "$_os" in
		linux) printf "linux" ;;
		darwin) printf "darwin" ;;
		mingw*|msys*|cygwin*|windows_nt) printf "windows" ;;
		*) err "unsupported OS: $_os" ;;
	esac
}

detect_arch() {
	_arch="$(uname -m)"
	case "$_arch" in
		x86_64|amd64) printf "amd64" ;;
		aarch64|arm64) printf "arm64" ;;
		*) err "unsupported architecture: $_arch" ;;
	esac
}

detect_extension() {
	case "$(detect_os)" in
		windows) printf "zip" ;;
		*) printf "tar.gz" ;;
	esac
}

download() {
	_url="$1"
	_file="$2"

	if command -v curl >/dev/null 2>&1; then
		curl --proto '=https' --tlsv1.2 -fSL -o "$_file" "$_url"
	elif command -v wget >/dev/null 2>&1; then
		wget --https-only --secure-protocol=TLSv1_2 -O "$_file" "$_url"
	else
		err "need curl or wget to download files"
	fi
}

latest_version() {
	_url="https://github.com/${REPO}/releases/latest"
	if command -v curl >/dev/null 2>&1; then
		curl --proto '=https' --tlsv1.2 -fLS -o /dev/null -w '%{url_effective}' "$_url" \
			| sed 's|.*/tag/||' | sed 's|^v||'
	elif command -v wget >/dev/null 2>&1; then
		wget --https-only --secure-protocol=TLSv1_2 -O /dev/null "$_url" 2>&1 \
			| grep -o 'Location: .*' | tail -1 | sed 's|.*/tag/||' | sed 's|^v||' | tr -d '\r'
	else
		err "need curl or wget to discover latest version"
	fi
}

verify_checksum() {
	_archive="$1"
	_checksums_url="$2"

	if ! command -v sha256sum >/dev/null 2>&1; then
		if ! command -v shasum >/dev/null 2>&1; then
			warn "no sha256sum or shasum found; skipping checksum verification"
			return 0
		fi
	fi

	_checksum_file="${_archive}.checksums"
	download "$_checksums_url" "$_checksum_file" 2>/dev/null || {
		warn "could not download checksums.txt; skipping checksum verification"
		return 0
	}

	_basename="$(basename "$_archive")"
	_expected="$(grep "$_basename" "$_checksum_file" | awk '{print $1}')"
	if [ -z "$_expected" ]; then
		warn "archive not found in checksums.txt; skipping checksum verification"
		return 0
	fi

	if command -v sha256sum >/dev/null 2>&1; then
		_actual="$(sha256sum -b "$_archive" | awk '{print $1}')"
	else
		_actual="$(shasum -a 256 -b "$_archive" | awk '{print $1}')"
	fi

	if [ "$_expected" != "$_actual" ]; then
		err "checksum mismatch for $_basename\n  expected: $_expected\n  actual:   $_actual"
	fi

	info "Checksum verified."
}

confirm() {
	_msg="$1"
	printf '%s [y/N] ' "$_msg"
	read -r _answer < /dev/tty || _answer=""
	case "$_answer" in
		y*|Y*) return 0 ;;
		*) err "aborted" ;;
	esac
}

INSTALL_DIR="${INSTALL_DIR:-${HOME}/.local/bin}"
INSTALL_VERSION="${INSTALL_VERSION:-}"
SKIP_CONFIRM=0

while [ $# -gt 0 ]; do
	case "$1" in
		--dir) INSTALL_DIR="$2"; shift 2 ;;
		--version) INSTALL_VERSION="$2"; shift 2 ;;
		-y|--yes) SKIP_CONFIRM=1; shift ;;
		-h|--help) usage; exit 0 ;;
		*) err "unknown option: $1" ;;
	esac
done

need_cmd uname
need_cmd mktemp

_os="$(detect_os)"
_arch="$(detect_arch)"
_ext="$(detect_extension)"

if [ -z "$INSTALL_VERSION" ]; then
	need_cmd curl
	INSTALL_VERSION="$(latest_version)"
fi

_archive_name="branchless-pr_${INSTALL_VERSION}_${_os}_${_arch}.${_ext}"
_download_url="https://github.com/${REPO}/releases/download/v${INSTALL_VERSION}/${_archive_name}"
_checksums_url="https://github.com/${REPO}/releases/download/v${INSTALL_VERSION}/checksums.txt"

_dst_bin="${INSTALL_DIR}/${BINARY}"
_dst_symlink="${INSTALL_DIR}/${SYMLINK}"

info ""
info "${BOLD}branchless-pr installer${NO_COLOR}"
info "  version:   v${INSTALL_VERSION}"
info "  platform: ${_os}/${_arch}"
info "  install:  ${INSTALL_DIR}"
info "  binary:   ${_dst_bin}"
info "  symlink:  ${_dst_symlink} -> ${BINARY}"
info ""

if [ "$SKIP_CONFIRM" -eq 0 ]; then
	confirm "Install ${BINARY} v${INSTALL_VERSION} to ${INSTALL_DIR}?"
fi

_tmpdir="$(mktemp -d)"
trap 'rm -rf "$_tmpdir"' EXIT INT TERM

info "Downloading ${_archive_name}..."
download "$_download_url" "${_tmpdir}/${_archive_name}"

verify_checksum "${_tmpdir}/${_archive_name}" "$_checksums_url"

info "Extracting..."
case "$_ext" in
	tar.gz) tar -xzf "${_tmpdir}/${_archive_name}" -C "$_tmpdir" ;;
	zip) unzip -o -q "${_tmpdir}/${_archive_name}" -d "$_tmpdir" ;;
esac

_src_binary=""
for _candidate in "${_tmpdir}/${BINARY}" "${_tmpdir}/${SYMLINK}" "${_tmpdir}/branchless-pr_${INSTALL_VERSION}_${_os}_${_arch}/${BINARY}" "${_tmpdir}/branchless-pr_${INSTALL_VERSION}_${_os}_${_arch}/${SYMLINK}"; do
	if [ -f "$_candidate" ]; then
		_src_binary="$_candidate"
		break
	fi
done
if [ -z "$_src_binary" ]; then
	err "could not find ${BINARY} or ${SYMLINK} binary in archive"
fi

mkdir -p "$INSTALL_DIR"

mv "$_src_binary" "$_dst_bin"
chmod +x "$_dst_bin"

ln -sf "$BINARY" "$_dst_symlink"

success "Installed ${BINARY} v${INSTALL_VERSION} to ${_dst_bin}"
success "Created symlink ${_dst_symlink} -> ${BINARY}"
info ""

case ":${PATH}:" in
	*":${INSTALL_DIR}:"*) ;;
	*)
		warn "${INSTALL_DIR} is not in your PATH"
		info "Add it by running:"
		info "  echo 'export PATH=\"${INSTALL_DIR}:\$PATH\"' >> ~/.bashrc"
		info "  source ~/.bashrc"
		info ""
		;;
esac

if ! command -v gh >/dev/null 2>&1; then
	warn "GitHub CLI ('gh') is not installed. bpr requires gh to interact with GitHub PRs."
	info "  Install it from: https://cli.github.com"
	info ""
fi

info "${YELLOW}Note: the 'stack-pr' binary is deprecated.${NO_COLOR}"
info "Use '${BINARY}' instead. The '${SYMLINK}' symlink is provided for backward compatibility."
info ""
