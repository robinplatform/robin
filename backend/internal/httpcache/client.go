package httpcache

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"robinplatform.dev/internal/log"
)

type CacheClient struct {
	cache  Cache
	client *http.Client
}

func NewClient(filename string, maxSize int) (CacheClient, error) {
	cache, err := New(filename, maxSize)
	if err != nil {
		return CacheClient{}, err
	}

	client := &CacheClient{
		cache:  cache,
		client: &http.Client{},
	}
	return *client, nil
}

// parseCacheControl will attempt to parse the given `Cache-Control` header and
// extract a deadline for the cache entry, as well as a bool indicating whether the
// entry should be cached.
//
// A nil deadline indicates that the entry is immutable, and can be cached forever.
//
// This method will only attempt to parse the `max-age` directive, and ignore all other
// directives.
func parseCacheControl(cacheControl string) (*time.Duration, bool) {
	if cacheControl == "" {
		return nil, false
	}

	if strings.Contains(cacheControl, "immutable") {
		return nil, true
	}

	maxAgeStart := strings.Index(cacheControl, "max-age=")
	if maxAgeStart == -1 {
		return nil, false
	}

	maxAgeStart += len("max-age=")
	if cacheControl[maxAgeStart] == '"' {
		maxAgeStart++
	}

	maxAgeEnd := len(cacheControl)
	for i := maxAgeStart; i < len(cacheControl); i++ {
		if cacheControl[i] == ',' || cacheControl[i] == '"' {
			maxAgeEnd = i
			break
		}
	}

	maxAge, err := strconv.ParseInt(cacheControl[maxAgeStart:maxAgeEnd], 10, 64)
	if err != nil {
		logger.Warn("Failed to parse cache-control", log.Ctx{
			"cacheControl": cacheControl,
			"err":          err,
			"maxAgeValue":  cacheControl[maxAgeStart:maxAgeEnd],
		})
		return nil, false
	}
	if maxAge <= 0 {
		return nil, false
	}

	duration := time.Duration(maxAge) * time.Second
	return &duration, true
}

// parseAge will attempt to parse the given `Age` header and extract the elapsed duration
func parseAge(age string) time.Duration {
	if age == "" {
		return 0 * time.Second
	}

	elapsedSecs, err := strconv.ParseInt(age, 10, 64)
	if err != nil {
		logger.Warn("Failed to parse age header", log.Ctx{
			"age": age,
			"err": err,
		})
		return 0 * time.Second
	}
	if elapsedSecs <= 0 {
		return 0 * time.Second
	}

	return time.Duration(elapsedSecs) * time.Second
}

func (client *CacheClient) Get(targetUrl string) (string, bool, error) {
	if value, ok := client.cache.Get(targetUrl); ok {
		return value, true, nil
	}

	resp, err := client.client.Get(targetUrl)
	if err != nil {
		return "", false, err
	}
	defer resp.Body.Close()

	buf, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", false, err
	}

	if resp.StatusCode != http.StatusOK {
		return "", false, fmt.Errorf("failed to get %s: %s", targetUrl, string(buf))
	}

	cacheControl := resp.Header.Get("Cache-Control")
	maxAge, shouldCache := parseCacheControl(cacheControl)
	if !shouldCache {
		logger.Debug("HTTP resource is not cacheable", log.Ctx{
			"targetUrl":    targetUrl,
			"cacheControl": cacheControl,
		})
		return string(buf), false, nil
	}

	var ttl *time.Duration
	if maxAge != nil {
		age := parseAge(resp.Header.Get("Age"))
		ttlLocal := *maxAge - age
		ttl = &ttlLocal
	}

	client.cache.Set(targetUrl, string(buf), ttl)
	return string(buf), false, nil
}

func (client *CacheClient) GetCacheSize() int {
	return client.cache.GetSize()
}

func (client *CacheClient) Save() error {
	return client.cache.Save()
}
