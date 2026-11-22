#!/usr/bin/env bash
# Fails when Capture's code reaches a known vulnerability that has a fixed version.
#
# Reachable vulnerabilities without a fix (for example in github.com/docker/docker,
# whose fixes ship only in the newer github.com/moby/moby modules) are reported as
# warnings, so they stay visible without blocking every pull request.
#
# Usage: govulncheck-gate.sh <govulncheck -format json output>
set -euo pipefail

report="${1:?usage: $0 <govulncheck.json>}"

# A finding is reachable when its first trace frame names a function.
# govulncheck prints a stream of JSON objects, so slurp them into one array.
findings=$(jq -s '
  [ .[] | .finding? // empty | select(.trace[0].function != null) ]
  | group_by(.osv)
  | map({
      id: .[0].osv,
      module: .[0].trace[0].module,
      version: (.[0].trace[0].version // ""),
      fixed: (.[0].fixed_version // "")
    })
' "$report")

fixable=$(jq '[.[] | select(.fixed != "")]' <<<"$findings")
unfixable=$(jq '[.[] | select(.fixed == "")]' <<<"$findings")

jq -r '.[] | "::warning title=\(.id) has no fix::\(.module)@\(.version) is reachable from Capture; see https://pkg.go.dev/vuln/\(.id)"' <<<"$unfixable"

count=$(jq 'length' <<<"$fixable")
if [ "$count" -gt 0 ]; then
  jq -r '.[] | "::error title=\(.id)::\(.module)@\(.version) is reachable from Capture; fixed in \(.fixed). See https://pkg.go.dev/vuln/\(.id)"' <<<"$fixable"
  echo "${count} reachable vulnerabilities have a fix available. Update the module, or the Go version in go.mod for the standard library."
  exit 1
fi

echo "No reachable vulnerabilities with an available fix ($(jq 'length' <<<"$unfixable") without a fix, reported as warnings)."
