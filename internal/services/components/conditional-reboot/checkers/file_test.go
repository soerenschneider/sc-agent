package checkers

import (
	"context"
	"testing"
)

func TestFileChecker_IsHealthy(t *testing.T) {
	type fields struct {
		file         string
		wantsAbsence bool
	}
	tests := []struct {
		name    string
		fields  fields
		want    bool
		wantErr bool
	}{
		{
			name: "wants absence - file exists",
			fields: fields{
				file:         "./file_test.go",
				wantsAbsence: true,
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "doesn't want absence - file exists",
			fields: fields{
				file:         "./file_test.go",
				wantsAbsence: false,
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "wants absence - file doesn't exists",
			fields: fields{
				file:         "./file_test.go.notexisting",
				wantsAbsence: true,
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "doesn't want absence - file doesn't exists",
			fields: fields{
				file:         "./file_test.go.notexisting",
				wantsAbsence: false,
			},
			want:    false,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &FileChecker{
				file:         tt.fields.file,
				wantsAbsence: tt.fields.wantsAbsence,
			}
			got, err := c.IsHealthy(context.Background())
			if (err != nil) != tt.wantErr {
				t.Errorf("IsHealthy() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("IsHealthy() got = %v, want %v", got, tt.want)
			}
		})
	}
}
