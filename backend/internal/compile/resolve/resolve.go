package resolve

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

type resolverResult struct {
	exists  bool
	content []byte
}

type Resolver struct {
	FS              fs.FS
	Extensions      []string
	EnableDebugLogs bool
	resolveCache    map[string]resolverResult
}

var Extensions = []string{".js", ".jsx", ".ts", ".tsx", ".json", ".css"}

func (resolver *Resolver) cacheSet(target string, result resolverResult) {
	if resolver.resolveCache == nil {
		resolver.resolveCache = make(map[string]resolverResult, 100)
	}
	resolver.resolveCache[target] = result
}

func (resolver *Resolver) debugf(msg string, args ...interface{}) {
	if resolver.EnableDebugLogs {
		fmt.Fprintf(os.Stderr, "[resolve] "+msg+"\n", args...)
	}
}

func (resolver *Resolver) resolveLocal(target string) (string, error) {
	if resolver.FS == nil {
		return "", fmt.Errorf("resolver has no FS to resolve within")
	}

	target = filepath.Clean(target)

	// Check if the file exists exactly
	if _, ok := resolver.ReadFile(target); ok {
		return target, nil
	}

	searchExtensions := resolver.Extensions
	if searchExtensions == nil {
		searchExtensions = Extensions
	}

	// Otherwise check if it exists with any of the extensions, in priority order
	for _, ext := range searchExtensions {
		if _, ok := resolver.ReadFile(target + ext); ok {
			return target + ext, nil
		}
	}

	// Otherwise check if it exists as a directory, with an index file, with any
	// of the extensions, in priority order
	for _, ext := range searchExtensions {
		targetWithExt := filepath.Clean(filepath.Join(target, "index"+ext))
		if _, ok := resolver.ReadFile(targetWithExt); ok {
			return targetWithExt, nil
		}
	}

	return "", fmt.Errorf("could not resolve: %s", target)
}

func (resolver *Resolver) resolveModule(source, target string) (string, error) {
	searchDir := filepath.Dir(source)

	// TODO: Maybe put a numeric iteration limit here, to prevent infinite loops due
	// to bad code
	for searchDir != "/" {
		nodeModulesDir := filepath.Join(searchDir, "node_modules")
		if result, err := resolver.resolveLocal(filepath.Join(nodeModulesDir, target)); err == nil {
			return result, nil
		}
		searchDir = filepath.Dir(searchDir)
	}

	return "", fmt.Errorf("could not resolve: %s", target)
}

// ReadFile returns the contents of the file at the given path, if the file exists
// at that exact filepath. If the file does not exist, it returns nil and false.
// This is preferable to reading the file directly from the given FS, because during
// module resolution, the cache will be populated with file contents.
func (resolver *Resolver) ReadFile(target string) ([]byte, bool) {
	if res, ok := resolver.resolveCache[target]; ok {
		return res.content, res.exists
	}

	reader, err := resolver.FS.Open(target)
	if err != nil {
		resolver.debugf("miss: %s (%v)", target, err)
		resolver.cacheSet(target, resolverResult{false, nil})
		return nil, false
	}
	defer reader.Close()

	content, err := io.ReadAll(reader)
	if err == nil {
		resolver.debugf("found: %s", target)
	} else {
		resolver.debugf("miss: %s (%v)", target, err)
	}

	resolver.cacheSet(target, resolverResult{err == nil, content})
	return content, err == nil
}

// Resolve searches for a file relative to the root of the filesystem, and fails
// if a module is requested to be resolved.
func (resolver *Resolver) Resolve(target string) (string, error) {
	// If the path does not start with a dot, it's referring to a node module
	if target[0] != '.' {
		return resolver.resolveModule("./index.js", target)
	}
	return resolver.resolveLocal(target)
}

// ResolveFrom searches for a file at the target path, relative to the source filepath.
// Canonically, this means `source` is the file that is trying to import from `target`.
func (resolver *Resolver) ResolveFrom(source, target string) (string, error) {
	if source == "" {
		return "", fmt.Errorf("source path is empty")
	}
	if target == "" {
		return "", fmt.Errorf("target path is empty")
	}

	if target[0] != '.' && target[0] != '/' {
		return resolver.resolveModule(source, target)
	}

	resolver.debugf("resolving from %s: %s", source, target)

	if target[0] == '/' {
		return resolver.resolveLocal(target[1:])
	}

	targetRelPath := "." + string(filepath.Separator) + filepath.Join(filepath.Dir(source), target)
	resolved, err := resolver.Resolve(targetRelPath)
	if err != nil {
		return "", fmt.Errorf("could not resolve: %s", target)
	}
	return resolved, nil
}
