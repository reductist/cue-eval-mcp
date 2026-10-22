#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
repository_root="$(cd -- "${script_dir}/../.." && pwd)"
container_runtime="${CONTAINER_RUNTIME:-docker}"
image="${CUE_EVAL_IMAGE:-cue-eval-mcp:smoke}"

cd -- "${repository_root}"
"${container_runtime}" build -f Containerfile -t "${image}" .
runtime_user="$("${container_runtime}" image inspect --format '{{.Config.User}}' "${image}")"
if [[ "${runtime_user}" != "65532:65532" ]]; then
  echo "unexpected container runtime user: ${runtime_user}" >&2
  exit 1
fi
"${container_runtime}" run --rm \
  --network none \
  --read-only \
  --tmpfs /tmp:rw,noexec,nosuid,size=16m \
  --mount "type=bind,src=${repository_root},dst=/workspace,readonly" \
  "${image}" --help >/dev/null

initialize='{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"container-smoke","version":"0.0.0"}}}'
initialized='{"jsonrpc":"2.0","method":"notifications/initialized","params":{}}'
list_tools='{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}'
responses="$({ printf '%s\n%s\n%s\n' "${initialize}" "${initialized}" "${list_tools}"; sleep 1; } | \
  "${container_runtime}" run --rm -i \
    --network none \
    --read-only \
    --tmpfs /tmp:rw,noexec,nosuid,size=16m \
    --mount "type=bind,src=${repository_root},dst=/workspace,readonly" \
    "${image}")"

printf '%s\n' "${responses}" | jq -s -e '
  any(.[]; .id == 1 and .result.protocolVersion != null) and
  any(.[]; .id == 2 and
    ([.result.tools[].name] | sort) ==
    (["discover_contracts", "evaluate", "inspect_contract"] | sort))
' >/dev/null

printf '%s\n' "container CLI and MCP protocol smoke checks passed: ${image}"
