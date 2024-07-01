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

SUPPORTED_TARGETS="linux_amd64 linux_arm64 windows_amd64 darwin_amd64 darwin_arm64"

get_latest_release() {
  curl --silent "https://api.github.com/repos/mr-karan/doggo/releases/latest" |
    grep '"tag_name":' |
    sed -E 's/.*"([^"]+)".*/\1/'
}

detect_platform() {
  platform="$(uname -s | tr '[:upper:]' '[:lower:]')"
  case "${platform}" in
    linux) platform="linux" ;;
    darwin) platform="darwin" ;;
    msys*|mingw*) platform="windows" ;;
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
    x86_64) arch="amd64" ;;
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
  filename="doggo_${version_no_v}_${platform}_${arch}.tar.gz"
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

  info "Verifying downloaded file..."
  if ! file "${filename}" | grep -q "gzip compressed data"; then
    error "Downloaded file is not in gzip format. Installation failed."
    error "File type:"
    file "${filename}"
    rm -f "${filename}"
    exit 1
  fi

  info "Extracting ${filename}..."
  if ! tar -xzvf "${filename}"; then
    error "Failed to extract ${filename}"
    rm -f "${filename}"
    exit 1
  fi

  info "Installing doggo..."
  if [ ! -f "doggo" ]; then
    error "doggo binary not found in the extracted files"
    error "Extracted files:"
    ls -la
    rm -f "${filename}"
    exit 1
  fi

  chmod +x doggo
  if ! sudo mv doggo /usr/local/bin/; then
    error "Failed to move doggo to /usr/local/bin/"
    rm -f "${filename}" "doggo"
    exit 1
  fi

  info "Cleaning up..."
  rm -f "${filename}"

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