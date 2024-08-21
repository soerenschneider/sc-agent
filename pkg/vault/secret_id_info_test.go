package vault

import (
	"math"
	"testing"
	"time"
)

func withinTolerance(a, b, epsilon float64) bool {
	if a == b {
		return true
	}

	diff := math.Abs(a - b)
	if b == 0 {
		return diff < epsilon
	}
	return (diff / math.Abs(b)) < epsilon
}

func TestSecretIdInfo_GetPercentage(t *testing.T) {
	type fields struct {
		Expiration      time.Time
		CreationTime    time.Time
		LastUpdatedTime time.Time
		Ttl             int64
	}
	tests := []struct {
		name   string
		fields fields
		want   float64
	}{
		{
			name: "valid for 1h, 30m left, ergo 50% lifetime",
			fields: fields{
				Expiration:   time.Now().Add(30 * 60 * time.Second),
				CreationTime: time.Now().Add(-30 * 60 * time.Second),
			},
			want: 50,
		},
		{
			name: "valid for 1h, 15m left, ergo 25% lifetime",
			fields: fields{
				Expiration:   time.Now().Add(15 * 60 * time.Second),
				CreationTime: time.Now().Add(-45 * 60 * time.Second),
			},
			want: 25,
		},
		{
			name: "no value set for expiration, does not expire, 100% lifetime",
			want: 100,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := SecretIdInfo{
				Expiration:      tt.fields.Expiration,
				CreationTime:    tt.fields.CreationTime,
				LastUpdatedTime: tt.fields.LastUpdatedTime,
				Ttl:             tt.fields.Ttl,
			}
			if got := s.GetPercentage(); !withinTolerance(got, tt.want, 0.001) {
				t.Errorf("GetPercentage() = %v, want %v", got, tt.want)
			}
		})
	}
}
