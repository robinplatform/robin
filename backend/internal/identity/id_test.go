package identity

import "testing"

func CategoryFuncTester(t *testing.T, expected string, fields ...string) {
	o := Category(fields...)

	if o != expected {
		t.Fatalf("failed to sanitize bad input, got: %v, expected: %v", o, expected)
	}
}

func TestCategoryFunc(t *testing.T) {
	CategoryFuncTester(t, "/logs/apps/hello/world", "logs", "apps", "hello", "world")
	CategoryFuncTester(t, "/logs/%2E%2E/apps/hello%2Fworld", "logs", "..", "apps", "hello/world")
}
