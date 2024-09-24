#!/bin/bash

export CLI_VERSION=$GITHUB_REF_NAME
export MACOS_AMD64_SHA256=$(sha256sum "_output/ndc-prometheus-darwin-amd64" | awk '{ print $1 }')
export MACOS_ARM64_SHA256=$(sha256sum "_output/ndc-prometheus-darwin-arm64" | awk '{ print $1 }')
export LINUX_AMD64_SHA256=$(sha256sum "_output/ndc-prometheus-linux-amd64" | awk '{ print $1 }')
export LINUX_ARM64_SHA256=$(sha256sum "_output/ndc-prometheus-linux-arm64" | awk '{ print $1 }')
export WINDOWS_AMD64_SHA256=$(sha256sum "_output/ndc-prometheus-windows-amd64.exe" | awk '{ print $1 }')

envsubst < .github/scripts/plugin-manifest.yaml > release/manifest.yaml