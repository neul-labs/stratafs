package monitor

import (
	"path/filepath"
	"testing"
)

func TestSanitizeRemotePath(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		want      string
		shouldErr bool
	}{
		{name: "simple relative", input: "docs/file.txt", want: filepath.Join("docs", "file.txt")},
		{name: "leading slash", input: "/nested/path/file.txt", want: filepath.Join("nested", "path", "file.txt")},
		{name: "collapse parent", input: "a/../b/file.txt", want: filepath.Join("b", "file.txt")},
		{name: "root only", input: "/", shouldErr: true},
		{name: "parent escape", input: "../etc/passwd", shouldErr: true},
		{name: "double parent escape", input: "../../etc/passwd", shouldErr: true},
		{name: "empty", input: "", shouldErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := sanitizeRemotePath(tt.input)
			if tt.shouldErr {
				if err == nil {
					t.Fatalf("expected error for input %q", tt.input)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}
