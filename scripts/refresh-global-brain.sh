#!/bin/sh

set -eu

SCRIPT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
REPO_ROOT="$(CDPATH= cd -- "${SCRIPT_DIR}/.." && pwd)"
INSTALL_DIR="${BRAIN_INSTALL_DIR:-${HOME}/.local/bin}"
BIN_PATH="${INSTALL_DIR}/brain"
REPO_SKILL_PATH="${REPO_ROOT}/skills/brain"

die() {
  printf 'brain refresh: %s\n' "$1" >&2
  exit 1
}

need_cmd() {
  command -v "$1" >/dev/null 2>&1
}

global_skill_path() {
  agent="$1"
  case "$agent" in
    codex) printf '%s\n' "${HOME}/.codex/skills/brain" ;;
    claude) printf '%s\n' "${HOME}/.claude/skills/brain" ;;
    copilot) printf '%s\n' "${HOME}/.copilot/skills/brain" ;;
    openclaw) printf '%s\n' "${HOME}/.openclaw/skills/brain" ;;
    pi) printf '%s\n' "${HOME}/.pi/agent/skills/brain" ;;
    ai) printf '%s\n' "${HOME}/.ai/skills/brain" ;;
    *) return 1 ;;
  esac
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

set --
for agent in codex claude copilot openclaw pi ai; do
  path="$(global_skill_path "${agent}")"
  if [ -e "${path}" ]; then
    set -- "$@" --agent "${agent}"
  fi
done

if [ "$#" -ne 0 ]; then
  "${BIN_PATH}" skills install --scope global "$@" >/dev/null
  for agent in codex claude copilot openclaw pi ai; do
    path="$(global_skill_path "${agent}")"
    if [ -e "${path}" ]; then
      [ -d "${path}" ] || die "global ${agent} brain skill was not installed"
      diff -qr --exclude='.brain-skill-manifest.json' "${REPO_SKILL_PATH}" "${path}" >/dev/null 2>&1 || die "global ${agent} brain skill does not match repo copy"
    fi
  done
fi

printf 'Refreshed global brain\n'
printf '  binary: %s\n' "${BIN_PATH}"
printf '  commit: %s\n' "${COMMIT}"
if [ "$#" -eq 0 ]; then
  printf '  skills: none detected\n'
else
  printf '  skills: refreshed existing global installs\n'
fi
