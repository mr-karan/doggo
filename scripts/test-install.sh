#!/usr/bin/env sh

set -eu

SCRIPT_DIR="$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)"
DOGGO_TEST_TMP_DIR="$(mktemp -d "${TMPDIR:-/tmp}/doggo-install-test.XXXXXX")"

cleanup() {
  rm -rf "${DOGGO_TEST_TMP_DIR}"
}
trap cleanup 0 1 2 3 15

INSTALL_SCRIPT="${SCRIPT_DIR}/../install.sh"
TESTABLE_INSTALL_SCRIPT="${DOGGO_TEST_TMP_DIR}/install-functions.sh"

if [ "$(tail -n 1 "${INSTALL_SCRIPT}")" != "main" ]; then
  printf 'not ok - install.sh must end with the main invocation\n' >&2
  exit 1
fi
sed '$d' "${INSTALL_SCRIPT}" >"${TESTABLE_INSTALL_SCRIPT}"
# shellcheck source=../install.sh
. "${TESTABLE_INSTALL_SCRIPT}"

fail() {
  printf 'not ok - %s\n' "$1" >&2
  exit 1
}

assert_equal() {
  actual="$1"
  expected="$2"
  description="$3"

  if [ "${actual}" != "${expected}" ]; then
    fail "${description}: expected '${expected}', got '${actual}'"
  fi
  printf 'ok - %s\n' "${description}"
}

assert_supported() {
  target="$1"
  if ! is_supported_target "${target}"; then
    fail "${target} should be supported"
  fi
  printf 'ok - %s is supported\n' "${target}"
}

assert_equal "$(detect_arch armv6l)" "armv6" "armv6l selects the ARMv6 artifact"
assert_equal "$(detect_arch armv7l)" "armv7" "armv7l selects the ARMv7 artifact"
assert_equal "$(detect_arch armv8l)" "armv7" "32-bit armv8l selects the ARMv7 artifact"

if (detect_arch armv5l) >/dev/null 2>&1; then
  fail "armv5l should remain unsupported"
fi
printf 'ok - armv5l remains unsupported\n'

assert_supported "linux_armv6"
assert_supported "linux_armv7"

if is_supported_target linux_arm; then
  fail "the ambiguous linux_arm installer target should remain unsupported"
fi
printf 'ok - ambiguous linux_arm installer target remains unsupported\n'

assert_equal \
  "$(artifact_filename linux armv6)" \
  "doggo-linux-armv6.tar.gz" \
  "ARMv6 artifact name"
assert_equal \
  "$(artifact_filename linux armv7)" \
  "doggo-linux-armv7.tar.gz" \
  "ARMv7 artifact name"
assert_equal \
  "$(release_url v1.2.3 "$(artifact_filename linux armv6)")" \
  "https://github.com/mr-karan/doggo/releases/download/v1.2.3/doggo-linux-armv6.tar.gz" \
  "ARMv6 release URL"
assert_equal \
  "$(release_url v1.2.3 "$(artifact_filename linux armv7)")" \
  "https://github.com/mr-karan/doggo/releases/download/v1.2.3/doggo-linux-armv7.tar.gz" \
  "ARMv7 release URL"
assert_equal \
  "$(compatibility_filename linux armv7)" \
  "doggo-linux-arm.tar.gz" \
  "ARMv7 compatibility artifact"

if compatibility_filename linux armv6 >/dev/null; then
  fail "ARMv6 must not fall back to the GOARM=7 compatibility artifact"
fi
printf 'ok - ARMv6 rejects the GOARM=7 compatibility artifact\n'

download_file() {
  return 1
}

if armv6_error="$(download_and_install v1.2.3 linux armv6 2>&1)"; then
  fail "a missing ARMv6 release asset should fail"
fi

case "${armv6_error}" in
  *"No ARMv6 release asset found for doggo v1.2.3."*) ;;
  *) fail "missing ARMv6 asset should produce a clear error" ;;
esac
case "${armv6_error}" in
  *"doggo_1.2.3_Linux_armv6.tar.gz"*)
    fail "missing ARMv6 asset must not try a fabricated legacy URL"
    ;;
esac
printf 'ok - missing ARMv6 asset fails without a fabricated legacy URL\n'
