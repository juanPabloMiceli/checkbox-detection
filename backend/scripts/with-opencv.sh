#!/usr/bin/env bash
set -euo pipefail

# Use an explicit installation, pkg-config's search path, or the local Mac setup.
prefix="${OPENCV_PREFIX:-}"
if [[ -z "$prefix" ]] && ! pkg-config --exists opencv4; then
  prefix=/opt/opencv-4.13
fi
if [[ -n "$prefix" ]]; then
  export PKG_CONFIG_PATH="$prefix/lib/pkgconfig${PKG_CONFIG_PATH:+:$PKG_CONFIG_PATH}"
fi
if ! pkg-config --exact-version=4.13.0 opencv4; then
  printf '%s\n' 'OpenCV 4.13.0 is required. Set OPENCV_PREFIX or use Docker.' >&2
  exit 1
fi

# Embed the native library path so binaries and tests work without shell setup.
lib_dir="$(pkg-config --variable=libdir opencv4)"
go_command="$1"
subcommand="$2"
shift 2
case "$subcommand" in
  build|test|run) exec "$go_command" "$subcommand" "-ldflags=-extldflags=-Wl,-rpath,$lib_dir" "$@" ;;
  *) exec "$go_command" "$subcommand" "$@" ;;
esac
