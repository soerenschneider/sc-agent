package storage

import (
	"fmt"
	"os"
	"reflect"
	"testing"
)

func TestNewFilesystemStorageFromUri(t *testing.T) {
	tests := []struct {
		name    string
		uri     string
		want    *FilesystemStorage
		wantErr bool
	}{
		{
			name: "Simple",
			uri:  "file:///home/soeren/.certs/cert.pem",
			want: &FilesystemStorage{
				FilePath:  "/home/soeren/.certs/cert.pem",
				FileOwner: "root",
				FileGroup: "root",
				Mode:      defaultFileMode,
			},
			wantErr: false,
		},
		{
			name: "Permission",
			uri:  "file:///home/soeren/.certs/cert.pem?chmod=755",
			want: &FilesystemStorage{
				FilePath:  "/home/soeren/.certs/cert.pem",
				FileOwner: "root",
				FileGroup: "root",
				Mode:      os.FileMode(0755),
			},
			wantErr: false,
		},
		{
			name: "With user and group",
			uri:  fmt.Sprintf("file://root:%s@/home/soeren/.certs/cert.pem", getOsDependentGroup()),
			want: &FilesystemStorage{
				FilePath:  "/home/soeren/.certs/cert.pem",
				FileOwner: "root",
				FileGroup: getOsDependentGroup(),
				Mode:      defaultFileMode,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewFilesystemStorageFromUri(tt.uri)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewFilesystemStorageFromUri() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !compareFilesystemStorage(got, tt.want) {
				t.Errorf("NewFilesystemStorageFromUri() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func compareFilesystemStorage(a, b *FilesystemStorage) bool {
	a.fs = nil
	b.fs = nil
	return reflect.DeepEqual(a, b)
}
