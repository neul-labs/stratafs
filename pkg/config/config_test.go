package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultConfigIncludesEnvironmentDirectories(t *testing.T) {
	globalDir := t.TempDir()
	t.Setenv("STRATAFS_GLOBAL_DIR", globalDir)

	additionalRoot := t.TempDir()
	dirOne := filepath.Join(additionalRoot, "dir-one")
	dirTwo := filepath.Join(additionalRoot, "dir-two")
	if err := os.MkdirAll(dirOne, 0o755); err != nil {
		t.Fatalf("failed to create dirOne: %v", err)
	}
	if err := os.MkdirAll(dirTwo, 0o755); err != nil {
		t.Fatalf("failed to create dirTwo: %v", err)
	}

	t.Setenv("STRATAFS_DIRS", strings.Join([]string{dirOne, dirTwo}, ","))

	cfg := DefaultConfig()

	if len(cfg.Sources) != 3 {
		t.Fatalf("expected 3 sources (default + env), got %d", len(cfg.Sources))
	}

	paths := make(map[string]struct{})
	for _, source := range cfg.Sources {
		paths[source.Path] = struct{}{}
	}

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}

	for _, expected := range []string{wd, dirOne, dirTwo} {
		if _, ok := paths[expected]; !ok {
			t.Fatalf("expected path %s to be present in sources", expected)
		}
	}
}
