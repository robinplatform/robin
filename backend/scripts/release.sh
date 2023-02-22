#!/bin/bash
set -eo pipefail

cd "$(dirname $0)/.."

# Validate target channel
TARGET_CHANNEL="$1"
if test -z "$TARGET_CHANNEL"; then
    echo "Usage: $0 <target-channel>"
    exit 1
fi
if test "$TARGET_CHANNEL" != "stable" && test "$TARGET_CHANNEL" != "beta" && test "$TARGET_CHANNEL" != "nightly"; then
    echo "Invalid target channel: $TARGET_CHANNEL"
    exit 1
fi

# Check dependencies exist
if ! test -e "../frontend/out"; then
    echo "frontend must be built first"
    exit 1
fi
if ! which s3cmd &>/dev/null; then
    echo "s3cmd is not installed"
    exit 1
fi
if test "$TARGET_CHANNEL" == "stable" && ! which jq &>/dev/null; then
    echo "jq is not installed"
    exit 1
fi

# Verify that s3cmd is configured correctly
if ! s3cmd ls "s3://robinplatform/releases/${TARGET_CHANNEL}" &>/dev/null; then
    echo "s3cmd is not configured, or the target channel does not exist: $TARGET_CHANNEL"
    exit 1
fi

# Generation is not platform specific, so we will just generate once
go generate -tags prod -x ./...

buildDir=`mktemp -d`

echo ""
echo "Temporary build directory: $buildDir"
echo ""

for platform in darwin linux windows; do
    for arch in amd64 arm64; do
        ext=""
        if test "$platform" = "windows"; then
            ext=".exe"
        fi

        if [ -t 1 ]; then
            echo -n "Building for: ${platform}/${arch}"
        fi

        platformDir="${buildDir}/${platform}-${arch}"
        mkdir -p "${platformDir}"

        cp ../LICENSE ${platformDir}
        mkdir ${platformDir}/bin

        GOOS=$platform GOARCH=$arch go build \
            -o "${platformDir}/bin/robin${ext}" \
            -tags prod \
            ./cmd/cli

        cd "${platformDir}"

        tar czf "../robin-${platform}-${arch}.tar.gz" .
        cat "../robin-${platform}-${arch}.tar.gz" | shasum -a 256 | awk '{print $1}' > "../robin-${platform}-${arch}.tar.gz.sha256"

        binSize=`du -h "${platformDir}/bin/robin${ext}" | awk '{print $1}'`
        size=`du -h "../robin-${platform}-${arch}.tar.gz" | awk '{print $1}'`
        sha256=`cat "../robin-${platform}-${arch}.tar.gz.sha256"`

        echo -e "\rBuilt: robin-${platform}-${arch}.tar.gz (size: ${size}, binary size: ${binSize}, sha256: ${sha256})"

        cd $OLDPWD
        rm -rf "${platformDir}"
    done
done

echo ""
echo "Publishing assets to CDN ..."
echo ""

cd "$buildDir"
s3cmd put `find . -type f` "s3://robinplatform/releases/${TARGET_CHANNEL}/"

echo ""
echo "Released to $TARGET_CHANNEL"
echo ""
