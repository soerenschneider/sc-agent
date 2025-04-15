package http_replication

import "testing"

func TestFileTest_Accept(t *testing.T) {
	type fields struct {
		Type   string
		Invert bool
		Arg    string
	}
	type args struct {
		value []byte
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    bool
		wantErr bool
	}{
		{
			name: "sha256 match",
			fields: fields{
				Type: "sha256",
				Arg:  "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824", // sha256("hello")
			},
			args: args{
				value: []byte("hello"),
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "sha256 no match",
			fields: fields{
				Type: "sha256",
				Arg:  "wronghashvalue",
			},
			args:    args{[]byte("hello")},
			want:    false,
			wantErr: false,
		},
		{
			name: "sha256 with invert",
			fields: fields{
				Type:   "sha256",
				Invert: true,
				Arg:    "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824",
			},
			args:    args{[]byte("hello")},
			want:    false,
			wantErr: false,
		},
		{
			name: "regex match",
			fields: fields{
				Type: "regex",
				Arg:  "^h.*o$",
			},
			args:    args{[]byte("hello")},
			want:    true,
			wantErr: false,
		},
		{
			name: "regex no match",
			fields: fields{
				Type: "regex",
				Arg:  "^x.*z$",
			},
			args:    args{[]byte("hello")},
			want:    false,
			wantErr: false,
		},
		{
			name: "regex with invert",
			fields: fields{
				Type:   "regex",
				Invert: true,
				Arg:    "^h.*o$",
			},
			args:    args{[]byte("hello")},
			want:    false,
			wantErr: false,
		},
		{
			name: "regex invalid pattern",
			fields: fields{
				Type: "regex",
				Arg:  "(unclosed",
			},
			args:    args{[]byte("hello")},
			want:    false,
			wantErr: true,
		},
		{
			name: "starts_with match",
			fields: fields{
				Type: "starts_with",
				Arg:  "he",
			},
			args:    args{[]byte("hello")},
			want:    true,
			wantErr: false,
		},
		{
			name: "starts_with no match",
			fields: fields{
				Type: "starts_with",
				Arg:  "wo",
			},
			args:    args{[]byte("hello")},
			want:    false,
			wantErr: false,
		},
		{
			name: "starts_with inverted match",
			fields: fields{
				Type:   "starts_with",
				Invert: true,
				Arg:    "he",
			},
			args:    args{[]byte("hello")},
			want:    false,
			wantErr: false,
		},
		{
			name: "ends_with match",
			fields: fields{
				Type: "ends_with",
				Arg:  "lo",
			},
			args:    args{[]byte("hello")},
			want:    true,
			wantErr: false,
		},
		{
			name: "ends_with no match",
			fields: fields{
				Type: "ends_with",
				Arg:  "he",
			},
			args:    args{[]byte("hello")},
			want:    false,
			wantErr: false,
		},
		{
			name: "ends_with inverted",
			fields: fields{
				Type:   "ends_with",
				Invert: true,
				Arg:    "lo",
			},
			args:    args{[]byte("hello")},
			want:    false,
			wantErr: false,
		},
		{
			name: "unknown type",
			fields: fields{
				Type: "unknown_type",
				Arg:  "something",
			},
			args:    args{[]byte("value")},
			want:    false,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &FileValidation{
				Test:         tt.fields.Type,
				InvertResult: tt.fields.Invert,
				Arg:          tt.fields.Arg,
			}
			got, err := f.Accept(tt.args.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("Accept() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Accept() got = %v, want %v", got, tt.want)
			}
		})
	}
}
