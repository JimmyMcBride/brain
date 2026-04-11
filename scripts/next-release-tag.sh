#!/bin/sh

set -eu

python3 - <<'PY'
import re
import subprocess
import sys

TAG_RE = re.compile(r"^v(\d+)\.(\d+)\.(\d+)$")


def git_lines(*args):
    result = subprocess.run(
        ["git", *args],
        check=True,
        capture_output=True,
        text=True,
    )
    return [line.strip() for line in result.stdout.splitlines() if line.strip()]


def parse_semver(tag):
    match = TAG_RE.match(tag)
    if not match:
        return None
    return tuple(int(part) for part in match.groups())


def max_semver(tags):
    parsed = [(parse_semver(tag), tag) for tag in tags]
    parsed = [item for item in parsed if item[0] is not None]
    if not parsed:
        return None
    parsed.sort(key=lambda item: item[0])
    return parsed[-1]


try:
    head_tags = git_lines("tag", "--points-at", "HEAD", "--list", "v*")
    all_tags = git_lines("tag", "--list", "v*")
except subprocess.CalledProcessError as exc:
    sys.stderr.write(exc.stderr)
    sys.exit(exc.returncode)

head_best = max_semver(head_tags)
if head_best is not None:
    print(f"tag={head_best[1]}")
    print("head_already_tagged=true")
    raise SystemExit(0)

latest = max_semver(all_tags)
if latest is None:
    next_tag = "v0.1.0"
else:
    major, minor, patch = latest[0]
    next_tag = f"v{major}.{minor}.{patch + 1}"

print(f"tag={next_tag}")
print("head_already_tagged=false")
PY
