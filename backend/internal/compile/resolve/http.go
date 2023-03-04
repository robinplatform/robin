package resolve

import (
	"fmt"
	"io"
	"io/fs"
	"net/url"
	"path"
	"strings"
	"time"

	"robinplatform.dev/internal/httpcache"
)

// HttpResolverFs is an implementation of fs.FS that resolves files from a CDN.
type HttpResolverFs struct {
	BaseURL *url.URL
	client  httpcache.CacheClient
}

// NewHttpResolver creates a new resolver that is backed by a CDN. Only the scheme, host, and
// user information are used from the given URL. The path is ignored.
func NewHttpResolver(givenUrl *url.URL, client httpcache.CacheClient) *Resolver {
	baseUrl := &url.URL{
		Scheme: givenUrl.Scheme,
		Host:   givenUrl.Host,
		User:   givenUrl.User,
	}
	return &Resolver{
		FS: &HttpResolverFs{
			BaseURL: baseUrl,
			client:  client,
		},
	}
}

// HttpFileEntry is an implementation of fs.File that is backed by an HTTP response.
type HttpFileEntry struct {
	path     string
	contents string
}

func (entry HttpFileEntry) Name() string {
	return path.Base(entry.path)
}

func (entry HttpFileEntry) Size() int64 {
	panic(fmt.Errorf("unsupported method"))
}

func (entry HttpFileEntry) Mode() fs.FileMode {
	panic(fmt.Errorf("unsupported method"))
}

func (entry HttpFileEntry) ModTime() time.Time {
	panic(fmt.Errorf("unsupported method"))
}

func (entry HttpFileEntry) IsDir() bool {
	panic(fmt.Errorf("unsupported method"))
}

func (entry HttpFileEntry) Sys() interface{} {
	panic(fmt.Errorf("unsupported method"))
}

func (entry HttpFileEntry) Stat() (fs.FileInfo, error) {
	return entry, nil
}

func (entry HttpFileEntry) Read(p []byte) (n int, err error) {
	n = copy(p, entry.contents)
	return n, io.EOF
}

func (entry HttpFileEntry) Close() error {
	return nil
}

func isNodeBuiltinPath(path string) bool {
	path = path[1:]
	if path[0] != 'v' {
		return false
	}

	// skip all numbers next
	for i := 1; i < len(path); i++ {
		if path[i] < '0' || path[i] > '9' {
			path = path[i:]
			break
		}
	}

	// next should be a slash
	if path[0] != '/' {
		return false
	}
	path = path[1:]

	return strings.HasPrefix(path, "node_") && !strings.ContainsRune(path, '/')
}

func (hfs *HttpResolverFs) Open(filename string) (fs.File, error) {
	fileUrl := hfs.BaseURL.ResolveReference(&url.URL{Path: filename})

	// This is a bit silly, but esm.sh is really slow to detect bad URLs, but unpkg.com is very
	// fast thanks to edge caching. So we check if the file exists on unpkg.com first, and if it
	// does, we allow the request to go through.
	//
	// unpkg.com also realizes that the URL is actually immutable, and will ask the client to cache
	// it while esm.sh reports the response as 'no-cache'.
	//
	// However, we will skip node builtin polyfills, which are hosted on `esm.sh`, but don't exist
	// on `unpkg.com`.
	if fileUrl.Scheme == "https" && fileUrl.Host == "esm.sh" && !isNodeBuiltinPath(fileUrl.Path) {
		_, err := hfs.client.Head(fmt.Sprintf("https://unpkg.com%s", fileUrl.Path))
		if err != nil {
			return nil, err
		}
	}

	res, err := hfs.client.Get(fileUrl.String())

	// TODO: return a fs.ErrNotExist if the status code is 404
	if err != nil {
		return nil, err
	}

	return HttpFileEntry{
		path:     filename,
		contents: res.Body,
	}, nil
}
