#!/bin/sh

set -eu

OWNER="JimmyMcBride"
REPO="brain"
API_BASE="https://api.github.com"
RELEASE_BASE="https://github.com/${OWNER}/${REPO}/releases/download"
SOURCE_BASE="https://codeload.github.com/${OWNER}/${REPO}/tar.gz/refs/heads"
INSTALL_DIR="${BRAIN_INSTALL_DIR:-${HOME}/.local/bin}"
VERSION="${BRAIN_VERSION:-}"

die() {
  printf 'brain install: %s\n' "$1" >&2
  exit 1
}

need_cmd() {
  command -v "$1" >/dev/null 2>&1
}

fetch() {
  url="$1"
  out="$2"

  if need_cmd curl; then
    curl -fsSL "$url" -o "$out"
    return
  fi

  if need_cmd wget; then
    wget -qO "$out" "$url"
    return
  fi

  die "need curl or wget"
}

fetch_text() {
  url="$1"

  if need_cmd curl; then
    curl -fsSL "$url"
    return
  fi

  if need_cmd wget; then
    wget -qO- "$url"
    return
  fi

  die "need curl or wget"
}

latest_version() {
  if need_cmd curl; then
    RESPONSE="$(curl -fsSL "${API_BASE}/repos/${OWNER}/${REPO}/releases/latest" 2>/dev/null || true)"
  elif need_cmd wget; then
    RESPONSE="$(wget -qO- "${API_BASE}/repos/${OWNER}/${REPO}/releases/latest" 2>/dev/null || true)"
  else
    die "need curl or wget"
  fi

  printf '%s\n' "${RESPONSE}" |
    sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' |
    head -n 1
}

detect_os() {
  case "$(uname -s)" in
    Linux) printf 'linux\n' ;;
    Darwin) printf 'darwin\n' ;;
    *) die "unsupported OS: $(uname -s) (Windows users should run scripts/install.ps1 from PowerShell)" ;;
  esac
}

detect_arch() {
  case "$(uname -m)" in
    x86_64|amd64) printf 'amd64\n' ;;
    aarch64|arm64) printf 'arm64\n' ;;
    *) die "unsupported architecture: $(uname -m)" ;;
  esac
}

checksum_file() {
  file="$1"

  if need_cmd sha256sum; then
    sha256sum "$file" | awk '{print $1}'
    return
  fi

  if need_cmd shasum; then
    shasum -a 256 "$file" | awk '{print $1}'
    return
  fi

  if need_cmd openssl; then
    openssl dgst -sha256 "$file" | awk '{print $NF}'
    return
  fi

  die "need sha256sum, shasum, or openssl for checksum verification"
}

install_from_source_main() {
  need_cmd go || die "no published release found and Go is not installed"

  SRC_ARCHIVE="${TMPDIR}/${REPO}-main.tar.gz"
  fetch "${SOURCE_BASE}/main" "${SRC_ARCHIVE}"
  tar -xzf "${SRC_ARCHIVE}" -C "${TMPDIR}"

  SRC_DIR="$(find "${TMPDIR}" -maxdepth 1 -type d -name "${REPO}-*" | head -n 1)"
  [ -n "${SRC_DIR}" ] || die "could not unpack source archive"

  (
    cd "${SRC_DIR}"
    go build -o "${TMPDIR}/brain" .
  )
  [ -f "${TMPDIR}/brain" ] || die "source build did not produce brain binary"

  mkdir -p "${INSTALL_DIR}"
  if need_cmd install; then
    install -m 0755 "${TMPDIR}/brain" "${INSTALL_DIR}/brain"
  else
    cp "${TMPDIR}/brain" "${INSTALL_DIR}/brain"
    chmod 0755 "${INSTALL_DIR}/brain"
  fi

  printf 'Installed to %s/brain by building the current main branch source\n' "${INSTALL_DIR}"

  RESOLVED="$(command -v brain 2>/dev/null || true)"
  if [ "${RESOLVED}" != "${INSTALL_DIR}/brain" ]; then
    printf 'PATH note: ensure %s is ahead of older brain installs\n' "${INSTALL_DIR}"
  fi
}

VERSION="${VERSION:-$(latest_version)}"
if [ -z "${VERSION}" ]; then
  TMPDIR="$(mktemp -d)"
  trap 'rm -rf "${TMPDIR}"' EXIT INT TERM
  install_from_source_main
  exit 0
fi

OS="$(detect_os)"
ARCH="$(detect_arch)"
ARCHIVE="brain_${VERSION}_${OS}_${ARCH}.tar.gz"
CHECKSUMS="brain_${VERSION}_checksums.txt"
ARCHIVE_URL="${RELEASE_BASE}/${VERSION}/${ARCHIVE}"
CHECKSUMS_URL="${RELEASE_BASE}/${VERSION}/${CHECKSUMS}"

TMPDIR="$(mktemp -d)"
trap 'rm -rf "${TMPDIR}"' EXIT INT TERM

ARCHIVE_PATH="${TMPDIR}/${ARCHIVE}"
CHECKSUMS_PATH="${TMPDIR}/${CHECKSUMS}"
BIN_PATH="${TMPDIR}/brain"

printf 'Installing brain %s for %s/%s\n' "${VERSION}" "${OS}" "${ARCH}"
fetch "${ARCHIVE_URL}" "${ARCHIVE_PATH}"
fetch "${CHECKSUMS_URL}" "${CHECKSUMS_PATH}"

EXPECTED="$(awk -v name="${ARCHIVE}" '$2 == name { print $1 }' "${CHECKSUMS_PATH}")"
[ -n "${EXPECTED}" ] || die "checksum entry not found for ${ARCHIVE}"

ACTUAL="$(checksum_file "${ARCHIVE_PATH}")"
[ "${EXPECTED}" = "${ACTUAL}" ] || die "checksum mismatch for ${ARCHIVE}"

tar -xzf "${ARCHIVE_PATH}" -C "${TMPDIR}"
[ -f "${BIN_PATH}" ] || die "archive did not contain brain binary"

mkdir -p "${INSTALL_DIR}"

if need_cmd install; then
  install -m 0755 "${BIN_PATH}" "${INSTALL_DIR}/brain"
else
  cp "${BIN_PATH}" "${INSTALL_DIR}/brain"
  chmod 0755 "${INSTALL_DIR}/brain"
fi

printf 'Installed to %s/brain\n' "${INSTALL_DIR}"

RESOLVED="$(command -v brain 2>/dev/null || true)"
if [ "${RESOLVED}" != "${INSTALL_DIR}/brain" ]; then
  printf 'PATH note: ensure %s is ahead of older brain installs\n' "${INSTALL_DIR}"
fi
