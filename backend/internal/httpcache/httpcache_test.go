package httpcache

import (
	"crypto/rand"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func ptr[T any](d T) *T {
	return &d
}

func assertCacheControl(t *testing.T, value string, expectedDeadline *time.Duration, expectedShouldCache bool) {
	deadline, shouldCache := parseCacheControl(value)

	expectedDeadlineStr := ""
	if expectedDeadline != nil {
		expectedDeadlineStr = expectedDeadline.Truncate(time.Second).String()
	}

	deadlineStr := ""
	if deadline != nil {
		deadlineStr = deadline.Truncate(time.Second).String()
	}

	if deadlineStr != expectedDeadlineStr || shouldCache != expectedShouldCache {
		t.Fatalf("unexpected parsed cache control: %s, %v (expected %s, %v)", deadlineStr, shouldCache, expectedDeadlineStr, expectedShouldCache)
	}
}

func TestParseCacheControl(t *testing.T) {
	assertCacheControl(t, "", nil, false)
	assertCacheControl(t, "no-cache", nil, false)
	assertCacheControl(t, "max-age=0", nil, false)
	assertCacheControl(t, "max-age=1000", ptr(1000*time.Second), true)
	assertCacheControl(t, "max-age=1000, must-revalidate", ptr(1000*time.Second), true)
}

func TestHttpCache(t *testing.T) {
	server := http.Server{}
	server.Addr = ":0"
	server.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/immutable":
			w.Header().Set("Cache-Control", "immutable")
		case "/1s":
			w.Header().Set("Cache-Control", "max-age=2")
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Hello, world!"))
	})

	listener, err := net.Listen("tcp", server.Addr)
	if err != nil {
		t.Fatal(err)
	}
	go server.Serve(listener)

	client, err := NewClient("", 10*1000)
	if err != nil {
		t.Fatal(err)
	}

	{
		res, err := client.Get(fmt.Sprintf("http://%s/1s", listener.Addr().String()))
		if err != nil {
			t.Fatal(err)
		}

		if res.Body != "Hello, world!" {
			t.Fatalf("unexpected response: %s", res.Body)
		}
		if res.FromCache {
			t.Fatalf("unexpected cache hit")
		}

		// re-fetch from cache immediately
		res, err = client.Get(fmt.Sprintf("http://%s/1s", listener.Addr().String()))
		if err != nil {
			t.Fatal(err)
		}

		if res.Body != "Hello, world!" {
			t.Fatalf("unexpected response: %s", res.Body)
		}
		if !res.FromCache {
			t.Logf("%#v\n", client.cache)
			t.Fatalf("unexpected cache miss")
		}

		// wait for cache to expire
		time.Sleep(3 * time.Second)

		// re-fetch from server
		res, err = client.Get(fmt.Sprintf("http://%s/1s", listener.Addr().String()))
		if err != nil {
			t.Fatal(err)
		}

		if res.Body != "Hello, world!" {
			t.Fatalf("unexpected response: %s", res.Body)
		}
		if res.FromCache {
			t.Fatalf("unexpected cache hit")
		}
	}

	{
		// load immutable resource, should get cached
		res, err := client.Get(fmt.Sprintf("http://%s/immutable", listener.Addr().String()))
		if err != nil {
			t.Fatal(err)
		}

		if res.Body != "Hello, world!" {
			t.Fatalf("unexpected response: %s", res.Body)
		}
		if res.FromCache {
			t.Fatalf("unexpected cache hit")
		}

		// close the server, so we can test that the cache is still valid
		server.Close()
		listener.Close()

		// re-fetch from cache
		res, err = client.Get(fmt.Sprintf("http://%s/immutable", listener.Addr().String()))
		if err != nil {
			t.Fatal(err)
		}

		if res.Body != "Hello, world!" {
			t.Fatalf("unexpected response: %s", res.Body)
		}
		if !res.FromCache {
			t.Fatalf("unexpected cache miss")
		}
	}

	// HEAD request should succeed, even though the server is not running
	{
		exists, err := client.Head(fmt.Sprintf("http://%s/immutable", listener.Addr().String()))
		if err != nil {
			t.Fatal(err)
		}

		if !exists {
			t.Fatalf("unexpected HEAD response (expected resouce to exist)")
		}
	}
}

