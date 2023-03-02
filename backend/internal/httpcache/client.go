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

// NewClient will create a new CacheClient, which will use the given filename as the
// backing cache file. The cache will be limited to the given size, in bytes. The exact
// size of the cache file on disk might be slightly larger than the given size, due to
// the overhead of the cache file format. If an empty string is given for the filename,
// loading the cache will be skipped.
func NewClient(filename string, maxSize int) (CacheClient, error) {
	// Since the cache is always valid, we will always return a valid client
	cache, err := New(filename, maxSize)
	return CacheClient{
		cache:  cache,
		client: &http.Client{},
	}, err
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

// Head will perform a HEAD request to the given URL, and return a bool indicating whether
// the resource exists. If a copy of the resource is cached, the HEAD request will not be
// performed. The HEAD request will never be cached.
func (client *CacheClient) Head(targetUrl string) (bool, error) {
	if _, ok := client.cache.Get(targetUrl); ok {
		return true, nil
	}

	resp, err := client.client.Head(targetUrl)
	if err != nil {
		return false, err
	}
	resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return false, nil
	}
	if resp.StatusCode == http.StatusOK {
		return true, nil
	}

	return false, HttpError{
		URL:        targetUrl,
		StatusCode: resp.StatusCode,
		Status:     resp.Status,
	}
}

type HttpError struct {
	URL        string
	StatusCode int
	Status     string
}

func (err HttpError) Error() string {
	return err.Status
}

// Get will perform a GET request to the given URL, and return the response body. If a copy
// of the resource is cached, the GET request will not be performed. The GET request will
// be cached if the `Cache-Control` header contains a `max-age` or `immutable` directive.
func (client *CacheClient) Get(targetUrl string) (string, bool, error) {
	if entry, ok := client.cache.Get(targetUrl); ok {
		if entry.StatusCode != http.StatusOK {
			return "", false, HttpError{
				URL:        targetUrl,
				StatusCode: entry.StatusCode,
				Status:     fmt.Sprintf("HTTP %d", entry.StatusCode),
			}
		}

		return entry.Value, true, nil
	}

	logger.Debug("HTTP fetching", log.Ctx{
		"targetUrl": targetUrl,
	})
	requestStartTime := time.Now()
	resp, err := client.client.Get(targetUrl)
	if err != nil {
		return "", false, err
	}
	defer resp.Body.Close()

	buf, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", false, err
	}

	duration := time.Since(requestStartTime)
	if duration >= 50*time.Millisecond {
		logger.Debug("HTTP request took a long time", log.Ctx{
			"targetUrl": targetUrl,
			"duration":  duration.String(),
		})
	}

	cacheControl := resp.Header.Get("Cache-Control")
	maxAge, shouldCache := parseCacheControl(cacheControl)
	if shouldCache {
		entry := CacheEntry{
			Value:      string(buf),
			StatusCode: resp.StatusCode,
		}
		if maxAge != nil {
			age := parseAge(resp.Header.Get("Age"))
			ttlLocal := *maxAge - age

			deadline := time.Now().Add(ttlLocal).UnixNano()
			entry.Deadline = &deadline
		}

		client.cache.Set(targetUrl, entry)
	} else {
		logger.Debug("HTTP resource is not cacheable", log.Ctx{
			"targetUrl":    targetUrl,
			"cacheControl": cacheControl,
		})
	}

	if resp.StatusCode != http.StatusOK {
		return "", false, HttpError{
			URL:        targetUrl,
			StatusCode: resp.StatusCode,
			Status:     resp.Status,
		}
	}
	return string(buf), false, nil
}

// GetCacheSize will return the size of the cache in bytes
func (client *CacheClient) GetCacheSize() int {
	return client.cache.GetSize()
}

// Save will save the cache to disk
func (client *CacheClient) Save() error {
	return client.cache.Save()
}
