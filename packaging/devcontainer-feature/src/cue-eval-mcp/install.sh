#!/bin/sh
set -eu

# Clear CDPATH to avoid alternate directory lookup and extra output from cd.
feature_dir="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
install_root="${DESTDIR:-}"
install_dir="${install_root}/usr/local/bin"
install_target="${install_dir}/cue-eval-mcp"

artifact_path="${ARTIFACTPATH:-}"
expected_sha256="${ARTIFACTSHA256:-}"

machine="$(uname -m)"
case "${machine}" in
  x86_64 | amd64)
    architecture="amd64"
    ;;
  aarch64 | arm64)
    architecture="arm64"
    ;;
  *)
    echo "cue-eval-mcp Feature: unsupported architecture: ${machine}" >&2
    exit 1
    ;;
esac

if [ -n "${artifact_path}" ]; then
  if [ -z "${expected_sha256}" ]; then
    echo "cue-eval-mcp Feature: artifactSha256 is required with artifactPath" >&2
    exit 1
  fi
  source_binary="${artifact_path}"
else
  source_binary="${feature_dir}/artifacts/cue-eval-mcp-linux-${architecture}"
  checksum_file="${source_binary}.sha256"
  if [ ! -f "${checksum_file}" ]; then
    echo "cue-eval-mcp Feature: no staged ${architecture} artifact checksum" >&2
    echo "stage it with packaging/devcontainer-feature/scripts/stage-local-artifact.sh ${architecture}" >&2
    exit 1
  fi
  read -r expected_sha256 _ < "${checksum_file}"
fi

if [ ! -f "${source_binary}" ]; then
  echo "cue-eval-mcp Feature: artifact not found: ${source_binary}" >&2
  exit 1
fi

if [ "${#expected_sha256}" -ne 64 ]; then
  echo "cue-eval-mcp Feature: artifactSha256 must be 64 lowercase hexadecimal characters" >&2
  exit 1
fi
case "${expected_sha256}" in
  *[!0-9a-f]*)
  echo "cue-eval-mcp Feature: artifactSha256 must be 64 lowercase hexadecimal characters" >&2
  exit 1
    ;;
esac

actual_sha256="$(sha256sum -- "${source_binary}" | awk '{print $1}')"
if [ "${actual_sha256}" != "${expected_sha256}" ]; then
  echo "cue-eval-mcp Feature: artifact checksum mismatch" >&2
  exit 1
fi

if [ -e "${install_target}" ] || [ -L "${install_target}" ]; then
  if [ ! -f "${install_target}" ] || [ -L "${install_target}" ]; then
    echo "cue-eval-mcp Feature: refusing to replace non-regular ${install_target}" >&2
    exit 1
  fi
  installed_sha256="$(sha256sum -- "${install_target}" | awk '{print $1}')"
  if [ "${installed_sha256}" = "${actual_sha256}" ]; then
    exit 0
  fi
  echo "cue-eval-mcp Feature: refusing to replace a different ${install_target}" >&2
  exit 1
fi

install -d -m 0755 -- "${install_dir}"
install -m 0755 -- "${source_binary}" "${install_target}"
