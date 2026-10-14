#!/bin/sh
set -eu

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd -P)
REPO_ROOT=$(CDPATH= cd -- "$SCRIPT_DIR/../.." && pwd -P)

if env -u TF_ACC -u LITELLM_ACCEPTANCE_CONFIRM python3 "$SCRIPT_DIR/harness.py" preflight local >/dev/null 2>&1; then
  echo 'local preflight unexpectedly succeeded' >&2
  exit 1
fi
if env -u TF_ACC -u LITELLM_REMOTE_ACCEPTANCE_CONFIRM -u LITELLM_TEST_NAMESPACE python3 "$SCRIPT_DIR/harness.py" preflight remote >/dev/null 2>&1; then
  echo 'remote preflight unexpectedly succeeded' >&2
  exit 1
fi

TF_ACC=1 LITELLM_ACCEPTANCE_CONFIRM=local-v1.104.0 \
  python3 "$SCRIPT_DIR/harness.py" preflight local >/dev/null
sh "$SCRIPT_DIR/tests/private_plan_trigger_test.sh"
TF_ACC=1 LITELLM_REMOTE_ACCEPTANCE_CONFIRM=dev-disposable-objects-only \
LITELLM_TEST_NAMESPACE=issue210-abcdef12 \
  python3 "$SCRIPT_DIR/harness.py" preflight remote >/dev/null

# Importer detaches only the imported target; producer retains and destroys all
# owned objects. Failure injection must use the controlled endpoint proxy.
grep -q 'run_cli state rm "$address"' "$SCRIPT_DIR/run.sh"
grep -q 'producer-owned import cleanup' "$SCRIPT_DIR/run.sh"
grep -q 'fault_proxy.py' "$SCRIPT_DIR/run.sh"
grep -q -- '--require-reviewed-private-migration' "$SCRIPT_DIR/run.sh"
grep -q 'upgrade-private-plan-trigger-migration' "$SCRIPT_DIR/run.sh"
grep -q 'upgrade-reviewed-private-migration' "$SCRIPT_DIR/harness.py"
grep -q '"litellm_agent": \["id"\]' "$SCRIPT_DIR/matrix.json"
grep -q 'terraform validate' "$REPO_ROOT/internal_testing/smoke.sh"
grep -q 'unset SMOKE_SUPPLEMENTAL_ONLY SMOKE_FALLBACK_DELETE_UNSUPPORTED SMOKE_FALLBACK_DELETE_ADDRESS SMOKE_FALLBACK_IMPORT' "$REPO_ROOT/internal_testing/acceptance.sh"
grep -q 'terraform state rm "$fallback_delete_address"' "$REPO_ROOT/internal_testing/smoke.sh"
grep -q 'validate_fixture_name "$file"' "$REPO_ROOT/internal_testing/smoke.sh"
grep -q 'scenario command evidence is not bound to one exact' "$SCRIPT_DIR/harness.py"
# Organizations require a LiteLLM Enterprise license from 1.102.0: without the
# licensed confirmation, acceptance must prove the exact license diagnostic and
# the runner must record explicit skips rather than passes.
grep -q 'run_organization_license_gate_probe' "$REPO_ROOT/internal_testing/acceptance.sh"
grep -q 'title != "Error: LiteLLM Enterprise License Required"' "$REPO_ROOT/internal_testing/acceptance.sh"
grep -q 'resources organization_minimal.tf$' "$REPO_ROOT/internal_testing/acceptance.sh"
grep -q "record 'lifecycle:litellm_organization_member' lifecycle skipped enterprise-license-required" "$SCRIPT_DIR/run.sh"
grep -q '"type":"litellm_organization",.*"lane":"enterprise"' "$SCRIPT_DIR/matrix.json"
grep -q '"type":"litellm_organization_member",.*"lane":"enterprise"' "$SCRIPT_DIR/matrix.json"
grep -q 'CLI=$selected_cli' "$SCRIPT_DIR/run.sh"
grep -q 'rm -f -- "$port_file" "$stats_file"' "$SCRIPT_DIR/run.sh"
grep -q 'unforwarded team identity was not authoritatively absent' "$SCRIPT_DIR/run.sh"
grep -q 'run_cli state rm "$IMPORT_ADDRESS"' "$SCRIPT_DIR/run.sh"
grep -q "litellm_fallback.minimal|'litellm_fallback.fallback_imported\[0\]'" "$REPO_ROOT/internal_testing/smoke.sh"
if grep -q 'litellm_api_base=http://127.0.0.1:1' "$SCRIPT_DIR/run.sh"; then
  echo 'failure recovery uses a dead base URL instead of the controlled proxy' >&2
  exit 1
fi
if grep -E '(cp|source|\.) .*dev\.env' "$SCRIPT_DIR/run.sh" >/dev/null 2>&1; then
  echo 'runner attempts to read or copy dev.env' >&2
  exit 1
fi
if grep -q 'LITELLM_REMOTE_ACCEPTANCE_CONFIRM' "$REPO_ROOT/.github/workflows/upgrade-matrix.yml"; then
  echo 'workflow unexpectedly enables a remote mutation lane' >&2
  exit 1
fi
if grep -q 'report-records' "$SCRIPT_DIR/run.sh"; then
  echo 'runner still accepts forgeable manual report records' >&2
  exit 1
fi
if find "$REPO_ROOT/internal_testing" -name '*.tf' -type f \
  -exec grep -H -E 'error_message[[:space:]]*=[[:space:]]*"[a-z]' {} + | grep -q .; then
  echo 'Terraform fixture validation messages must start with uppercase prose for Terraform 1.1' >&2
  exit 1
fi

# A logical PWD beneath an attacker-controlled symlinked ancestor must resolve
# to physical repository/workspace roots before any Terraform CLI argument is
# assembled. Report publication remains strict no-follow and uses a physical
# private destination rather than weakening its ancestor policy.
symlink_test=$(mktemp -d "${TMPDIR:-/tmp}/issue210-symlink-cwd.XXXXXX")
physical_test=$(python3 -c 'from pathlib import Path; import sys; print(Path(sys.argv[1]).resolve())' "$symlink_test")
ln -s "$REPO_ROOT" "$physical_test/repository-alias"
mkdir -m 700 "$physical_test/report"
printf '%s\n' "$REPO_ROOT" >"$physical_test/expected-root"
# Formatting is checked separately by assembly. This fixture checks only the
# working directory and expected commands for the symlink-path regression.
cat >"$physical_test/cli" <<'SH'
#!/bin/sh
set -eu
[ "$(pwd -P)" = "$(cat "$(dirname "$0")/expected-root")" ]
[ "$#" -eq 4 ]
[ "$1" = fmt ]
[ "$2" = -check ]
[ "$3" = -recursive ]
case "$4" in examples|internal_testing) ;; *) exit 1 ;; esac
SH
chmod 700 "$physical_test/cli"
(
  cd "$physical_test/repository-alias"
  MATRIX_CLI="$physical_test/cli" \
  MATRIX_PROVIDER_BINARY="$physical_test/missing-provider" \
  MATRIX_REPORT="$physical_test/report/assembly.json" \
    sh internal_testing/upgrade_matrix/run.sh assembly >/dev/null
)
[ -s "$physical_test/report/assembly.json" ]
rm -rf "$physical_test"
