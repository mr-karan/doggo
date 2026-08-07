#!/usr/bin/env sh

set -eu

BOLD="$(tput bold 2>/dev/null || printf '')"
GREY="$(tput setaf 0 2>/dev/null || printf '')"
GREEN="$(tput setaf 2 2>/dev/null || printf '')"
YELLOW="$(tput setaf 3 2>/dev/null || printf '')"
BLUE="$(tput setaf 4 2>/dev/null || printf '')"
RED="$(tput setaf 1 2>/dev/null || printf '')"
NO_COLOR="$(tput sgr0 2>/dev/null || printf '')"

info() {
  printf '%s\n' "${BOLD}${GREY}>${NO_COLOR} $*"
}

warn() {
  printf '%s\n' "${YELLOW}! $*${NO_COLOR}"
}

error() {
  printf '%s\n' "${RED}x $*${NO_COLOR}" >&2
}

completed() {
  printf '%s\n' "${GREEN}✓${NO_COLOR} $*"
}

has() {
  command -v "$1" 1>/dev/null 2>&1
}

SUPPORTED_TARGETS="linux_x86_64 linux_aarch64 linux_armv6 linux_armv7 windows_x86_64 darwin_x86_64 darwin_aarch64"

get_latest_release() {
  curl --silent "https://api.github.com/repos/mr-karan/doggo/releases/latest" |
    grep '"tag_name":' |
    sed -E 's/.*"([^"]+)".*/\1/'
}

detect_platform() {
  platform="${1:-$(uname -s)}"
  case "${platform}" in
    Linux*) platform="linux" ;;
    Darwin*) platform="darwin" ;;
    MINGW*|MSYS*|CYGWIN*) platform="windows" ;;
    *)
      error "Unsupported platform: ${platform}"
      exit 1
      ;;
  esac
  printf '%s' "${platform}"
}

detect_arch() {
  arch="${1:-$(uname -m)}"
  case "${arch}" in
    x86_64) arch="x86_64" ;;
    aarch64|arm64) arch="aarch64" ;;
    armv6l) arch="armv6" ;;
    armv7l|armv8l) arch="armv7" ;;
    *)
      error "Unsupported architecture: ${arch}"
      exit 1
      ;;
  esac
  printf '%s' "${arch}"
}

is_supported_target() {
  case " ${SUPPORTED_TARGETS} " in
    *" $1 "*) return 0 ;;
    *) return 1 ;;
  esac
}

artifact_filename() {
  platform="$1"
  arch="$2"

  if [ "${platform}" = "windows" ]; then
    printf 'doggo-%s-%s.zip' "${platform}" "${arch}"
  else
    printf 'doggo-%s-%s.tar.gz' "${platform}" "${arch}"
  fi
}

release_url() {
  version="$1"
  filename="$2"
  printf 'https://github.com/mr-karan/doggo/releases/download/%s/%s' "${version}" "${filename}"
}

compatibility_filename() {
  platform="$1"
  arch="$2"

  # Releases before explicit ARM variants used this name for a GOARM=7 build.
  if [ "${platform}_${arch}" = "linux_armv7" ]; then
    printf 'doggo-linux-arm.tar.gz'
    return 0
  fi
  return 1
}

legacy_platform() {
  case "$1" in
    linux) printf 'Linux' ;;
    darwin) printf 'Darwin' ;;
    windows) printf 'Windows' ;;
  esac
}

legacy_arch() {
  case "$1" in
    aarch64) printf 'arm64' ;;
    armv7) printf 'arm' ;;
    *) printf '%s' "$1" ;;
  esac
}

download_file() {
  url="$1"
  filename="$2"

  if has curl; then
    curl -fsSL "${url}" -o "${filename}"
  elif has wget; then
    wget -q "${url}" -O "${filename}"
  else
    error "Neither curl nor wget found. Please install one of them and try again."
    exit 1
  fi
}

