#!/usr/bin/env sh
# install.sh — Install kanbanzai from the latest GitHub Release
# Usage: curl -fsSL https://raw.githubusercontent.com/samphillips/kanbanzai/main/install.sh | sh
# Or:    ./install.sh [--version v1.0.0-alpha.1]
set -e

REPO="samphillips/kanbanzai"
BINARY_NAME="kanbanzai"
GITHUB_API="https://api.github.com/repos/${REPO}/releases"

# ── Colour helpers (no-op if not a terminal) ──────────────────────────────────
if [ -t 1 ]; then
  RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; BOLD='\033[1m'; RESET='\033[0m'
else
  RED=''; GREEN=''; YELLOW=''; BOLD=''; RESET=''
fi

info()  { printf "${BOLD}%s${RESET}\n" "$*"; }
ok()    { printf "${GREEN}✔ %s${RESET}\n" "$*"; }
warn()  { printf "${YELLOW}! %s${RESET}\n" "$*" >&2; }
die()   { printf "${RED}✖ %s${RESET}\n" "$*" >&2; exit 1; }

# ── Argument parsing ──────────────────────────────────────────────────────────
REQUESTED_VERSION=""
while [ $# -gt 0 ]; do
  case "$1" in
    --version|-v)
      REQUESTED_VERSION="$2"
      shift 2
      ;;
    --version=*)
      REQUESTED_VERSION="${1#*=}"
      shift
      ;;
    *)
      die "Unknown argument: $1"
      ;;
  esac
done

# ── Prerequisite checks ───────────────────────────────────────────────────────
need_cmd() { command -v "$1" >/dev/null 2>&1 || die "Required command not found: $1"; }
need_cmd uname
need_cmd curl
need_cmd tar

# ── OS detection ──────────────────────────────────────────────────────────────
detect_os() {
  _uname_s="$(uname -s)"
  case "$_uname_s" in
    Darwin)  echo "darwin"  ;;
    Linux)   echo "linux"   ;;
    MINGW*|MSYS*|CYGWIN*) echo "windows" ;;
    *) die "Unsupported operating system: $_uname_s" ;;
  esac
}

# ── Architecture detection ────────────────────────────────────────────────────
detect_arch() {
  _uname_m="$(uname -m)"
  case "$_uname_m" in
    arm64|aarch64) echo "arm64" ;;
    x86_64|amd64)  echo "amd64" ;;
    *) die "Unsupported architecture: $_uname_m" ;;
  esac
}

OS="$(detect_os)"
ARCH="$(detect_arch)"
info "Detected platform: ${OS}/${ARCH}"

# ── Resolve target version ────────────────────────────────────────────────────
if [ -n "$REQUESTED_VERSION" ]; then
  VERSION="$REQUESTED_VERSION"
  # Ensure version has a leading 'v' for the tag lookup
  case "$VERSION" in
    v*) ;;
    *)  VERSION="v${VERSION}" ;;
  esac
  info "Requested version: ${VERSION}"
  RELEASE_URL="${GITHUB_API}/tags/${VERSION}"
else
  info "Fetching latest release from GitHub..."
  RELEASE_URL="${GITHUB_API}/latest"
fi

# Fetch the release JSON and extract the tag_name
RELEASE_JSON="$(curl -fsSL "$RELEASE_URL")" \
  || die "Failed to fetch release information from ${RELEASE_URL}"

# Extract tag_name using only POSIX tools (no jq dependency)
TAG="$(printf '%s' "$RELEASE_JSON" | grep '"tag_name"' | head -1 | sed 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/')"
[ -n "$TAG" ] || die "Could not determine release tag. Is the release published on GitHub?"

# Version string in archive filenames has no leading 'v' (GoReleaser default)
VERSION_BARE="${TAG#v}"
info "Installing ${BINARY_NAME} ${TAG}"

# ── Construct asset URLs ──────────────────────────────────────────────────────
case "$OS" in
  windows) EXT="zip" ;;
  *)       EXT="tar.gz" ;;
esac

ARCHIVE_NAME="${BINARY_NAME}_${VERSION_BARE}_${OS}_${ARCH}.${EXT}"
BASE_URL="https://github.com/${REPO}/releases/download/${TAG}"
ARCHIVE_URL="${BASE_URL}/${ARCHIVE_NAME}"
CHECKSUM_URL="${BASE_URL}/checksums.txt"

