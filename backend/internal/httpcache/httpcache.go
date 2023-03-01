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

type cacheEntry struct {
	Value    string
	Deadline *time.Time
	LastUsed *int64
}

type httpCache struct {
	filename string
	mux      *sync.RWMutex
	maxSize  int

	Size   int
	Values map[string]*cacheEntry
}

type Cache interface {
	Delete(string)
	Get(string) (string, bool)
	Set(string, string, *time.Duration)
	GetSize() int
	Save() error
}

func New(filename string, maxSize int) (Cache, error) {
	cache := &httpCache{
		filename: filename,
		mux:      &sync.RWMutex{},
		maxSize:  maxSize,
		Values:   make(map[string]*cacheEntry),
	}
	fmt.Printf("cache max size: %d\n", maxSize)
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
		"numEntries": len(cache.Values),
		"size":       cache.Size,
		"maxSize":    cache.maxSize,
	})

	return nil
}

func (cache *httpCache) Save() error {
	if cache.filename == "" {
		return fmt.Errorf("cannot save cache without a filename")
	}

	cache.mux.RLock()
	defer cache.mux.RUnlock()

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
	return cache.Size
}

// delete performs a delete of the given cache entry. This method must be called with a write
// lock on the cache.
func (cache *httpCache) delete(key string) {
	if node, ok := cache.Values[key]; ok {
		logger.Debug("Removing from cache", log.Ctx{
			"url":      key,
			"size":     len(node.Value),
			"lastUsed": time.Unix(0, *node.LastUsed).String(),
		})
		cache.Size -= len(node.Value)
		delete(cache.Values, key)
	}
}

func (cache *httpCache) Delete(key string) {
	cache.mux.Lock()
	cache.delete(key)
	cache.mux.Unlock()
}

func (cache *httpCache) Get(key string) (string, bool) {
	cache.mux.RLock()
	node, ok := cache.Values[key]
	cache.mux.RUnlock()

	if ok {
		// If the entry has expired, delete and pretend it wasn't found
		if node.Deadline != nil && node.Deadline.Before(time.Now()) {
			cache.Delete(key)
			return "", false
		}

		atomic.StoreInt64(node.LastUsed, time.Now().UnixNano())
		return node.Value, true
	}

	return "", false
}

func (cache *httpCache) Set(key, value string, ttl *time.Duration) {
	// do not allow single resources that are larger than the cache
	if len(value) >= cache.maxSize {
		logger.Debug("Refusing to cache large resource", log.Ctx{
			"url":  key,
			"size": len(value),
			"max":  cache.maxSize,
		})
		return
	}

	cache.mux.Lock()
	defer cache.mux.Unlock()

	cache.delete(key)

	lastUsed := int64(time.Now().UnixNano())
	node := &cacheEntry{
		Value:    value,
		LastUsed: &lastUsed,
	}
	if ttl != nil {
		deadline := time.Now().Add(*ttl)
		node.Deadline = &deadline
	}

	cache.Values[key] = node
	cache.Size += len(value)

	logger.Debug("Added to cache", log.Ctx{
		"url":              key,
		"size":             len(value),
		"updatedCacheSize": cache.Size,
	})

	for cache.Size > cache.maxSize && len(cache.Values) > 1 {
		var lastUsedKey string
		var lastUsedEntry *cacheEntry
		for key, node := range cache.Values {
			if lastUsedEntry == nil || *node.LastUsed < *lastUsedEntry.LastUsed {
				lastUsedKey = key
				lastUsedEntry = node
			}
		}

		cache.delete(lastUsedKey)
	}
}
