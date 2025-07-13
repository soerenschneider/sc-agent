package mqtt

import (
	"testing"
)

func Test_getResponseTopic(t *testing.T) {
	type args struct {
		requestTopic string
		clientID     string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "no client id needed",
			args: args{
				requestTopic: "sc-agent/hostname.tld/power_status/cmd-name",
				clientID:     "xxx",
			},
			want: "sc-agent/hostname.tld/power_status/response/cmd-name",
		},
		{
			name: "client id used for groups",
			args: args{
				requestTopic: "sc-agent/group/foo/power_status/cmd-name",
				clientID:     "xxx",
			},
			want: "sc-agent/group/foo/power_status/response/cmd-name/xxx",
		},
		{
			name: "replace sc-agent in client-id",
			args: args{
				requestTopic: "sc-agent/group/foo/power_status/cmd-name",
				clientID:     "sc-agent-xxx",
			},
			want: "sc-agent/group/foo/power_status/response/cmd-name/xxx",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getResponseTopic(tt.args.requestTopic, tt.args.clientID); got != tt.want {
				t.Errorf("getResponseTopic() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getTopicForGroup(t *testing.T) {
	type args struct {
		globalPrefix string
		groupName    string
		topicPattern string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "implicit prefix",
			args: args{
				groupName:    "mpd",
				topicPattern: "system/shutdown",
			},
			want: "sc-agent/group/mpd/system/shutdown",
		},
		{
			name: "explicit prefix",
			args: args{
				globalPrefix: "something-else",
				groupName:    "mpd",
				topicPattern: "system/shutdown",
			},
			want: "something-else/group/mpd/system/shutdown",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getTopicForGroup(tt.args.globalPrefix, tt.args.groupName, tt.args.topicPattern); got != tt.want {
				t.Errorf("getTopicForGroup() = %v, want %v", got, tt.want)
			}
		})
	}
}
