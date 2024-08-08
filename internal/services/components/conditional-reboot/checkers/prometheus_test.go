package checkers

import (
	"testing"
)

func TestPrometheusChecker_evaluateResponse(t *testing.T) {
	tests := []struct {
		name                string
		mapResultsAsHealthy bool
		responseLength      int
		want                bool
	}{
		{
			name:                "query returned data, mapResponseToHealthy=true",
			responseLength:      1,
			mapResultsAsHealthy: true,
			want:                true,
		},
		{
			name:                "query returned no data, mapResponseToHealthy=true",
			responseLength:      0,
			mapResultsAsHealthy: true,
			want:                false,
		},
		{
			name:                "query returned data, mapResponseToHealthy=false",
			responseLength:      1,
			mapResultsAsHealthy: false,
			want:                false,
		},
		{
			name:                "query returned no data, mapResponseToHealthy=false",
			responseLength:      0,
			mapResultsAsHealthy: false,
			want:                true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &PrometheusChecker{
				wantResponse: tt.mapResultsAsHealthy,
			}
			if got := c.evaluateResponse(tt.responseLength); got != tt.want {
				t.Errorf("evaluateResponse() = %v, want %v", got, tt.want)
			}
		})
	}
}
