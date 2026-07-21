#!/bin/sh
set -eu

REPO="kakashy/tenda-n300"
BINARY="tenda-n300"

# ---------------------------------------------------------------------------
# Terminal helpers (respects NO_COLOR, see https://no-color.org)
# ---------------------------------------------------------------------------
if [ -t 2 ] && [ -z "${NO_COLOR:-}" ]; then
  _info()    { printf '\033[0;34m>\033[0m %s\n' "$*" >&2; }
  _success() { printf '\033[0;32m✓\033[0m %s\n' "$*" >&2; }
  _warn()    { printf '\033[0;33m!\033[0m %s\n' "$*" >&2; }
  _error()   { printf '\033[0;31m✗\033[0m %s\n' "$*" >&2; }
else
  _info()    { printf '> %s\n' "$*" >&2; }
  _success() { printf '✓ %s\n' "$*" >&2; }
  _warn()    { printf '! %s\n' "$*" >&2; }
  _error()   { printf '✗ %s\n' "$*" >&2; }
fi

_die() { _error "$@"; exit 1; }

# ---------------------------------------------------------------------------
# Command check
# ---------------------------------------------------------------------------
_need_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    _die "required command '$1' not found"
  fi
}

# ---------------------------------------------------------------------------
# Platform detection
# ---------------------------------------------------------------------------
_detect_platform() {
  _os=$(uname -s | tr '[:upper:]' '[:lower:]')
  _arch=$(uname -m)

  case "$_os" in
    linux)  _os="linux"  ;;
    darwin) _os="darwin" ;;
    *)      _die "unsupported OS: $_os (only linux/darwin)" ;;
  esac

  case "$_arch" in
    x86_64|amd64)           _arch="amd64" ;;
    aarch64|arm64)          _arch="arm64" ;;
    *)                      _die "unsupported architecture: $_arch" ;;
  esac

  printf '%s/%s' "$_os" "$_arch"
}

# ---------------------------------------------------------------------------
# Download helpers
# ---------------------------------------------------------------------------
_download_stdout() {
  if command -v curl >/dev/null 2>&1; then
    curl -fsSL --proto =https --tlsv1.2 "$@"
  elif command -v wget >/dev/null 2>&1; then
    wget -qO- --https-only "$@"
  else
    _die "need curl or wget to download files"
  fi
}

# Resolve the latest release tag.
# Tries the curl redirect trick first (avoids the GitHub API rate limit),
# then falls back to the API with an optional GITHUB_TOKEN.
_get_latest_tag() {
  _tag=""

  # Try redirect-based lookup (no API call, no rate limit).
  if command -v curl >/dev/null 2>&1; then
    _latest_url=$(curl -fsSLI --proto =https \
      -o /dev/null -w '%{url_effective}' \
      "https://github.com/$REPO/releases/latest" 2>/dev/null) || true
    if [ -n "$_latest_url" ]; then
      _tag=${_latest_url##*/}
    fi
  fi

  if [ -n "$_tag" ]; then
    printf '%s' "$_tag"
    return 0
  fi

  # Fallback: GitHub API (rate limited to 60 req/h without token).
  if command -v curl >/dev/null 2>&1; then
    if [ -n "${GITHUB_TOKEN:-}" ]; then
      _json=$(curl -fsSL -H "Authorization: Bearer $GITHUB_TOKEN" \
        "https://api.github.com/repos/$REPO/releases/latest" 2>/dev/null) || true
    else
      _json=$(curl -fsSL \
        "https://api.github.com/repos/$REPO/releases/latest" 2>/dev/null) || true
    fi
  elif command -v wget >/dev/null 2>&1; then
    if [ -n "${GITHUB_TOKEN:-}" ]; then
      _json=$(wget -qO- --header "Authorization: Bearer $GITHUB_TOKEN" \
        "https://api.github.com/repos/$REPO/releases/latest" 2>/dev/null) || true
    else
      _json=$(wget -qO- \
        "https://api.github.com/repos/$REPO/releases/latest" 2>/dev/null) || true
    fi
  fi

  if [ -n "${_json:-}" ]; then
    _tag=$(printf '%s' "$_json" | grep '"tag_name"' | head -1 | cut -d'"' -f4)
  fi

  test -n "$_tag" || return 1
  printf '%s' "$_tag"
}

# ---------------------------------------------------------------------------
# SHA-256 helper – returns hex digest of a file
# ---------------------------------------------------------------------------
_sha256() {
  _file="$1"
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$_file" | cut -d' ' -f1
  elif command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "$_file" | cut -d' ' -f1
  elif command -v openssl >/dev/null 2>&1; then
    openssl dgst -sha256 "$_file" | awk '{print $NF}'
  else
    return 1
  fi
}

# ---------------------------------------------------------------------------
# Temp directory
# ---------------------------------------------------------------------------
_mktempdir() {
  if command -v mktemp >/dev/null 2>&1; then
    _d=$(mktemp -d) || return 1
    printf '%s' "$_d"
  else
    _dir="${TMPDIR:-/tmp}/${BINARY}.$$"
    mkdir -p "$_dir" || return 1
    printf '%s' "$_dir"
  fi
}

# ---------------------------------------------------------------------------
# Shell completion installer
# ---------------------------------------------------------------------------
_install_completion() {
  _shell="$1"
  _binary="$2"
  _prefix="$3"

  case "$_shell" in
    bash)
      for _d in "$_prefix/share/bash-completion/completions" \
                /etc/bash_completion.d; do
        if [ -d "$_d" ] && [ -w "$_d" ]; then
          "$_binary" completion bash > "$_d/$BINARY" 2>/dev/null || \
            _warn "failed to write bash completion to $_d/$BINARY"
          _success "bash completion: $_d/$BINARY"
          return
        fi
      done
      _udir="$HOME/.local/share/bash-completion/completions"
      mkdir -p "$_udir"
      "$_binary" completion bash > "$_udir/$BINARY" 2>/dev/null || \
        _warn "failed to write bash completion to $_udir/$BINARY"
      _success "bash completion: $_udir/$BINARY"
      ;;
    zsh)
      for _d in /usr/local/share/zsh/site-functions \
                /usr/share/zsh/site-functions \
                "$_prefix/share/zsh/site-functions"; do
        if [ -d "$_d" ] && [ -w "$_d" ]; then
          "$_binary" completion zsh > "$_d/_$BINARY" 2>/dev/null || \
            _warn "failed to write zsh completion to $_d/_$BINARY"
          _success "zsh completion: $_d/_$BINARY"
          return
        fi
      done
      _line="source <($BINARY completion zsh)"
      if [ -f "$HOME/.zshrc" ] && grep -qs "source <($BINARY completion zsh)" "$HOME/.zshrc" 2>/dev/null; then
        _info "zsh completion already configured in ~/.zshrc"
      else
        printf '%s\n' "$_line" >> "$HOME/.zshrc"
        _success "zsh completion: added source line to ~/.zshrc"
      fi
      ;;
  esac
}