# ── Temporary working directory ───────────────────────────────────────────────
TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT INT TERM

ARCHIVE_PATH="${TMP_DIR}/${ARCHIVE_NAME}"
CHECKSUM_PATH="${TMP_DIR}/checksums.txt"

# ── Download ──────────────────────────────────────────────────────────────────
info "Downloading ${ARCHIVE_NAME}..."
curl -fsSL --progress-bar -o "$ARCHIVE_PATH" "$ARCHIVE_URL" \
  || die "Download failed: ${ARCHIVE_URL}"

info "Downloading checksums.txt..."
curl -fsSL -o "$CHECKSUM_PATH" "$CHECKSUM_URL" \
  || die "Download failed: ${CHECKSUM_URL}"

# ── Checksum validation ───────────────────────────────────────────────────────
info "Validating checksum..."
cd "$TMP_DIR"

if command -v sha256sum >/dev/null 2>&1; then
  # Linux / most systems
  grep "${ARCHIVE_NAME}" checksums.txt | sha256sum -c - \
    || die "Checksum validation FAILED for ${ARCHIVE_NAME}. Aborting installation."
elif command -v shasum >/dev/null 2>&1; then
  # macOS
  grep "${ARCHIVE_NAME}" checksums.txt | shasum -a 256 -c - \
    || die "Checksum validation FAILED for ${ARCHIVE_NAME}. Aborting installation."
else
  die "No checksum tool available (sha256sum or shasum). Cannot verify download integrity."
fi

ok "Checksum verified"
cd - >/dev/null

# ── Extract binary ────────────────────────────────────────────────────────────
info "Extracting archive..."
case "$EXT" in
  tar.gz)
    tar -xzf "$ARCHIVE_PATH" -C "$TMP_DIR" \
      || die "Failed to extract ${ARCHIVE_NAME}"
    ;;
  zip)
    need_cmd unzip
    unzip -q "$ARCHIVE_PATH" -d "$TMP_DIR" \
      || die "Failed to extract ${ARCHIVE_NAME}"
    ;;
esac

EXTRACTED_BINARY="${TMP_DIR}/${BINARY_NAME}"
[ -f "$EXTRACTED_BINARY" ] || EXTRACTED_BINARY="${TMP_DIR}/${BINARY_NAME}.exe"
[ -f "$EXTRACTED_BINARY" ] || die "Binary not found in archive after extraction"

chmod +x "$EXTRACTED_BINARY"

# ── Install binary ────────────────────────────────────────────────────────────
INSTALL_DIR=""
SYSTEM_DIR="/usr/local/bin"
USER_DIR="${HOME}/.local/bin"

if [ -w "$SYSTEM_DIR" ] || ([ -d "$SYSTEM_DIR" ] && touch "${SYSTEM_DIR}/.kbz_write_test" 2>/dev/null && rm "${SYSTEM_DIR}/.kbz_write_test"); then
  INSTALL_DIR="$SYSTEM_DIR"
else
  warn "${SYSTEM_DIR} is not writable; installing to ${USER_DIR}"
  mkdir -p "$USER_DIR"
  INSTALL_DIR="$USER_DIR"
fi

INSTALL_PATH="${INSTALL_DIR}/${BINARY_NAME}"

# If already installed, show previous version before replacing
if [ -x "$INSTALL_PATH" ]; then
  PREV_VERSION="$("$INSTALL_PATH" --version 2>/dev/null || echo 'unknown')"
  warn "Upgrading from: ${PREV_VERSION}"
fi

cp "$EXTRACTED_BINARY" "$INSTALL_PATH"
chmod +x "$INSTALL_PATH"

# ── Verify installed binary ───────────────────────────────────────────────────
INSTALLED_VERSION="$("$INSTALL_PATH" --version 2>/dev/null)" \
  || die "Installed binary failed to run. Installation may be incomplete."

ok "Installed: ${INSTALL_PATH}"
ok "Version:   ${INSTALLED_VERSION}"

# Warn if the install directory is not on PATH
case ":${PATH}:" in
  *":${INSTALL_DIR}:"*) ;;
  *)
    warn "${INSTALL_DIR} is not in your PATH."
    warn "Add the following to your shell profile:"
    warn "  export PATH=\"${INSTALL_DIR}:\$PATH\""
    ;;
esac

echo ""
info "Installation complete. Run '${BINARY_NAME} --help' to get started."
