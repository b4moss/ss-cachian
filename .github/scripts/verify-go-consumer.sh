#!/usr/bin/env bash
# Verify the Go module path matches the monorepo subdirectory layout and that a
# downstream module can import + build against this checkout (replace-based
# consumer smoke). Full `go get` from the public proxy requires the release tag
# `go/sscachian/vX.Y.Z` on main after merge.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
MOD_DIR="${ROOT}/go/sscachian"
EXPECTED_PATH="github.com/b4moss/ss-cachian/go/sscachian"
EXPECTED_TAG_PREFIX="go/sscachian/"

cd "${MOD_DIR}"
MOD_PATH="$(GOTOOLCHAIN=auto go list -m)"
if [[ "${MOD_PATH}" != "${EXPECTED_PATH}" ]]; then
  echo "module path mismatch:"
  echo "  got:  ${MOD_PATH}"
  echo "  want: ${EXPECTED_PATH}"
  echo "Subdirectory modules must declare a path that includes go/sscachian"
  echo "and be released with tag ${EXPECTED_TAG_PREFIX}vX.Y.Z (not a root v* tag alone)."
  exit 1
fi

# Nested-module zip exclusion: a root-tagged module path without the subdirectory
# would omit this tree from the proxy zip. Guard the layout contract.
if [[ ! -f "${MOD_DIR}/go.mod" ]]; then
  echo "missing ${MOD_DIR}/go.mod"
  exit 1
fi
if [[ -f "${ROOT}/go.mod" ]]; then
  echo "unexpected root go.mod; this repo ships the Go module only under go/sscachian/"
  exit 1
fi

echo "module path ok: ${MOD_PATH}"
echo "release tag form: ${EXPECTED_TAG_PREFIX}vX.Y.Z (plus root vX.Y.Z for npm CD)"

tmpdir="$(mktemp -d)"
trap 'rm -rf "${tmpdir}"' EXIT
cd "${tmpdir}"

GOTOOLCHAIN=auto go mod init example.com/sscachian-consumer

cat > main.go <<EOF
package main

import (
	"fmt"

	sscachian "${MOD_PATH}"
	_ "${MOD_PATH}/driver/memory"
)

func main() {
	_ = sscachian.DefaultKeyBuilder
	fmt.Println("ss-cachian go consumer ok")
}
EOF

GOTOOLCHAIN=auto go mod edit -require="${MOD_PATH}@v0.0.0"
GOTOOLCHAIN=auto go mod edit -replace="${MOD_PATH}=${MOD_DIR}"
GOTOOLCHAIN=auto go mod tidy
GOTOOLCHAIN=auto go build -o consumer .
./consumer

echo "go consumer import/build ok (replace → ${MOD_DIR})"
