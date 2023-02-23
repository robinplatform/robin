package resolve

import (
	"fmt"
	"io"
	"io/fs"
	"path"
)

type resolverResult struct {
	exists  bool
	content []byte
}

type Resolver struct {
	FS           fs.FS
	Extensions   []string
	resolveCache map[string]resolverResult
}

var Extensions = []string{".js", ".jsx", ".ts", ".tsx", ".json", ".css"}

func (resolver *Resolver) ReadFile(target string) ([]byte, bool) {
	if res, ok := resolver.resolveCache[target]; ok {
		return res.content, res.exists
	}

	reader, err := resolver.FS.Open(target)
	if err != nil {
		resolver.resolveCache[target] = resolverResult{false, nil}
		return nil, false
	}
	defer reader.Close()

	content, err := io.ReadAll(reader)
	resolver.resolveCache[target] = resolverResult{
		exists:  err == nil,
		content: content,
	}
	return content, err == nil
}

func (resolver *Resolver) Resolve(target string) (string, error) {
	if resolver.FS == nil {
		return "", fmt.Errorf("resolver must be given a filesystem to search within")
	}
	if resolver.Extensions == nil {
		resolver.Extensions = Extensions
	}
	if resolver.resolveCache == nil {
		resolver.resolveCache = make(map[string]resolverResult, 10)
	}

	target = path.Clean(target)

	// Check if the file exists exactly
	if _, ok := resolver.ReadFile(target); ok {
		return target, nil
	}

	// Otherwise check if it exists with any of the extensions, in priority order
	for _, ext := range resolver.Extensions {
		if _, ok := resolver.ReadFile(target + ext); ok {
			return target + ext, nil
		}
	}

	// Otherwise check if it exists as a directory, with an index file, with any
	// of the extensions, in priority order
	for _, ext := range resolver.Extensions {
		targetWithExt := path.Clean(path.Join(target, "index"+ext))
		if _, ok := resolver.ReadFile(targetWithExt); ok {
			return targetWithExt, nil
		}
	}

	return "", fmt.Errorf("could not resolve: %s", target)
}

func (resolver *Resolver) ResolveFrom(source, target string) (string, error) {
	sourceDir := path.Dir(source)
	return resolver.Resolve(path.Join(sourceDir, target))
}
