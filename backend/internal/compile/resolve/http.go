package resolve

import (
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path"
	"time"
)

type HttpResolverFs struct {
	BaseURL *url.URL
}

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
