package preconditions

import (
	"reflect"
	"testing"
	"time"
)

type testClock struct {
	ret time.Time
}

func (t *testClock) Now() time.Time {
	return t.ret
}

func TestWindowPreconditionFromMap(t *testing.T) {
	tests := []struct {
		name    string
		args    map[string]any
		want    *WindowedPrecondition
		wantErr bool
	}{
		{
			name:    "empty map",
			args:    map[string]any{},
			want:    nil,
			wantErr: true,
		},
		{
			name: "success",
			args: map[string]any{
				"from": "12:00",
				"to":   "08:00",
			},
			want: &WindowedPrecondition{
				startTime: "12:00",
				endTime:   "08:00",
				clock:     &realClock{},
			},
			wantErr: false,
		},
		{
			name: "wrong args",
			args: map[string]any{
				"asdf": "12:00",
				"fdsa": "08:00",
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "missing from",
			args: map[string]any{
				"to": "12:00",
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "missing to",
			args: map[string]any{
				"from": "08:00",
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "wrong type",
			args: map[string]any{
				"from": "12",
				"to":   23,
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := WindowPreconditionFromMap(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("WindowPreconditionFromMap() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("WindowPreconditionFromMap() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_extractHourAndMinute(t *testing.T) {
	type args struct {
		input string
	}
	tests := []struct {
		name    string
		args    args
		want    *Delimiter
		wantErr bool
	}{
		{
			name: "valid",
			args: args{
				input: "01:11",
			},
			want: &Delimiter{
				hour:   1,
				minute: 11,
			},
			wantErr: false,
		},
		{
			name: "valid",
			args: args{
				input: "00:00",
			},
			want: &Delimiter{
				hour:   0,
				minute: 0,
			},
			wantErr: false,
		},
		{
			name: "invalid - not enough runes",
			args: args{
				input: "1:11",
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "invalid - too many runes",
			args: args{
				input: "120:11",
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := extractHourAndMinute(tt.args.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("extractHourAndMinute() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("extractHourAndMinute() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWindowedPrecondition_PerformCheck(t *testing.T) {
	now := time.Now()
	type fields struct {
		startTime string
		endTime   string
		clock     Clock
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name: "no overlap - too early",
			fields: fields{
				startTime: "14:00",
				endTime:   "16:00",
				clock: &testClock{
					ret: time.Date(now.Year(), now.Month(), now.Day(), 13, 10, 30, 0, time.Local),
				},
			},
			want: false,
		},
		{
			name: "no overlap - too late",
			fields: fields{
				startTime: "14:00",
				endTime:   "16:00",
				clock: &testClock{
					ret: time.Date(now.Year(), now.Month(), now.Day(), 17, 10, 30, 0, time.Local),
				},
			},
			want: false,
		},
		{
			name: "no overlap - within",
			fields: fields{
				startTime: "14:00",
				endTime:   "16:00",
				clock: &testClock{
					ret: time.Date(now.Year(), now.Month(), now.Day(), 15, 10, 30, 0, time.Local),
				},
			},
			want: true,
		},
		{
			name: "overlapping time - too early",
			fields: fields{
				startTime: "14:00",
				endTime:   "08:00",
				clock: &testClock{
					ret: time.Date(now.Year(), now.Month(), now.Day(), 12, 10, 30, 0, time.Local),
				},
			},
			want: false,
		},
		{
			name: "overlapping time - too late",
			fields: fields{
				startTime: "14:00",
				endTime:   "08:00",
				clock: &testClock{
					ret: time.Date(now.Year(), now.Month(), now.Day(), 10, 10, 30, 0, time.Local),
				},
			},
			want: false,
		},
		{
			name: "overlapping time - within - tomorrow",
			fields: fields{
				startTime: "14:00",
				endTime:   "08:00",
				clock: &testClock{
					ret: time.Date(now.Year(), now.Month(), now.Day(), 00, 10, 30, 0, time.Local),
				},
			},
			want: true,
		},
		{
			name: "overlapping time - within - today",
			fields: fields{
				startTime: "14:00",
				endTime:   "08:00",
				clock: &testClock{
					ret: time.Date(now.Year(), now.Month(), now.Day(), 18, 10, 30, 0, time.Local),
				},
			},
			want: true,
		},
		{
			name: "overlapping time - within - today",
			fields: fields{
				startTime: "14:00",
				endTime:   "00:00",
				clock: &testClock{
					ret: time.Date(now.Year(), now.Month(), now.Day(), 23, 10, 30, 0, time.Local),
				},
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &WindowedPrecondition{
				startTime: tt.fields.startTime,
				endTime:   tt.fields.endTime,
				clock:     tt.fields.clock,
			}
			if got := c.PerformCheck(); got != tt.want {
				t.Errorf("PerformCheck() = %v, want %v", got, tt.want)
			}
		})
	}
}
