package compilerServer

import (
	"robinplatform.dev/internal/config"
	"robinplatform.dev/internal/httpcache"
	"robinplatform.dev/internal/log"
)

var httpClient httpcache.CacheClient

func init() {
	var err error
	cacheFilename := config.GetHttpCachePath()
	httpClient, err = httpcache.NewClient(cacheFilename, 100*1024*1024)
	if err != nil {
		httpLogger := log.New("http")
		httpLogger.Debug("Failed to load HTTP cache, will recreate", log.Ctx{
			"error": err,
			"path":  cacheFilename,
		})
	}
}
