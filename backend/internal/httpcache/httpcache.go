package httpcache

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"robinplatform.dev/internal/log"
)

var logger = log.New("http")

type CacheEntry struct {
	StatusCode    int
	Value         string
	ContentCached bool
	Deadline      *int64
	LastUsed      *int64
}

type httpCache struct {
	filename string
	mux      *sync.Mutex
	maxSize  int

	size   int
	values map[string]*CacheEntry
}

type Cache interface {
	Delete(string)
	Get(string) (CacheEntry, bool)
	Set(string, CacheEntry)
	GetSize() int
	Save() error
}

// New creates a new cache with the given filename. If the filename is non-empty, it will
// attempt to load the cache from disk. If this fails, the cache will still be usable but will
// return with an error.
func New(filename string, maxSize int) (Cache, error) {
	cache := &httpCache{
		filename: filename,
		mux:      &sync.Mutex{},
		maxSize:  maxSize,
		values:   make(map[string]*CacheEntry),
	}
	return cache, cache.open()
}

func (cache *httpCache) open() error {
	cache.mux.Lock()
	defer cache.mux.Unlock()

	if cache.filename == "" {
		return nil
	}

	file, err := os.Open(cache.filename)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to open cache from %s: %w", cache.filename, err)
	}

	buf, err := io.ReadAll(file)
	if err != nil {
		return fmt.Errorf("failed to read cache from %s: %w", cache.filename, err)
	}

	if err := json.Unmarshal(buf, cache); err != nil {
		return fmt.Errorf("failed to unmarshal cache from %s: %w", cache.filename, err)
	}

	logger.Debug("Loaded http cache", log.Ctx{
		"filename":   cache.filename,
		"numEntries": len(cache.values),
		"size":       cache.size,
		"maxSize":    cache.maxSize,
	})

	return nil
}

func (cache *httpCache) Save() error {
	cache.mux.Lock()
	defer cache.mux.Unlock()

	if cache.filename == "" {
		return fmt.Errorf("cannot save cache without a filename")
	}

	cache.compact()

	file, err := os.Create(cache.filename)
	if err != nil {
		return fmt.Errorf("failed to create cache in %s: %w", cache.filename, err)
	}
	defer file.Close()

	buf, err := json.Marshal(cache)
	if err != nil {
		return fmt.Errorf("failed to marshal cache to %s: %w", cache.filename, err)
	}

	if _, err := file.Write(buf); err != nil {
		return fmt.Errorf("failed to write cache to %s: %w", cache.filename, err)
	}

	return nil
}

func (cache *httpCache) GetSize() int {
	cache.mux.Lock()
	defer cache.mux.Unlock()

	return cache.size
}

// delete performs a delete of the given cache entry. This method must be called with a write
// lock on the cache.
func (cache *httpCache) delete(key string) {
	if node, ok := cache.values[key]; ok {
		logger.Debug("Removing from cache", log.Ctx{
			"url":      key,
			"size":     len(node.Value),
			"lastUsed": time.Unix(0, *node.LastUsed).String(),
		})
		cache.size -= len(node.Value)
		delete(cache.values, key)
	}
}

func (cache *httpCache) compact() {
	if cache.size < cache.maxSize {
		return
	}

	cacheStartSize := cache.size
	cacheStartNumEntries := len(cache.values)

	// delete all stale entries
	for key, entry := range cache.values {
		if entry.Deadline != nil && *entry.Deadline < time.Now().UnixNano() {
			cache.delete(key)
		}
	}

	// until we reach target size, delete the last used entry
	for cache.size > cache.maxSize {
		var lastUsedKey string
		var lastUsedEntry *CacheEntry
		for key, node := range cache.values {
			if lastUsedEntry == nil || *node.LastUsed < *lastUsedEntry.LastUsed {
				lastUsedKey = key
				lastUsedEntry = node
			}
		}

		if lastUsedEntry == nil {
			break
		}
		cache.delete(lastUsedKey)
	}

	logger.Debug("HTTP cache compacted", log.Ctx{
		"startSize":       cacheStartSize,
		"startNumEntries": cacheStartNumEntries,
		"endSize":         cache.size,
		"endNumEntries":   len(cache.values),
	})
}

func (cache *httpCache) Delete(key string) {
	cache.mux.Lock()
	defer cache.mux.Unlock()

	cache.delete(key)
}

func (cache *httpCache) Get(key string) (CacheEntry, bool) {
	cache.mux.Lock()
	defer cache.mux.Unlock()

	node, ok := cache.values[key]
	if ok {
		// If the entry has expired, delete and pretend it wasn't found
		if node.Deadline != nil && *node.Deadline < time.Now().UnixNano() {
			cache.delete(key)
			return CacheEntry{}, false
		}

		atomic.StoreInt64(node.LastUsed, time.Now().UnixNano())
		return *node, true
	}

	return CacheEntry{}, false
}

func (cache *httpCache) Set(key string, entry CacheEntry) {
	cache.mux.Lock()
	defer cache.mux.Unlock()

	// do not allow single resources that are larger than the cache
	if len(entry.Value) >= cache.maxSize {
		logger.Debug("Refusing to cache large resource", log.Ctx{
			"url":  key,
			"size": len(entry.Value),
			"max":  cache.maxSize,
		})
		return
	}

	cache.delete(key)

	lastUsed := int64(time.Now().UnixNano())
	entry.LastUsed = &lastUsed

	cache.values[key] = &entry
	cache.size += len(entry.Value)

	logger.Debug("Added to cache", log.Ctx{
		"url":              key,
		"size":             len(entry.Value),
		"updatedCacheSize": cache.size,
	})
	cache.compact()
}