func TestHttpCacheMaxSize(t *testing.T) {
	// identify temporary location to store cache
	tmpDir := os.TempDir()
	tmpHash := make([]byte, 8)
	rand.Read(tmpHash)
	tmpCacheFile := filepath.Join(tmpDir, fmt.Sprintf("httpcache_test_%x", tmpHash))
	defer os.Remove(tmpCacheFile)

	// start server to test against
	server := http.Server{}
	server.Addr = ":0"
	server.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "immutable")
		w.WriteHeader(http.StatusOK)

		// we will always return 1000 byte responses
		w.Write([]byte(strings.Repeat("a", 1000)))
	})

	listener, err := net.Listen("tcp", server.Addr)
	if err != nil {
		t.Fatal(err)
	}
	go server.Serve(listener)

	// this client can store up to 10 responses
	client, err := NewClient(tmpCacheFile, 10*1000)
	if err != nil {
		t.Fatal(err)
	}

	// verify that cache file does not exist
	if _, err := os.Stat(tmpCacheFile); err == nil {
		t.Fatalf("cache file should not exist")
	} else if !os.IsNotExist(err) {
		t.Fatal(err)
	}

	// make 10 requests, which should all be a cache miss
	for i := 0; i < 10; i++ {
		tmpUrl := fmt.Sprintf("http://%s/%d", listener.Addr().String(), i)
		res, err := client.Get(tmpUrl)
		if err != nil {
			t.Fatal(err)
		}

		if len(res.Body) != 1000 {
			t.Fatalf("unexpected response: %s", res.Body)
		}
		if res.FromCache {
			t.Fatalf("unexpected cache hit: %s", tmpUrl)
		}
	}

	// save cache, and reopen
	if err := client.Save(); err != nil {
		t.Fatal(err)
	}

	client, err = NewClient(tmpCacheFile, 10*1000)
	if err != nil {
		t.Fatal(err)
	}

	// re-fetch all 10, should all hit the cache
	for i := 0; i < 10; i++ {
		tmpUrl := fmt.Sprintf("http://%s/%d", listener.Addr().String(), i)
		res, err := client.Get(tmpUrl)
		if err != nil {
			t.Fatal(err)
		}

		if len(res.Body) != 1000 {
			t.Fatalf("unexpected response: %s", res.Body)
		}
		if !res.FromCache {
			t.Fatalf("unexpected cache miss: %s", tmpUrl)
		}

		// small delay to make sure that the order of the cache is preserved
		time.Sleep(10 * time.Millisecond)
	}

	// make one more request, which should immediately cause the first request
	// to get bumped out of the cache
	{
		res, err := client.Get(fmt.Sprintf("http://%s/10", listener.Addr().String()))
		if err != nil {
			t.Fatal(err)
		}

		if len(res.Body) != 1000 {
			t.Fatalf("unexpected response: %s", res.Body)
		}
		if res.FromCache {
			t.Fatalf("unexpected cache hit")
		}
	}

	// check first request again
	{
		res, err := client.Get(fmt.Sprintf("http://%s/0", listener.Addr().String()))
		if err != nil {
			t.Fatal(err)
		}

		if len(res.Body) != 1000 {
			t.Fatalf("unexpected response: %s", res.Body)
		}

		if res.FromCache {
			t.Fatalf("unexpected cache hit")
		}
	}

	// save cache, and reopen
	if err := client.Save(); err != nil {
		t.Fatal(err)
	}

	client, err = NewClient(tmpCacheFile, 10*1000)
	if err != nil {
		t.Fatal(err)
	}

	// make sure that the cache size is adjusted
	if cacheSize := client.GetCacheSize(); cacheSize > 10*1000 {
		t.Fatalf("unexpected cache size: %d", cacheSize)
	}

	// read cache by hand, for debugging
	{
		fd, err := os.Open(tmpCacheFile)
		if err != nil {
			t.Fatal(err)
		}

		buf, err := io.ReadAll(fd)
		if err != nil {
			t.Fatal(err)
		}

		t.Logf("Persisted cache file: %s\n", tmpCacheFile)
		t.Logf("Persisted cache size: %d\n", len(buf))
	}
}

