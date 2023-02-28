package resolve

import (
	"path/filepath"
	"testing"
	"testing/fstest"
)

type resolverTester struct {
	t        *testing.T
	fs       map[string]*fstest.MapFile
	resolver *Resolver
}

func createTester(t *testing.T, inputVfs map[string]*fstest.MapFile) resolverTester {
	vfs := make(map[string]*fstest.MapFile, len(inputVfs))

	for key, value := range inputVfs {
		vfs[filepath.FromSlash(key)] = value
	}

	return resolverTester{
		t:        t,
		fs:       vfs,
		resolver: &Resolver{FS: fstest.MapFS(vfs)},
	}
}

func (test *resolverTester) assertNotExists(target string) {
	if resolvedTarget, err := test.resolver.Resolve(target); err == nil {
		test.t.Fatalf("expected error when resolving '%s', got '%s'", target, resolvedTarget)
	}
}

func (test *resolverTester) assertResolved(target, expected string) {
	target = filepath.FromSlash(target)
	expected = filepath.FromSlash(expected)

	if resolvedTarget, err := test.resolver.Resolve(target); err != nil {
		test.t.Fatalf("expected no error when resolving '%s', got '%s'", target, err)
	} else if resolvedTarget != expected {
		test.t.Fatalf("expected '%s', got '%s'", expected, resolvedTarget)
	}
}

func (test *resolverTester) assertResolvedFrom(source, target, expected string) {
	source = filepath.FromSlash(source)
	target = filepath.FromSlash(target)
	expected = filepath.FromSlash(expected)

	if resolvedTarget, err := test.resolver.ResolveFrom(source, target); err != nil {
		test.t.Fatalf("expected no error when resolving '%s' from '%s', got '%s'", target, source, err)
	} else if resolvedTarget != expected {
		test.t.Fatalf("failed to resolve from %s: expected '%s', got '%s'", source, expected, resolvedTarget)
	}
}

func TestFileResolvers(t *testing.T) {
	tester := createTester(t, map[string]*fstest.MapFile{
		"index.js":      {},
		"bar.js":        {},
		"src/index.js":  {},
		"src/foo.js":    {},
		"src/data.json": {},
	})

	tester.assertNotExists("./foo")
	tester.assertNotExists("../")
	tester.assertResolved("./bar", "bar.js")
	tester.assertResolved("./src/foo", "src/foo.js")
	tester.assertResolved("./src/data", "src/data.json")

	// Resolve relative paths
	tester.assertResolvedFrom("./src/foo.js", "./foo", "src/foo.js")
	tester.assertResolvedFrom("./src/index.js", "./data", "src/data.json")
	tester.assertResolvedFrom("./src/index.js", "../bar", "bar.js")

	// Should stop resolving json if we remove the extension
	{
		tester := createTester(t, map[string]*fstest.MapFile{
			"src/data.json": {},
		})
		tester.resolver.Extensions = []string{".js"}
		tester.assertNotExists("./src/data")
	}
}

func TestDirResolvers(t *testing.T) {
	tester := createTester(t, map[string]*fstest.MapFile{
		"index.js":          {},
		"bar.js":            {},
		"src/index.js":      {},
		"src/foo.js":        {},
		"src/bar/index.ts":  {},
		"src/json/foo.json": {},
	})

	tester.assertResolved("./", "index.js")
	tester.assertResolved("./src", "src/index.js")
	tester.assertResolved("./src/bar", "src/bar/index.ts")
	tester.assertResolved("./src/json/foo", "src/json/foo.json")
	tester.assertNotExists("./src/json")

	// Resolve relative paths
	tester.assertResolvedFrom("./src/foo.js", "./", "src/index.js")
	tester.assertResolvedFrom("./src/index.js", "./bar", "src/bar/index.ts")
	tester.assertResolvedFrom("./src/index.js", "../bar", "bar.js")

	// Should stop resolving json if we remove the extension
	{
		tester := createTester(t, map[string]*fstest.MapFile{
			"src/index.json": {},
		})
		tester.resolver.Extensions = []string{".js"}
		tester.assertNotExists("./src")
	}
}

func TestModuleFileResolvers(t *testing.T) {
	tester := createTester(t, map[string]*fstest.MapFile{
		"src/index.js":              {},
		"src/foo.js":                {},
		"src/data.json":             {},
		"node_modules/bar/index.js": {},
		"node_modules/bar/foo.js":   {},
		"node_modules/bar/node_modules/lodash/index.js": {},
	})

	tester.assertNotExists("./src/bar")
	tester.assertResolvedFrom("./src/foo.js", "bar", "node_modules/bar/index.js")
	tester.assertResolvedFrom("./node_modules/bar/index.js", "./foo", "node_modules/bar/foo.js")
	tester.assertResolvedFrom("./node_modules/bar/index.js", "lodash", "node_modules/bar/node_modules/lodash/index.js")
}
