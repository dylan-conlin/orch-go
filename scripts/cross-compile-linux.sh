#!/bin/bash
# Cross-compile Go binaries for Linux amd64 (Docker containers)
#
# Usage: ./scripts/cross-compile-linux.sh [--all|--bd|--orch|--kb|--skillc]
#
# Output: ~/.local/bin/linux-amd64/{bd,orch,kb,skillc}
#
# Context: Docker containers are Linux but host is macOS. Volume mounting
# macOS binaries fails with 'Exec format error'. This script builds Linux
# binaries that can be volume-mounted into Docker containers.

set -e

OUTPUT_DIR="${HOME}/.local/bin/linux-amd64"
GOOS=linux
GOARCH=amd64

# Project directories
BD_DIR="${HOME}/Documents/personal/beads"
ORCH_DIR="${HOME}/Documents/personal/orch-go"
KB_DIR="${HOME}/Documents/personal/kb-cli"
SKILLC_DIR="${HOME}/Documents/personal/skillc"

# Build flags (match each project's Makefile patterns)
get_bd_ldflags() {
    cd "$BD_DIR"
    local commit=$(git rev-parse HEAD 2>/dev/null || echo "unknown")
    local short_commit=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
    local branch=$(git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "unknown")
    echo "-X main.Build=${short_commit} -X main.Commit=${commit} -X main.Branch=${branch} -X main.SourceDir=${BD_DIR}"
}

get_orch_ldflags() {
    cd "$ORCH_DIR"
    local version=$(git describe --tags --always --dirty 2>/dev/null || echo "dev")
    local build_time=$(date -u '+%Y-%m-%dT%H:%M:%SZ')
    local git_hash=$(git rev-parse HEAD 2>/dev/null || echo "unknown")
    echo "-X main.version=${version} -X main.buildTime=${build_time} -X main.sourceDir=${ORCH_DIR} -X main.gitHash=${git_hash}"
}

get_kb_ldflags() {
    cd "$KB_DIR"
    local version=$(git describe --tags --always --dirty 2>/dev/null || echo "dev")
    local build_time=$(date -u '+%Y-%m-%dT%H:%M:%SZ')
    local git_hash=$(git rev-parse HEAD 2>/dev/null || echo "unknown")
    echo "-X main.Version=${version} -X main.BuildTime=${build_time} -X main.SourceDir=${KB_DIR} -X main.GitHash=${git_hash}"
}

get_skillc_ldflags() {
    cd "$SKILLC_DIR"
    local version=$(git describe --tags --always --dirty 2>/dev/null || echo "dev")
    local commit=$(git rev-parse HEAD 2>/dev/null || echo "none")
    local build_time=$(date -u '+%Y-%m-%dT%H:%M:%SZ')
    local git_hash=$(git rev-parse HEAD 2>/dev/null || echo "unknown")
    echo "-X main.Version=${version} -X main.Commit=${commit} -X main.BuildTime=${build_time} -X main.SourceDir=${SKILLC_DIR} -X main.GitHash=${git_hash}"
}

build_bd() {
    echo "Building bd for linux/amd64..."
    if [[ ! -d "$BD_DIR" ]]; then
        echo "Error: beads directory not found at $BD_DIR"
        return 1
    fi
    cd "$BD_DIR"
    local ldflags=$(get_bd_ldflags)
    GOOS=$GOOS GOARCH=$GOARCH go build -ldflags="$ldflags" -o "${OUTPUT_DIR}/bd" ./cmd/bd
    echo "  -> ${OUTPUT_DIR}/bd"
}

build_orch() {
    echo "Building orch for linux/amd64..."
    if [[ ! -d "$ORCH_DIR" ]]; then
        echo "Error: orch-go directory not found at $ORCH_DIR"
        return 1
    fi
    cd "$ORCH_DIR"
    local ldflags=$(get_orch_ldflags)
    GOOS=$GOOS GOARCH=$GOARCH go build -ldflags="$ldflags" -o "${OUTPUT_DIR}/orch" ./cmd/orch/
    echo "  -> ${OUTPUT_DIR}/orch"
}

build_kb() {
    echo "Building kb for linux/amd64..."
    if [[ ! -d "$KB_DIR" ]]; then
        echo "Error: kb-cli directory not found at $KB_DIR"
        return 1
    fi
    cd "$KB_DIR"
    local ldflags=$(get_kb_ldflags)
    GOOS=$GOOS GOARCH=$GOARCH go build -ldflags="$ldflags" -o "${OUTPUT_DIR}/kb" ./cmd/kb/
    echo "  -> ${OUTPUT_DIR}/kb"
}

build_skillc() {
    echo "Building skillc for linux/amd64..."
    if [[ ! -d "$SKILLC_DIR" ]]; then
        echo "Error: skillc directory not found at $SKILLC_DIR"
        return 1
    fi
    cd "$SKILLC_DIR"
    local ldflags=$(get_skillc_ldflags)
    GOOS=$GOOS GOARCH=$GOARCH go build -ldflags="$ldflags" -o "${OUTPUT_DIR}/skillc" ./cmd/skillc/
    echo "  -> ${OUTPUT_DIR}/skillc"
}

build_all() {
    build_bd
    build_orch
    build_kb
    build_skillc
}

main() {
    echo "Cross-compiling Go binaries for linux/amd64"
    echo "Output: ${OUTPUT_DIR}"
    echo ""

    # Create output directory
    mkdir -p "$OUTPUT_DIR"

    case "${1:-all}" in
        --bd)
            build_bd
            ;;
        --orch)
            build_orch
            ;;
        --kb)
            build_kb
            ;;
        --skillc)
            build_skillc
            ;;
        --all|all|"")
            build_all
            ;;
        *)
            echo "Usage: $0 [--all|--bd|--orch|--kb|--skillc]"
            exit 1
            ;;
    esac

    echo ""
    echo "Done. Linux binaries are in ${OUTPUT_DIR}"
    echo ""
    echo "These binaries will be used by Docker spawns via orch spawn --backend docker"
}

main "$@"
