#!/usr/bin/env bash
set -euo pipefail

test_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
feature_dir="$(cd -- "${test_dir}/../src/cue-eval-mcp" && pwd)"
temporary_dir="$(mktemp -d)"
trap 'rm -rf -- "${temporary_dir}"' EXIT

fake_binary="${temporary_dir}/cue-eval-mcp"
cat > "${fake_binary}" <<'SCRIPT'
#!/usr/bin/env sh
printf '%s\n' 'cue-eval-mcp test artifact'
SCRIPT
chmod 0755 "${fake_binary}"
digest="$(sha256sum -- "${fake_binary}" | awk '{print $1}')"

DESTDIR="${temporary_dir}/root" \
ARTIFACTPATH="${fake_binary}" \
ARTIFACTSHA256="${digest}" \
  sh "${feature_dir}/install.sh"

installed="${temporary_dir}/root/usr/local/bin/cue-eval-mcp"
test -x "${installed}"
test "$("${installed}")" = "cue-eval-mcp test artifact"

# Installing the same verified bytes is idempotent.
DESTDIR="${temporary_dir}/root" \
ARTIFACTPATH="${fake_binary}" \
ARTIFACTSHA256="${digest}" \
  sh "${feature_dir}/install.sh"

if DESTDIR="${temporary_dir}/bad-root" \
  ARTIFACTPATH="${fake_binary}" \
  ARTIFACTSHA256="$(printf '0%.0s' {1..64})" \
  sh "${feature_dir}/install.sh" >/dev/null 2>&1; then
  echo "checksum mismatch unexpectedly succeeded" >&2
  exit 1
fi

if DESTDIR="${temporary_dir}/missing-checksum" \
  ARTIFACTPATH="${fake_binary}" \
  ARTIFACTSHA256="" \
  sh "${feature_dir}/install.sh" >/dev/null 2>&1; then
  echo "missing checksum unexpectedly succeeded" >&2
  exit 1
fi

printf '%s\n' 'different artifact' >> "${fake_binary}"
different_digest="$(sha256sum -- "${fake_binary}" | awk '{print $1}')"
if DESTDIR="${temporary_dir}/root" \
  ARTIFACTPATH="${fake_binary}" \
  ARTIFACTSHA256="${different_digest}" \
  sh "${feature_dir}/install.sh" >/dev/null 2>&1; then
  echo "replacement of a different installed binary unexpectedly succeeded" >&2
  exit 1
fi

printf '%s\n' "Dev Container Feature installer checks passed"
