package formatter

import (
	"testing"
)

func TestEnvVarFormatter_Format(t *testing.T) {
	tests := []struct {
		name          string
		uppercaseKeys bool
		data          map[string]any
		want          string
	}{
		{
			name: "empty",
			data: map[string]any{},
			want: "",
		},
		{
			name: "sorted keys",
			data: map[string]any{"password": "secret", "api_key": "abc", "user": "admin"},
			want: "api_key=abc\npassword=secret\nuser=admin\n",
		},
		{
			name:          "uppercase keys",
			uppercaseKeys: true,
			data:          map[string]any{"password": "secret", "api_key": "abc"},
			want:          "API_KEY=abc\nPASSWORD=secret\n",
		},
		{
			name: "non-string values",
			data: map[string]any{"port": 8080, "enabled": true},
			want: "enabled=true\nport=8080\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewEnvVarFormatter(tt.uppercaseKeys).Format(tt.data)
			if err != nil {
				t.Fatalf("Format() unexpected error: %v", err)
			}
			if string(got) != tt.want {
				t.Fatalf("Format() got = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestEnvVarFormatter_FormatIsDeterministic(t *testing.T) {
	data := map[string]any{"a": "1", "b": "2", "c": "3", "d": "4", "e": "5", "f": "6"}
	formatter := NewEnvVarFormatter(false)

	first, err := formatter.Format(data)
	if err != nil {
		t.Fatal(err)
	}
	for range 50 {
		got, err := formatter.Format(data)
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != string(first) {
			t.Fatalf("Format() not deterministic: %q vs %q", first, got)
		}
	}
}
