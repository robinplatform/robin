#!/bin/bash
set -eo pipefail

cd "$(dirname $0)/.."

# Validate target channel
TARGET_CHANNEL="$1"
if test -z "$TARGET_CHANNEL"; then
    echo "Usage: $0 <target-channel>"
    echo ""
    exit 1
fi
if test "$TARGET_CHANNEL" != "stable" && test "$TARGET_CHANNEL" != "beta" && test "$TARGET_CHANNEL" != "nightly"; then
    echo "Invalid target channel: $TARGET_CHANNEL"
    echo ""
    exit 1
fi

# Check dependencies exist
if ! test -e "../frontend/out"; then
    echo "frontend must be built first"
    echo ""
    exit 1
fi
if ! which s3cmd &>/dev/null; then
    echo "s3cmd is not installed"
    echo ""
    exit 1
fi
if test "$TARGET_CHANNEL" == "stable" && ! which jq &>/dev/null; then
    echo "jq is not installed"
    echo ""
    exit 1
fi

# Verify that s3cmd is configured correctly
if ! s3cmd ls "s3://robinplatform/releases/${TARGET_CHANNEL}" &>/dev/null; then
    echo "s3cmd is not configured, or the target channel does not exist: $TARGET_CHANNEL"
    echo ""
    exit 1
fi

# Generation is not platform specific, so we will just generate once
go generate -tags prod -x ./...

# Figure out release version
echo ""
if test -z "$ROBIN_VERSION"; then
    export ROBIN_VERSION=`git describe --tags --always`
fi
if test "$TARGET_CHANNEL" == "stable" && (test "${ROBIN_VERSION:0:1}" != "v" || echo "$ROBIN_VERSION" | grep '-' &>/dev/null); then
    echo "Latest commit is not tagged, cannot release to stable channel"
    echo ""
    exit 1
fi

# Verify that the version is not already published, if stable
if test "$TARGET_CHANNEL" == "stable" && ! test -z "`s3cmd ls s3://robinplatform/releases/stable/${ROBIN_VERSION}`"; then
    echo "Version already published: $ROBIN_VERSION"
    echo ""
    exit 1
fi

echo "Building version: $ROBIN_VERSION"

buildDir=`mktemp -d`
mkdir -p "${buildDir}/tarballs"

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
        echo -n "$ROBIN_VERSION" > "${platformDir}/VERSION"
        mkdir ${platformDir}/bin

        GOOS=$platform GOARCH=$arch go build \
            -o "${platformDir}/bin/robin${ext}" \
            -tags prod \
            -ldflags "-X robinplatform.dev/internal/config.robinVersion=${ROBIN_VERSION}" \
            ./cmd/cli
        GOOS=$platform GOARCH=$arch go build \
            -o "${platformDir}/bin/robin-upgrade${ext}" \
            -tags prod \
            -ldflags "-X robinplatform.dev/internal/config.robinVersion=${ROBIN_VERSION}" \
            ./cmd/upgrade

        cd "${platformDir}"

        tar czf "${buildDir}/tarballs/robin-${platform}-${arch}.tar.gz" .

        binSize=`du -h "${platformDir}/bin/robin${ext}" | awk '{print $1}'`
        size=`du -h "${buildDir}/tarballs/robin-${platform}-${arch}.tar.gz" | awk '{print $1}'`

        echo -e "\rBuilt: robin-${platform}-${arch}.tar.gz (size: ${size}, binary size: ${binSize})"

        # For stable releases, upload the upgrade binary separately
        if test "$TARGET_CHANNEL" == "stable"; then
            mkdir -p "${buildDir}/installers"
            cp "bin/robin-upgrade${ext}" "${buildDir}/installers/robin-upgrade-${platform}-${arch}${ext}"
        fi

        cd $OLDPWD
        rm -rf "${platformDir}"
    done
done

echo ""
echo "Publishing assets to CDN ..."
echo ""

cd "$buildDir/tarballs"
if test "$TARGET_CHANNEL" == "stable"; then
    s3cmd put * "s3://robinplatform/releases/${TARGET_CHANNEL}/${ROBIN_VERSION}/" --acl-public --cf-invalidate
else
    s3cmd put * "s3://robinplatform/releases/${TARGET_CHANNEL}/" --acl-public --cf-invalidate
fi

echo -n "$ROBIN_VERSION" > latest.txt
s3cmd put latest.txt "s3://robinplatform/releases/${TARGET_CHANNEL}/latest.txt" --acl-public --cf-invalidate

if test "$TARGET_CHANNEL" == "stable"; then
    echo ""
    echo "Publishing installers to CDN ..."
    echo ""

    cd "$buildDir/installers"
    s3cmd put * "s3://robinplatform/releases/installers/" --acl-public --cf-invalidate
fi

echo ""
echo "Released to $TARGET_CHANNEL"
echo ""

cd "$buildDir/.."
rm -rf "$buildDir"
