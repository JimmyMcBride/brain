#!/bin/sh

set -eu

SCRIPT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
REPO_ROOT="$(CDPATH= cd -- "${SCRIPT_DIR}/.." && pwd)"
INSTALL_DIR="${BRAIN_INSTALL_DIR:-${HOME}/.local/bin}"
BIN_PATH="${INSTALL_DIR}/brain"
GLOBAL_SKILL_PATH="${HOME}/.codex/skills/brain"
REPO_SKILL_PATH="${REPO_ROOT}/skills/brain"

die() {
  printf 'brain refresh: %s\n' "$1" >&2
  exit 1
}

need_cmd() {
  command -v "$1" >/dev/null 2>&1
}

need_cmd git || die "need git"
need_cmd go || die "need go"
need_cmd diff || die "need diff"

[ -d "${REPO_ROOT}/.git" ] || die "repo root does not look like a git checkout: ${REPO_ROOT}"
[ -d "${REPO_SKILL_PATH}" ] || die "missing repo skill source: ${REPO_SKILL_PATH}"

COMMIT="$(git -C "${REPO_ROOT}" rev-parse HEAD)"
DATE="$(git -C "${REPO_ROOT}" show -s --format=%cI HEAD)"

mkdir -p "${INSTALL_DIR}"

go build \
  -C "${REPO_ROOT}" \
  -ldflags "-X brain/internal/buildinfo.Commit=${COMMIT} -X brain/internal/buildinfo.Date=${DATE}" \
  -o "${BIN_PATH}" \
  .

[ -x "${BIN_PATH}" ] || die "build did not produce ${BIN_PATH}"

VERSION_OUTPUT="$("${BIN_PATH}" version)"
printf '%s\n' "${VERSION_OUTPUT}" | grep -F "commit:  ${COMMIT}" >/dev/null 2>&1 || die "installed binary commit does not match ${COMMIT}"

"${BIN_PATH}" skills install --scope global --agent codex --mode copy --project "${REPO_ROOT}" >/dev/null

[ -d "${GLOBAL_SKILL_PATH}" ] || die "global Codex brain skill was not installed"
diff -qr "${REPO_SKILL_PATH}" "${GLOBAL_SKILL_PATH}" >/dev/null 2>&1 || die "global Codex brain skill does not match repo copy"

printf 'Refreshed global brain\n'
printf '  binary: %s\n' "${BIN_PATH}"
printf '  commit: %s\n' "${COMMIT}"
printf '  skill:  %s\n' "${GLOBAL_SKILL_PATH}"
