package upgrade

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"runtime"

	"robinplatform.dev/internal/config"
)

func getCdnEndpoint(filepath string) string {
	return fmt.Sprintf("https://robinplatform.nyc3.cdn.digitaloceanspaces.com/%s", url.PathEscape(filepath))
}

func getTarEndpoint(channel config.ReleaseChannel, version string) string {
	if channel == config.ReleaseChannelStable {
		return fmt.Sprintf("releases/stable/%s/robin-%s-%s.tar.gz", version, runtime.GOOS, runtime.GOARCH)
	}
	return fmt.Sprintf("releases/%s/robin-%s-%s.tar.gz", channel, runtime.GOOS, runtime.GOARCH)
}

func fetchString(filepath string) (string, error) {
	endpoint := getCdnEndpoint(filepath)
	res, err := http.Get(endpoint)
	if err != nil {
		return "", fmt.Errorf("failed to fetch %s: %w", endpoint, err)
	}
	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("fetching %s resulted in status code %d", endpoint, res.StatusCode)
	}

	buf, err := io.ReadAll(res.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read %s: %w", endpoint, err)
	}

	return string(buf), nil
}

func getLatestVersion(channel config.ReleaseChannel) (string, error) {
	return fetchString(fmt.Sprintf("releases/%s/latest.txt", channel))
}
