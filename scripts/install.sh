#!/bin/bash
# she installer - curl | bash installation
# Usage: curl -fsSL https://antonysastre.github.io/sheets/install.sh | bash
#
# Or run locally after cloning:
#   ./scripts/install.sh
#
# Environment variables:
#   SHE_INSTALL_DIR - Override install directory (default: ~/.local/bin)
set -e

REPO="antonysastre/sheets"
BINARY_NAME="she"
DEFAULT_INSTALL_DIR="${HOME}/.local/bin"

detect_os() {
    case "$(uname -s)" in
        Linux*) echo "linux" ;;
        Darwin*) echo "darwin" ;;
        *) echo "unsupported" ;;
    esac
}

detect_arch() {
    case "$(uname -m)" in
        x86_64|amd64) echo "amd64" ;;
        arm64|aarch64) echo "arm64" ;;
        *) echo "unsupported" ;;
    esac
}

get_latest_version() {
    local version
    version=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" 2>/dev/null | grep '"tag_name"' | sed -E 's/.*"tag_name": *"v?([^"]+)".*/\1/')
    echo "${version#v}"
}

error() {
    echo "Error: $1" >&2
    exit 1
}

install() {
    local os
    local arch
    local version
    local archive_name
    local download_url
    local install_dir
    local tmpdir

    os=$(detect_os)
    arch=$(detect_arch)

    if [[ "$os" == "unsupported" ]]; then
        error "Unsupported OS. Get a release from https://github.com/${REPO}/releases"
    fi

    if [[ "$arch" == "unsupported" ]]; then
        error "Unsupported architecture. Get a release from https://github.com/${REPO}/releases"
    fi

    version=$(get_latest_version)
    if [[ -z "$version" ]]; then
        error "Failed to fetch latest version. Please check your internet connection."
    fi

    archive_name="${BINARY_NAME}_${os}_${arch}.tar.gz"
    download_url="https://github.com/${REPO}/releases/download/v${version}/${archive_name}"

    echo "Installing ${BINARY_NAME} v${version} for ${os}/${arch}..."

    tmpdir=$(mktemp -d)
    trap "rm -rf $tmpdir" EXIT

    echo "Downloading..."
    curl -fsSL -o "${tmpdir}/${archive_name}" "$download_url" || error "Failed to download release. Check releases at https://github.com/${REPO}/releases"

    echo "Extracting..."
    tar -xzf "${tmpdir}/${archive_name}" -C "$tmpdir"

    install_dir="${SHE_INSTALL_DIR:-$DEFAULT_INSTALL_DIR}"
    mkdir -p "$install_dir"

    if cp "${tmpdir}/${BINARY_NAME}" "$install_dir/${BINARY_NAME}"; then
        chmod +x "$install_dir/${BINARY_NAME}"
        echo "Installed to ${install_dir}/${BINARY_NAME}"

        if [[ ":$PATH:" != *":${install_dir}:"* ]]; then
            echo ""
            echo "Add to your PATH:"
            echo "  export PATH=\"\${HOME}/.local/bin:\$PATH\""
            echo ""
            echo "Or add this to your ~/.bashrc or ~/.zshrc:"
            echo "  echo 'export PATH=\"\${HOME}/.local/bin:\$PATH\"' >> ~/.bashrc"
        fi
    else
        error "Failed to copy binary to ${install_dir}"
    fi
}

main() {
    echo "she installer"
    echo "============"
    echo ""
    install
    echo ""
    echo "Done! Run 'she --help' to get started."
}

main "$@"