func TestHttpCacheWithPersistedDeadline(t *testing.T) {
	tmpDir := os.TempDir()
	tmpHash := make([]byte, 8)
	rand.Read(tmpHash)
	tmpCacheFile := filepath.Join(tmpDir, fmt.Sprintf("httpcache_test_%x", tmpHash))
	defer os.Remove(tmpCacheFile)

	// start server to test against
	server := http.Server{}
	server.Addr = ":0"
	server.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "max-age=2")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Hello, world!"))
	})

	listener, err := net.Listen("tcp", server.Addr)
	if err != nil {
		t.Fatal(err)
	}
	go server.Serve(listener)

	// create a client that can store up to 10 responses
	client, err := NewClient(tmpCacheFile, 10*1000)
	if err != nil {
		t.Fatal(err)
	}

	// run a single request to cache it
	{
		res, err := client.Get(fmt.Sprintf("http://%s/", listener.Addr().String()))
		if err != nil {
			t.Fatal(err)
		}

		if res.Body != "Hello, world!" {
			t.Fatalf("unexpected response: %s", res.Body)
		}

		if res.FromCache {
			t.Fatalf("unexpected cache hit")
		}
	}

	//  make sure the value is in the cache
	{
		res, err := client.Get(fmt.Sprintf("http://%s/", listener.Addr().String()))
		if err != nil {
			t.Fatal(err)
		}

		if res.Body != "Hello, world!" {
			t.Fatalf("unexpected response: %s", res.Body)
		}
		if !res.FromCache {
			t.Fatalf("unexpected cache miss")
		}
	}

	// save the cache
	if err := client.Save(); err != nil {
		t.Fatal(err)
	}

	// wait for the cache to expire
	time.Sleep(3 * time.Second)

	// reload the cache
	client, err = NewClient(tmpCacheFile, 10*1000)
	if err != nil {
		t.Fatal(err)
	}

	// re-run the request, which should be a cache miss
	{
		res, err := client.Get(fmt.Sprintf("http://%s/", listener.Addr().String()))
		if err != nil {
			t.Fatal(err)
		}

		if res.Body != "Hello, world!" {
			t.Fatalf("unexpected response: %s", res.Body)
		}
		if res.FromCache {
			t.Fatalf("unexpected cache hit")
		}
	}
}

func BenchmarkClientColdGet(b *testing.B) {
	server := http.Server{}
	server.Addr = ":0"
	server.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "immutable")
		w.WriteHeader(http.StatusOK)

		// we will always return 1000 byte responses
		w.Write([]byte(strings.Repeat("a", 1000)))
	})

	listener, err := net.Listen("tcp", server.Addr)
	if err != nil {
		b.Fatal(err)
	}
	go server.Serve(listener)

	client, err := NewClient("", 2000)
	if err != nil {
		b.Fatal(err)
	}

	targetUrl := fmt.Sprintf("http://%s/", listener.Addr().String())

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		res, err := client.Get(targetUrl + strconv.FormatInt(int64(i), 10))
		if err != nil {
			b.Fatal(err)
		}

		if len(res.Body) != 1000 {
			b.Fatalf("unexpected response: %s", res.Body)
		}
		if res.FromCache {
			b.Fatalf("unexpected cache hit")
		}
	}
}

func BenchmarkClientWarmGet(b *testing.B) {
	server := http.Server{}
	server.Addr = ":0"
	server.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "immutable")
		w.WriteHeader(http.StatusOK)

		// we will always return 1000 byte responses
		w.Write([]byte(strings.Repeat("a", 1000)))
	})

	listener, err := net.Listen("tcp", server.Addr)
	if err != nil {
		b.Fatal(err)
	}
	go server.Serve(listener)

	// this client can store up to 10 responses
	client, err := NewClient("", 2000)
	if err != nil {
		b.Fatal(err)
	}

	targetUrl := fmt.Sprintf("http://%s/test", listener.Addr().String())

	{
		res, err := client.Get(targetUrl)
		if err != nil {
			b.Fatal(err)
		}

		if len(res.Body) != 1000 {
			b.Fatalf("unexpected response: %s", res.Body)
		}
		if res.FromCache {
			b.Fatalf("unexpected cache hit")
		}
	}

	server.Close()
	listener.Close()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		res, err := client.Get(targetUrl)
		if err != nil {
			b.Fatal(err)
		}

		if len(res.Body) != 1000 {
			b.Fatalf("unexpected response: %s", res.Body)
		}
		if !res.FromCache {
			b.Fatalf("unexpected cache miss")
		}
	}
}
