package main

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/pflag"
	"robinplatform.dev/internal/config"
	"robinplatform.dev/internal/upgrade"
)

var startTime = time.Now()

func main() {
	var channel string
	pflag.StringVar(&channel, "channel", "stable", "The release channel to use")
	pflag.Parse()

	var releaseChannel config.ReleaseChannel
	if err := releaseChannel.Parse(channel); err != nil {
		panic(err)
	}

	fmt.Printf("Upgrading %s ...", releaseChannel)
	updatedVersion, err := upgrade.UpgradeChannel(releaseChannel)
	if err != nil {
		fmt.Printf("\rFailed to upgrade %s: %s\n", releaseChannel, err)
		os.Exit(1)
	}
	fmt.Printf("\rSuccessfully upgraded %s to %s in %s!\n", releaseChannel, updatedVersion, time.Since(startTime).Truncate(time.Millisecond))
}