download_and_install() {
  version="$1"
  platform="$2"
  arch="$3"

  filename="$(artifact_filename "${platform}" "${arch}")"
  url="$(release_url "${version}" "${filename}")"

  info "Downloading doggo ${version} for ${platform}_${arch}..."
  info "Download URL: ${url}"

  if ! download_file "${url}" "${filename}"; then
    warn "Could not download ${filename}."
    rm -f "${filename}"

    if [ "${platform}_${arch}" = "linux_armv6" ]; then
      error "No ARMv6 release asset found for doggo ${version}."
      exit 1
    fi

    downloaded=false
    if filename="$(compatibility_filename "${platform}" "${arch}")"; then
      url="$(release_url "${version}" "${filename}")"
      info "Compatibility download URL: ${url}"
      if download_file "${url}" "${filename}"; then
        downloaded=true
      else
        rm -f "${filename}"
      fi
    fi

    if [ "${downloaded}" = false ]; then
      warn "Trying legacy release asset name."
      version_no_v="${version#v}"
      old_platform="$(legacy_platform "${platform}")"
      old_arch="$(legacy_arch "${arch}")"
      if [ "${platform}" = "windows" ]; then
        filename="doggo_${version_no_v}_${old_platform}_${old_arch}.zip"
      else
        filename="doggo_${version_no_v}_${old_platform}_${old_arch}.tar.gz"
      fi
      url="$(release_url "${version}" "${filename}")"
      info "Legacy download URL: ${url}"

      if ! download_file "${url}" "${filename}"; then
        error "Failed to download ${filename}"
        exit 1
      fi
    fi
  fi

  info "Verifying if file command exists"
  if ! command -v file > /dev/null 2>&1; then
      error "'file' command not found. Please install it."
      exit 1
  fi

  info "Verifying downloaded file..."
  if [ "${platform}" = "windows" ]; then
    if ! file "${filename}" | grep -q "Zip archive data"; then
      error "Downloaded file is not in zip format. Installation failed."
      error "File type:"
      file "${filename}"
      rm -f "${filename}"
      exit 1
    fi
  else
    if ! file "${filename}" | grep -q "gzip compressed data"; then
      error "Downloaded file is not in gzip format. Installation failed."
      error "File type:"
      file "${filename}"
      rm -f "${filename}"
      exit 1
    fi
  fi

  info "Extracting ${filename}..."
  extract_dir="doggo_extract"
  mkdir -p "${extract_dir}"
  if [ "${platform}" = "windows" ]; then
    if ! unzip -q "${filename}" -d "${extract_dir}"; then
      error "Failed to extract ${filename}"
      rm -rf "${filename}" "${extract_dir}"
      exit 1
    fi
  else
    if ! tar -xzf "${filename}" -C "${extract_dir}"; then
      error "Failed to extract ${filename}"
      rm -rf "${filename}" "${extract_dir}"
      exit 1
    fi
  fi

  info "Installing doggo..."
  binary_name="doggo"
  if [ "${platform}" = "windows" ]; then
    binary_name="doggo.exe"
  fi

  # Find the doggo binary in the extracted directory
  binary_path=$(find "${extract_dir}" -name "${binary_name}" -type f)

  if [ -z "${binary_path}" ]; then
    error "${binary_name} not found in the extracted files"
    error "Extracted files:"
    ls -R "${extract_dir}"
    rm -rf "${filename}" "${extract_dir}"
    exit 1
  fi

  chmod +x "${binary_path}"
  if ! sudo mv "${binary_path}" /usr/local/bin/doggo; then
    error "Failed to move doggo to /usr/local/bin/"
    rm -rf "${filename}" "${extract_dir}"
    exit 1
  fi

  info "Cleaning up..."
  rm -rf "${filename}" "${extract_dir}"

  completed "doggo ${version} has been installed to /usr/local/bin/doggo"
}

main() {
  printf '\n'

  if ! has curl && ! has wget; then
    error "Either curl or wget is required to download doggo. Please install one of them and try again."
    exit 1
  fi

  platform="$(detect_platform)"
  arch="$(detect_arch)"
  version="$(get_latest_release)"

  info "Latest doggo version: ${version}"
  info "Detected platform: ${platform}"
  info "Detected architecture: ${arch}"

  target="${platform}_${arch}"

  if ! is_supported_target "${target}"; then
    error "Unsupported target: ${target}"
    exit 1
  fi

  download_and_install "${version}" "${platform}" "${arch}"

  info "You can now use doggo by running 'doggo' in your terminal."
}

main
