#!/usr/bin/env sh

set -eu
printf '\n'

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

SUPPORTED_TARGETS="Linux_x86_64 Linux_arm64 Windows_x86_64 Darwin_x86_64 Darwin_arm64"

get_latest_release() {
  curl --silent "https://api.github.com/repos/mr-karan/doggo/releases/latest" |
    grep '"tag_name":' |
    sed -E 's/.*"([^"]+)".*/\1/'
}

detect_platform() {
  platform="$(uname -s)"
  case "${platform}" in
    Linux*) platform="Linux" ;;
    Darwin*) platform="Darwin" ;;
    MINGW*|MSYS*|CYGWIN*) platform="Windows" ;;
    *)
      error "Unsupported platform: ${platform}"
      exit 1
      ;;
  esac
  printf '%s' "${platform}"
}

detect_arch() {
  arch="$(uname -m)"
  case "${arch}" in
    x86_64) arch="x86_64" ;;
    aarch64|arm64) arch="arm64" ;;
    *)
      error "Unsupported architecture: ${arch}"
      exit 1
      ;;
  esac
  printf '%s' "${arch}"
}

download_and_install() {
  version="$1"
  platform="$2"
  arch="$3"

  # Remove 'v' prefix from version for filename
  version_no_v="${version#v}"
  if [ "${platform}" = "Windows" ]; then
    filename="doggo_${version_no_v}_${platform}_${arch}.zip"
  else
    filename="doggo_${version_no_v}_${platform}_${arch}.tar.gz"
  fi
  url="https://github.com/mr-karan/doggo/releases/download/${version}/${filename}"

  info "Downloading doggo ${version} for ${platform}_${arch}..."
  info "Download URL: ${url}"

  if has curl; then
    if ! curl -sSL "${url}" -o "${filename}"; then
      error "Failed to download ${filename}"
      error "Curl output:"
      curl -SL "${url}"
      exit 1
    fi
  elif has wget; then
    if ! wget -q "${url}" -O "${filename}"; then
      error "Failed to download ${filename}"
      error "Wget output:"
      wget "${url}"
      exit 1
    fi
  else
    error "Neither curl nor wget found. Please install one of them and try again."
    exit 1
  fi

  info "Verifying if file command exists"
  if ! command -v file > /dev/null 2>&1; then
      error "'file' command not found. Please install it."
      exit 1
  fi

  info "Verifying downloaded file..."
  if [ "${platform}" = "Windows" ]; then
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
  if [ "${platform}" = "Windows" ]; then
    if ! unzip -q "${filename}" -d "${extract_dir}"; then
      error "Failed to extract ${filename}"
      rm -rf "${filename}" "${extract_dir}"
      exit 1
    fi
  else
    if ! tar -xzvf "${filename}" -C "${extract_dir}"; then
      error "Failed to extract ${filename}"
      rm -rf "${filename}" "${extract_dir}"
      exit 1
    fi
  fi

  info "Installing doggo..."
  binary_name="doggo"
  if [ "${platform}" = "Windows" ]; then
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

  if ! echo "${SUPPORTED_TARGETS}" | grep -q "${target}"; then
    error "Unsupported target: ${target}"
    exit 1
  fi

  download_and_install "${version}" "${platform}" "${arch}"

  info "You can now use doggo by running 'doggo' in your terminal."
}

main
