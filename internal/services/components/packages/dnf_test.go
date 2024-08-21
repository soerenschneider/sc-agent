package packages

import (
	"reflect"
	"testing"

	"github.com/soerenschneider/sc-agent/internal/domain"
)

var exampleDnfListOutput = `
alternatives.x86_64                                                                                                   1.24-1.el9                                                                                                      @baseos   
annobin.x86_64                                                                                                        12.31-2.el9                                                                                                     @appstream
audit.x86_64                                                                                                          3.1.2-2.el9                                                                                                     @baseos   
audit-libs.x86_64                                                                                                     3.1.2-2.el9                                                                                                     @baseos   
`

var exampleDnfCheckUpdateOutput = `tpm2-tss.x86_64                                                                                               3.2.3-1.el9                                                                                       centos-stream-9-stable-baseos   
xfsprogs.x86_64                                                                                               6.4.0-3.el9                                                                                       centos-stream-9-stable-baseos   
yum.noarch                                                                                                    4.14.0-15.el9                                                                                     centos-stream-9-stable-baseos   
Obsoleting Packages
grub2-tools.x86_64                                                                                            1:2.06-82.el9                                                                                     centos-stream-9-stable-baseos   
    grub2-tools.x86_64                                                                                        1:2.06-80.el9                                                                                     @anaconda                       
grub2-tools-efi.x86_64 
`

func Test_parseDnfListOutput(t *testing.T) {
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
				exampleDnfListOutput,
			},
			want: []domain.PackageInfo{
				{
					Name:    "alternatives.x86_64",
					Version: "1.24-1.el9",
					Repo:    "@baseos",
				},
				{
					Name:    "annobin.x86_64",
					Version: "12.31-2.el9",
					Repo:    "@appstream",
				},
				{
					Name:    "audit.x86_64",
					Version: "3.1.2-2.el9",
					Repo:    "@baseos",
				},
				{
					Name:    "audit-libs.x86_64",
					Version: "3.1.2-2.el9",
					Repo:    "@baseos",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseDnfListOutput(tt.args.output); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseDnfListOutput() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_parseDnfCheckUpdateOutput(t *testing.T) {
	type args struct {
		output string
	}
	tests := []struct {
		name    string
		args    args
		want    []domain.PackageInfo
		wantErr bool
	}{
		{
			name: "happy path",
			args: args{
				output: exampleDnfCheckUpdateOutput,
			},
			want: []domain.PackageInfo{
				{
					Name:    "tpm2-tss.x86_64",
					Version: "3.2.3-1.el9",
					Repo:    "centos-stream-9-stable-baseos",
				},
				{
					Name:    "xfsprogs.x86_64",
					Version: "6.4.0-3.el9",
					Repo:    "centos-stream-9-stable-baseos",
				},
				{
					Name:    "yum.noarch",
					Version: "4.14.0-15.el9",
					Repo:    "centos-stream-9-stable-baseos",
				},
				{
					Name:    "grub2-tools.x86_64",
					Version: "1:2.06-82.el9",
					Repo:    "centos-stream-9-stable-baseos",
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseDnfCheckUpdateOutput(tt.args.output)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseDnfCheckUpdateOutput() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseDnfCheckUpdateOutput() got = %v, want %v", got, tt.want)
			}
		})
	}
}
