#!/bin/bash
set -eo pipefail

function get_goos() {
    case $(uname -s) in
        Darwin) echo "darwin" ;;
        Linux) echo "linux" ;;
        MINGW*) echo "windows" ;;
        *) echo "unknown" ;;
    esac
}

function get_goarch() {
    case $(uname -m) in
        x86_64) echo "amd64" ;;
        arm64) echo "arm64" ;;
        *) echo "unknown" ;;
    esac
}

if test -z "$CHANNEL"; then
    export CHANNEL="stable"
fi

installer=`mktemp`
curl -fsSL -o "$installer" "http://robinplatform.nyc3.digitaloceanspaces.com/releases/installers/robin-upgrade-$(get_goos)-$(get_goarch)"
chmod +x "$installer"
"$installer" --channel=$CHANNEL
