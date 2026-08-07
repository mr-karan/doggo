#!/usr/bin/env sh

set -eu

DOGGO_RELEASE_TEST_TMP_DIR="$(mktemp -d "${TMPDIR:-/tmp}/doggo-release-test.XXXXXX")"

cleanup() {
  rm -rf "${DOGGO_RELEASE_TEST_TMP_DIR}"
}
trap cleanup 0 1 2 3 15

fail() {
  printf 'not ok - %s\n' "$1" >&2
  exit 1
}

assert_archive() {
  archive_path="dist/$1"
  if [ ! -f "${archive_path}" ]; then
    fail "missing ${archive_path}"
  fi
  printf 'ok - found %s\n' "${archive_path}"
}

assert_goarm() {
  binary="$1"
  expected="$2"

  if [ ! -x "${binary}" ]; then
    fail "missing executable ${binary}"
  fi
  if ! go version -m "${binary}" | grep -Eq "[[:space:]]GOARM=${expected}$"; then
    fail "${binary} does not report GOARM=${expected}"
  fi
  printf 'ok - %s reports GOARM=%s\n' "${binary}" "${expected}"
}

assert_same_archive_binary() {
  explicit="dist/$1"
  compatibility="dist/$2"
  explicit_binary="${DOGGO_RELEASE_TEST_TMP_DIR}/explicit-doggo"
  compatibility_binary="${DOGGO_RELEASE_TEST_TMP_DIR}/compatibility-doggo"

  tar -xOf "${explicit}" doggo >"${explicit_binary}"
  tar -xOf "${compatibility}" doggo >"${compatibility_binary}"
  if ! cmp -s "${explicit_binary}" "${compatibility_binary}"; then
    fail "${compatibility} does not contain the ${explicit} binary"
  fi
  printf 'ok - %s matches %s\n' "${compatibility}" "${explicit}"
}

set -- dist/*_checksums.txt
if [ "$#" -ne 1 ] || [ ! -f "$1" ]; then
  fail "expected exactly one generated checksums file"
fi
CHECKSUMS_FILE="$1"

for archive_name in \
  doggo-linux-armv6.tar.gz \
  doggo-linux-armv7.tar.gz \
  doggo-linux-arm.tar.gz \
  doggo-freebsd-armv7.tar.gz \
  doggo-freebsd-arm.tar.gz \
  doggo-openbsd-armv7.tar.gz \
  doggo-openbsd-arm.tar.gz \
  doggo-netbsd-armv7.tar.gz \
  doggo-netbsd-arm.tar.gz
do
  assert_archive "${archive_name}"
  if ! awk '{ print $2 }' "${CHECKSUMS_FILE}" | grep -Fxq "${archive_name}"; then
    fail "${archive_name} is not listed in ${CHECKSUMS_FILE}"
  fi
  printf 'ok - %s lists %s\n' "${CHECKSUMS_FILE}" "${archive_name}"
done

archive_count="$(find dist -maxdepth 1 -type f \( -name '*.tar.gz' -o -name '*.zip' \) | wc -l | tr -d ' ')"
if [ "${archive_count}" -ne 9 ]; then
  fail "expected exactly 9 ARM archives, found ${archive_count}"
fi
printf 'ok - snapshot contains exactly 9 ARM archives\n'

binary_count="$(find dist -mindepth 2 -maxdepth 2 -type f -name doggo | wc -l | tr -d ' ')"
if [ "${binary_count}" -ne 5 ]; then
  fail "expected exactly 5 ARM binaries, found ${binary_count}"
fi
printf 'ok - snapshot contains exactly 5 ARM binaries\n'

for os in freebsd openbsd netbsd; do
  if [ -e "dist/doggo-${os}-armv6.tar.gz" ]; then
    fail "unexpected ARMv6 archive for ${os}"
  fi
done
printf 'ok - ARMv6 remains Linux-only\n'

if ! command -v go >/dev/null 2>&1; then
  fail "go is required to validate GOARM build metadata"
fi

assert_goarm "dist/cli-armv6_linux_arm_6/doggo" 6
assert_goarm "dist/cli-armv7_linux_arm_7/doggo" 7
assert_goarm "dist/cli-armv7_freebsd_arm_7/doggo" 7
assert_goarm "dist/cli-armv7_openbsd_arm_7/doggo" 7
assert_goarm "dist/cli-armv7_netbsd_arm_7/doggo" 7

for os in linux freebsd openbsd netbsd; do
  assert_same_archive_binary \
    "doggo-${os}-armv7.tar.gz" \
    "doggo-${os}-arm.tar.gz"
done
