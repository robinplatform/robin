package upgrade

import (
	"archive/tar"
	"compress/gzip"
	"crypto/rand"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"robinplatform.dev/internal/config"
	"robinplatform.dev/internal/log"
)

var logger = log.New("upgrade")

func createTempDir() (string, error) {
	buf := make([]byte, 4)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}

	tmp := filepath.Join(os.TempDir(), fmt.Sprintf("robin-upgrade-%x", buf))
	if err := os.Mkdir(tmp, 0755); os.IsExist(err) {
		return createTempDir()
	} else if err != nil {
		return "", err
	}
	return tmp, nil
}

func WatchForUpdates() {
	releaseChannel := config.GetReleaseChannel()
	installedVersion := config.GetRobinVersion()

	for {
		latestVersion, err := getLatestVersion(releaseChannel)
		if err != nil {
			logger.Debug("Failed to get latest version", log.Ctx{
				"channel": releaseChannel,
				"error":   err,
			})
		} else if latestVersion != installedVersion {
			newVersion, _, err := UpgradeChannel(releaseChannel)
			if err != nil {
				logger.Debug("Failed to auto-upgrade to latest version", log.Ctx{
					"channel":       releaseChannel,
					"latestVersion": latestVersion,
					"error":         err,
				})
			} else {
				logger.Info("Upgraded to latest version", log.Ctx{
					"previousVersion": installedVersion,
					"channel":         releaseChannel,
					"version":         newVersion,
				})
				installedVersion = newVersion
			}
		}

		time.Sleep(1 * time.Hour)
	}
}

func UpgradeChannel(releaseChannel config.ReleaseChannel) (string, string, error) {
	// Figure out where to download the new version from
	var assetEndpoint string
	if releaseChannel == config.ReleaseChannelStable {
		latestVersion, err := getLatestVersion(releaseChannel)
		if err != nil {
			return "", "", fmt.Errorf("failed to get latest version: %w", err)
		}

		latestVersion = strings.TrimSpace(latestVersion)
		assetEndpoint = getTarEndpoint(releaseChannel, latestVersion)
	} else {
		assetEndpoint = getTarEndpoint(releaseChannel, "")
	}

	// Create a temporary directory to download the new version into
	// The idea behind this is to allow the upgrade to be cancelled partway through without
	// leaving the user with a broken installation. So we download the new version into a
	// temporary directory, and then move it into place once the download is complete.
	tmp, err := createTempDir()
	if err != nil {
		return "", "", fmt.Errorf("failed to create temporary directory: %w", err)
	}

	res, err := http.Get(getCdnEndpoint(assetEndpoint))
	if err != nil {
		return "", "", fmt.Errorf("failed to download tarball for %s: %w", releaseChannel, err)
	}

	gzipReader, err := gzip.NewReader(res.Body)
	if err != nil {
		return "", "", fmt.Errorf("failed to decompress tarball for %s: %w", releaseChannel, err)
	}

	tarReader := tar.NewReader(gzipReader)
	robinVersion := ""
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", "", fmt.Errorf("failed to read tarball for %s: %w", releaseChannel, err)
		}

		// Create directories as needed, copying permissions from the tar header
		if header.Typeflag == tar.TypeDir {
			os.Mkdir(filepath.Join(tmp, header.Name), header.FileInfo().Mode())
			continue
		}

		// Skip non-files, and also whatever macOS puts in tarballs
		if header.Typeflag != tar.TypeReg || filepath.Base(header.Name)[0:2] == "._" {
			continue
		}

		file, err := os.Create(filepath.Join(tmp, header.Name))
		if err != nil {
			return "", "", fmt.Errorf("failed to upgrade %s: error while downloading %s: %w", releaseChannel, header.Name, err)
		}

		_, err = io.Copy(file, tarReader)
		if err != nil {
			return "", "", fmt.Errorf("failed to upgrade %s: error while downloading %s: %w", releaseChannel, header.Name, err)
		}

		file.Chmod(header.FileInfo().Mode())
		file.Close()

		// Read the VERSION file to figure out what version we just downloaded
		if header.Name == "./VERSION" {
			buf, err := os.ReadFile(filepath.Join(tmp, header.Name))
			if err != nil {
				return "", "", fmt.Errorf("failed to upgrade %s: error while reading VERSION file: %w", releaseChannel, err)
			}
			robinVersion = strings.TrimSpace(string(buf))
		}
	}

	channelDir := releaseChannel.GetInstallationPath()

	// Make sure the `.robin/channels` directory exists
	if err := os.MkdirAll(filepath.Dir(channelDir), 0755); err != nil {
		return "", "", err
	}

	// Delete the old channel directory
	if err := os.RemoveAll(channelDir); err != nil && !os.IsNotExist(err) {
		return "", "", err
	}

	// Move the new dir
	// TODO: this is apparently not atomic on windows
	if err := os.Rename(tmp, channelDir); err != nil {
		return "", "", err
	}

	// Make sure the general `bin` directory exists
	if err := os.MkdirAll(filepath.Join(config.GetRobinPath(), "bin"), 0755); err != nil {
		return "", "", err
	}

	// Make sure symlink exists for the new version
	targetExecName := "robin"
	linkExecName := "robin"
	upgradeExecName := "robin-upgrade"

	if releaseChannel != config.ReleaseChannelStable {
		linkExecName += "-" + string(releaseChannel)
	}
	if runtime.GOOS == "windows" {
		targetExecName += ".exe"
		linkExecName += ".exe"
		upgradeExecName += ".exe"
	}

	// delete and recreate symlink
	if err := os.Remove(filepath.Join(config.GetRobinPath(), "bin", linkExecName)); err != nil && !os.IsNotExist(err) {
		return robinVersion, "", fmt.Errorf("failed to upgrade %s: error while deleting symlink: %w", releaseChannel, err)
	}
	if err := os.Symlink(filepath.Join(channelDir, "bin", "robin"), filepath.Join(config.GetRobinPath(), "bin", linkExecName)); err != nil {
		return robinVersion, "", fmt.Errorf("failed to upgrade %s: error while creating symlink: %w", releaseChannel, err)
	}

	if releaseChannel == config.ReleaseChannelStable {
		// delete and recreate upgrade symlink
		if err := os.Remove(filepath.Join(config.GetRobinPath(), "bin", upgradeExecName)); err != nil && !os.IsNotExist(err) {
			return robinVersion, "", fmt.Errorf("failed to upgrade %s: error while deleting symlink: %w", releaseChannel, err)
		}
		if err := os.Symlink(filepath.Join(channelDir, "bin", upgradeExecName), filepath.Join(config.GetRobinPath(), "bin", upgradeExecName)); err != nil {
			return robinVersion, "", fmt.Errorf("failed to upgrade %s: error while creating symlink: %w", releaseChannel, err)
		}
	}

	return robinVersion, linkExecName, nil
}