# ===========================================================================
# MAIN
# ===========================================================================
main() {
  _need_cmd uname
  _need_cmd tar
  _need_cmd grep

  # ------ platform ------
  _platform=$(_detect_platform)
  _os=${_platform%/*}
  _arch=${_platform#*/}
  _info "Detected $_os/$_arch"

  # ------ install prefix ------
  _prefix="${PREFIX:-/usr/local}"
  if [ ! -d "$_prefix/bin" ] || [ ! -w "$_prefix/bin" ]; then
    _prefix="$HOME/.local"
    mkdir -p "$_prefix/bin"
    _info "Installing to ~/.local/bin (no write permission to /usr/local/bin)"
  fi

  # ------ latest version ------
  _info "Looking up latest release ..."
  _tag=$(_get_latest_tag) || _die "could not determine latest version"
  _version=${_tag#v}

  # ------ download ------
  _archive="${BINARY}_${_version}_${_os}_${_arch}.tar.gz"
  _base_url="https://github.com/$REPO/releases/download/$_tag"

  _tmpdir=$(_mktempdir) || _die "failed to create temporary directory"
  trap 'rm -rf "$_tmpdir"' EXIT INT TERM HUP

  _info "Downloading $BINARY $_version ($_os/$_arch) ..."
  _download_stdout "$_base_url/$_archive" > "$_tmpdir/$_archive" || \
    _die "download failed: $_archive"
  _download_stdout "$_base_url/checksums.txt" > "$_tmpdir/checksums.txt" || \
    _die "download failed: checksums.txt"

  # ------ checksum ------
  _info "Verifying checksum ..."
  _expected=$(grep -F "  $_archive" "$_tmpdir/checksums.txt" | awk '{print $1}')
  if [ -z "$_expected" ]; then
    _die "checksum entry not found for $_archive"
  fi
  _actual=$(_sha256 "$_tmpdir/$_archive") || \
    _die "no SHA-256 tool found (install sha256sum, shasum, or openssl)"

  if [ "$_expected" != "$_actual" ]; then
    _die "checksum mismatch for $_archive" \
      "(expected $_expected, got $_actual)"
  fi
  _success "Checksum verified"

  # ------ extract ------
  _info "Extracting ..."
  tar -xzf "$_tmpdir/$_archive" -C "$_tmpdir"
  _binary_path="$_tmpdir/$BINARY"
  if [ ! -f "$_binary_path" ]; then
    _die "binary not found in archive (expected $BINARY)"
  fi

  # ------ install ------
  _install_path="$_prefix/bin/$BINARY"
  cp "$_binary_path" "$_install_path" || _die "failed to copy binary to $_install_path"
  chmod +x "$_install_path" || _die "failed to set executable permission on $_install_path"
  _success "Installed $BINARY $_version to $_install_path"

  # ------ shell completions (run from installed binary to avoid noexec /tmp) ------
  if command -v bash >/dev/null 2>&1; then
    _install_completion bash "$_install_path" "$_prefix"
  fi
  if command -v zsh >/dev/null 2>&1; then
    _install_completion zsh "$_install_path" "$_prefix"
  fi

  # ------ PATH reminder ------
  case ":$PATH:" in
    *:"$_prefix/bin":*) ;;
    *)
      _warn "Add $_prefix/bin to your PATH:"
      _warn "  export PATH=\"$_prefix/bin:\$PATH\""
      ;;
  esac

  _success "Done! Run \`$BINARY --help\` to get started."
}

main "$@"
