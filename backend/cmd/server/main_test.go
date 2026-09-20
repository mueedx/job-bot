package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// setupRepo builds a minimal repo layout (root with a Makefile and data/, plus a
// backend/ subdirectory) and returns the two directories.
func setupRepo(t *testing.T) (root, backend string) {
	t.Helper()
	root = t.TempDir()
	backend = filepath.Join(root, "backend")
	for _, dir := range []string{filepath.Join(root, "data"), backend} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "Makefile"), []byte("help:\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return root, backend
}

// Running from backend/ with DATA_DIR=./data used to create a second, empty
// database in backend/data while the dashboard read the real one. The relative
// path does not exist there, so it must fall back to the repo's data/ directory.
func TestResolveDataDirFallsBackForMissingRelativePath(t *testing.T) {
	root, backend := setupRepo(t)
	t.Chdir(backend)
	t.Setenv("DATA_DIR", "./data")

	got := resolveDataDir()
	if got != filepath.Join(root, "data") {
		t.Fatalf("resolveDataDir() = %q, want %q", got, filepath.Join(root, "data"))
	}
}

func TestResolveDataDirPrefersRepoRootFromSubdirectory(t *testing.T) {
	root, backend := setupRepo(t)
	t.Chdir(backend)
	t.Setenv("DATA_DIR", "")

	got := resolveDataDir()
	if got != filepath.Join(root, "data") {
		t.Fatalf("resolveDataDir() = %q, want %q", got, filepath.Join(root, "data"))
	}
}

func TestResolveDataDirHonoursExistingDirectory(t *testing.T) {
	_, backend := setupRepo(t)
	custom := filepath.Join(t.TempDir(), "custom-data")
	if err := os.MkdirAll(custom, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(backend)
	t.Setenv("DATA_DIR", custom)

	if got := resolveDataDir(); got != custom {
		t.Fatalf("resolveDataDir() = %q, want the configured %q", got, custom)
	}
}

// An absolute path is an explicit choice (Docker sets DATA_DIR=/data), so it is
// honoured even before the directory exists.
func TestResolveDataDirHonoursAbsolutePathThatDoesNotExistYet(t *testing.T) {
	_, backend := setupRepo(t)
	abs := filepath.Join(t.TempDir(), "not-created-yet")
	t.Chdir(backend)
	t.Setenv("DATA_DIR", abs)

	if got := resolveDataDir(); got != abs {
		t.Fatalf("resolveDataDir() = %q, want the configured %q", got, abs)
	}
}

// Outside any repo (no Makefile above us) the old behaviour must still apply.
func TestResolveDataDirOutsideRepo(t *testing.T) {
	isolated := t.TempDir()
	t.Chdir(isolated)
	t.Setenv("DATA_DIR", "")

	got := resolveDataDir()
	if got != filepath.Join("..", "data") {
		t.Fatalf("resolveDataDir() = %q, want the ../data fallback", got)
	}
	if strings.Contains(got, isolated) {
		t.Fatalf("expected a relative fallback, got %q", got)
	}
}
