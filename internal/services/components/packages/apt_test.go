package packages

import (
	"reflect"
	"testing"

	"github.com/soerenschneider/sc-agent/internal/domain"
)

func Test_parseAptListUpgradeableOutput(t *testing.T) {
	type args struct {
		output string
	}
	tests := []struct {
		name string
		args args
		want domain.CheckUpdateResult
	}{
		{
			name: "happy path",
			args: args{
				output: `git-man/jammy-updates,jammy-security 1:2.34.1-1ubuntu1.11 all [upgradable from: 1:2.34.1-1ubuntu1]
git/jammy-updates,jammy-security 1:2.34.1-1ubuntu1.11 amd64 [upgradable from: 1:2.34.1-1ubuntu1]`,
			},
			want: domain.CheckUpdateResult{
				UpdatesAvailable: true,
				UpdatablePackages: []domain.PackageInfo{
					{Name: "git-man", Version: "", Repo: ""},
					{Name: "git", Version: "", Repo: ""},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseAptListUpgradeableOutput(tt.args.output); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseAptListUpgradeableOutput() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_parseAptListOutput(t *testing.T) {
	type args struct {
		output string
	}
	tests := []struct {
		name string
		args args
		want []domain.PackageInfo
	}{
		{
			name: "happy path",
			args: args{
				output: `7kaa/stable 2.15.5+dfsg-1 amd64
7zip-standalone/stable-backports 24.08+dfsg-1~bpo12+1 amd64
7zip/stable 22.01+dfsg-8 amd64`,
			},
			want: []domain.PackageInfo{
				{Name: "7kaa", Version: "2.15.5+dfsg-1", Repo: "stable"},
				{Name: "7zip-standalone", Version: "24.08+dfsg-1~bpo12+1", Repo: "stable-backports"},
				{Name: "7zip", Version: "22.01+dfsg-8", Repo: "stable"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseAptListOutput(tt.args.output); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseAptListOutput() = %v, want %v", got, tt.want)
			}
		})
	}
}
