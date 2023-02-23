package resolve

import (
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path"
	"time"
)

// HttpResolverFs is an implementation of fs.FS that resolves files from a CDN. Right now, we
// only use this to load files from unpkg.com.
//
// This design makes two assumptions:
//  1. The remote CDN is setup so that all files are visible and layed out in the same way
//     as a local file system. This is only true for unpkg.com for now.
//  2. The cost of a HEAD+GET request is more expensive than a GET request that returns a 404.
//     With this assumption, the resolver struct only has an "Open" (no stat) method, even though
//     on a local filesystem, you would usually use a statcache to perform resolution.
type HttpResolverFs struct {
	BaseURL *url.URL
}

// NewHttpResolver creates a new resolver that is backed by a CDN. Only the scheme, host, and
// user information are used from the given URL. The path is ignored.
func NewHttpResolver(givenUrl *url.URL) *Resolver {
	baseUrl := &url.URL{
		Scheme: givenUrl.Scheme,
		Host:   givenUrl.Host,
		User:   givenUrl.User,
	}
	return &Resolver{
		FS: &HttpResolverFs{
			BaseURL: baseUrl,
		},
	}
}

// HttpFileEntry is an implementation of fs.File that is backed by an HTTP response.
// This lines up with assumption (2) above.
type HttpFileEntry struct {
	res *http.Response
}

func (entry HttpFileEntry) Name() string {
	return path.Base(entry.res.Request.URL.Path)
}

func (entry HttpFileEntry) Size() int64 {
	return entry.res.ContentLength
}

func (entry HttpFileEntry) Mode() fs.FileMode {
	return 0
}

func (entry HttpFileEntry) ModTime() time.Time {
	return time.Now()
}

func (entry HttpFileEntry) IsDir() bool {
	return false
}

func (entry HttpFileEntry) Sys() interface{} {
	return nil
}

func (entry HttpFileEntry) Stat() (fs.FileInfo, error) {
	return entry, nil
}

func (entry HttpFileEntry) Read(p []byte) (n int, err error) {
	return entry.res.Body.Read(p)
}

func (entry HttpFileEntry) Close() error {
	return entry.res.Body.Close()
}

func (hfs *HttpResolverFs) Open(filename string) (fs.File, error) {
	fileUrl := hfs.BaseURL.ResolveReference(&url.URL{Path: filename})
	resp, err := http.Get(fileUrl.String())
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, &fs.PathError{
			Op:   "open",
			Path: filename,
			Err:  os.ErrNotExist,
		}
	}
	return HttpFileEntry{resp}, nil
}
