#!/usr/bin/env bash
# Skip a CI job when an ancestor (or other known commit) already has a green
# check-run for the same package + workflow identity.
#
# Usage:
#   ci-skip-if-ancestor-passed.sh <check-run-name> <path> [<path>...]
#
# Paths are compared as git tree/blob OIDs at HEAD vs candidate commits.
# Typically: the package dir plus .github/workflows/ci.yml (so CI logic
# changes still force a re-run).
#
# Outputs (GITHUB_OUTPUT):
#   hit=true|false
#   via=<commit> (when hit)
set -euo pipefail

if [[ $# -lt 2 ]]; then
  echo "usage: $0 <check-run-name> <path> [<path>...]" >&2
  exit 2
fi

CHECK_NAME="$1"
shift
PATHS=("$@")

emit() {
  local key="$1"
  local value="$2"
  if [[ "${GITHUB_OUTPUT:-}" ]]; then
    echo "${key}=${value}" >>"$GITHUB_OUTPUT"
  else
    echo "${key}=${value}"
  fi
}

head_oid() {
  local p="$1"
  if git cat-file -e "HEAD:${p}" 2>/dev/null; then
    git rev-parse "HEAD:${p}"
  else
    echo "MISSING"
  fi
}

commit_oid() {
  local c="$1"
  local p="$2"
  if git cat-file -e "${c}:${p}" 2>/dev/null; then
    git rev-parse "${c}:${p}"
  else
    echo "MISSING"
  fi
}

identity_match() {
  local c="$1"
  local p
  for p in "${PATHS[@]}"; do
    local a b
    a="$(head_oid "$p")"
    b="$(commit_oid "$c" "$p")"
    if [[ "$a" != "$b" ]]; then
      return 1
    fi
  done
  return 0
}

has_green_check() {
  local c="$1"
  # Exact name match; conclusion must be success.
  # --paginate may print one length per page; sum them.
  local name_json
  name_json="$(printf '%s' "$CHECK_NAME" | python3 -c 'import json,sys; print(json.dumps(sys.stdin.read()))')"
  gh api "repos/${GITHUB_REPOSITORY}/commits/${c}/check-runs" --paginate \
    --jq "[.check_runs[] | select(.name == ${name_json} and .conclusion == \"success\")] | length" \
    2>/dev/null | awk '{s+=$1} END {print s+0}'
}

REPO_ROOT="$(git rev-parse --show-toplevel)"
cd "$REPO_ROOT"

if [[ -z "${GITHUB_REPOSITORY:-}" ]]; then
  echo "GITHUB_REPOSITORY is required" >&2
  emit "hit" "false"
  exit 0
fi

if ! command -v gh >/dev/null 2>&1; then
  echo "gh not available; not skipping"
  emit "hit" "false"
  exit 0
fi

echo "Check name: ${CHECK_NAME}"
echo "Identity paths:"
for p in "${PATHS[@]}"; do
  echo "  ${p} @ HEAD = $(head_oid "$p")"
done

# Prefer recent ancestors of HEAD (covers merge stacks). Also scan a few
# recent tips from integration branches when available.
mapfile -t CANDIDATES < <(
  {
    git rev-list -n 120 HEAD
    git rev-parse --verify origin/develop 2>/dev/null || true
    git rev-parse --verify origin/main 2>/dev/null || true
    git rev-list -n 30 origin/develop 2>/dev/null || true
    git rev-list -n 30 origin/main 2>/dev/null || true
  } | awk 'NF && !seen[$0]++'
)

for c in "${CANDIDATES[@]}"; do
  if ! identity_match "$c"; then
    continue
  fi
  count="$(has_green_check "$c")"
  if [[ "$count" -gt 0 ]]; then
    echo "Skip: same identity already green on ${c} (${count} check-run(s) named '${CHECK_NAME}')"
    emit "hit" "true"
    emit "via" "$c"
    exit 0
  fi
done

echo "No ancestor/tip with matching identity and green '${CHECK_NAME}'"
emit "hit" "false"
