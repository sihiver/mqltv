#!/bin/sh
set -eu

ROOT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
SDK_DIR="$ROOT_DIR/openwrt-sdk-24.10.4-ipq40xx-generic_gcc-13.3.0_musl_eabi.Linux-x86_64"
TC_DIR="$ROOT_DIR/openwrt-toolchain-24.10.4-ipq40xx-generic_gcc-13.3.0_musl_eabi.Linux-x86_64"

export STAGING_DIR="$SDK_DIR/staging_dir"

# Use absolute compiler path so PATH issues don't matter.
exec "$TC_DIR"/toolchain-arm_cortex-a7+neon-vfpv4_gcc-13.3.0_musl_eabi/bin/arm-openwrt-linux-muslgnueabi-gcc "$@"
