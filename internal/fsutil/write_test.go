package fsutil

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
)

func TestWriteFileAtomic(t *testing.T) {
	tests := []struct {
		name string
		// existingMode is the mode of the existing file, 0 if there is none
		existingMode os.FileMode
		owner        string
		wantMode     os.FileMode
		wantErr      error
	}{
		{name: "new file", wantMode: 0600},
		{name: "keeps mode of existing file", existingMode: 0640, wantMode: 0640},
		{name: "unknown owner", existingMode: 0600, owner: "sc-agent-no-such-user", wantErr: ErrFileOwnership},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			file := filepath.Join(dir, "secret-id")
			if tt.existingMode != 0 {
				if err := os.WriteFile(file, []byte("old"), tt.existingMode); err != nil {
					t.Fatal(err)
				}
				// not affected by the umask
				if err := os.Chmod(file, tt.existingMode); err != nil {
					t.Fatal(err)
				}
			}

			err := WriteFileAtomic(afero.NewOsFs(), file, []byte("new"), tt.owner)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("WriteFileAtomic() error = %v, want %v", err, tt.wantErr)
			}

			want := "new"
			if tt.wantErr != nil {
				want = "old"
			}
			data, err := os.ReadFile(file)
			if err != nil || string(data) != want {
				t.Fatalf("expected file to contain %q, got %q (%v)", want, data, err)
			}
			if tt.wantErr == nil {
				info, err := os.Stat(file)
				if err != nil {
					t.Fatal(err)
				}
				if info.Mode().Perm() != tt.wantMode {
					t.Fatalf("expected mode %v, got %v", tt.wantMode, info.Mode().Perm())
				}
			}
			entries, err := os.ReadDir(dir)
			if err != nil {
				t.Fatal(err)
			}
			if len(entries) != 1 {
				t.Fatalf("expected no temporary files to be left, got %v", entries)
			}
		})
	}
}
