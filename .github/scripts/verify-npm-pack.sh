#!/usr/bin/env bash
# Fail if published package.json would break npm install for consumers:
# - reject file:/link:/workspace: dependency specs in dependencies/optionalDependencies
# - npm pack → install into an empty directory
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
PKG_DIR="${ROOT}/node/sscachian"
cd "${PKG_DIR}"

node --input-type=module <<'EOF'
import { readFileSync } from "node:fs";

const pkg = JSON.parse(readFileSync("package.json", "utf8"));
const blocks = ["dependencies", "optionalDependencies", "peerDependencies"];
const bad = [];
for (const block of blocks) {
  const deps = pkg[block] || {};
  for (const [name, spec] of Object.entries(deps)) {
    const s = String(spec);
    if (
      s.startsWith("file:") ||
      s.startsWith("link:") ||
      s.startsWith("workspace:") ||
      s.startsWith("./") ||
      s.startsWith("../")
    ) {
      bad.push(`${block}.${name}=${s}`);
    }
  }
}
if (bad.length) {
  console.error("Disallowed non-registry dependency specs in package.json:");
  for (const line of bad) console.error(`  - ${line}`);
  console.error("Published packages must depend on registry versions only.");
  process.exit(1);
}
console.log("package.json dependency specs: ok (no file:/link:/workspace:)");
EOF

tmpdir="$(mktemp -d)"
trap 'rm -rf "${tmpdir}"' EXIT

echo "Running npm pack in ${PKG_DIR}..."
tarball="$(npm pack --silent --pack-destination "${tmpdir}")"
tarball_path="${tmpdir}/${tarball}"
echo "Packed: ${tarball_path}"

install_dir="${tmpdir}/consumer"
mkdir -p "${install_dir}"
cd "${install_dir}"
npm init -y >/dev/null
npm install "${tarball_path}"

node --input-type=module <<'EOF'
import { readFileSync, existsSync } from "node:fs";
import { join } from "node:path";

const pkgJsonPath = join(process.cwd(), "node_modules", "@b4moss", "ss-cachian", "package.json");
if (!existsSync(pkgJsonPath)) {
  console.error("Installed package.json missing at", pkgJsonPath);
  process.exit(1);
}
const pkg = JSON.parse(readFileSync(pkgJsonPath, "utf8"));
if (pkg.name !== "@b4moss/ss-cachian") {
  console.error("Unexpected package after install:", pkg);
  process.exit(1);
}
const deps = pkg.dependencies || {};
for (const [name, spec] of Object.entries(deps)) {
  const s = String(spec);
  if (s.startsWith("file:") || s.startsWith("link:") || s.startsWith("workspace:")) {
    console.error(`Installed package still has non-registry dep: ${name}=${s}`);
    process.exit(1);
  }
}
// ESM import of the public entry (0.11.0 failed before reaching this)
await import("@b4moss/ss-cachian");
// Nested registry dep that 0.11.0 failed to fetch (ENOENT on missing vendor tarball)
const es4Pkg = join(process.cwd(), "node_modules", "@b4moss", "es4", "package.json");
if (!existsSync(es4Pkg)) {
  console.error("Expected @b4moss/es4 to be installed from the registry; missing", es4Pkg);
  process.exit(1);
}
const es4 = JSON.parse(readFileSync(es4Pkg, "utf8"));
console.log(`npm pack consumer install ok: ${pkg.name}@${pkg.version} (es4@${es4.version})`);
EOF
