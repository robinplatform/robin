package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"time"

	"robinplatform.dev/internal/config"
	"robinplatform.dev/internal/upgrade"
)

var startTime = time.Now()

func main() {
	var channel string
	flag.StringVar(&channel, "channel", "", "The release channel to use")
	flag.Parse()

	if channel == "" {
		fmt.Fprintf(os.Stderr, "No channel specified\n")
		fmt.Fprintf(os.Stderr, "Usage: robin-upgrade -channel <channel>\n")
		os.Exit(1)
	}

	var releaseChannel config.ReleaseChannel
	if err := releaseChannel.Parse(channel); err != nil {
		panic(err)
	}

	action := "Upgrading"
	if _, err := os.Stat(releaseChannel.GetInstallationPath()); os.IsNotExist(err) {
		action = "Installing"
	}

	fmt.Printf("%s %s ...", action, releaseChannel)
	updatedVersion, execName, err := upgrade.UpgradeChannel(releaseChannel)
	if err != nil {
		fmt.Printf("\rFailed to upgrade %s: %s\n", releaseChannel, err)
		os.Exit(1)
	}
	fmt.Printf("\rSuccessfully upgraded %s to %s in %s!\n", releaseChannel, updatedVersion, time.Since(startTime).Truncate(time.Millisecond))
	fmt.Printf("\n")
	fmt.Printf("Installed into ~/.robin/bin/%s\n", execName)

	if _, err := exec.LookPath(execName); err != nil {
		fmt.Printf("~/.robin/bin was not found in your $PATH, you may want to add it.\n")
	}
}
