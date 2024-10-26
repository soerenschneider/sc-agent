package checkers

import (
	"context"
	"errors"
	"sync"
	"testing"
)

func TestNeedrestartChecker_detectUpdates(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   bool
		want1  bool
	}{
		{
			name: "wants service updates",
			output: `NEEDRESTART-VER: 2.1
NEEDRESTART-KCUR: 3.19.3-tl1+
NEEDRESTART-KEXP: 3.19.3-tl1+
NEEDRESTART-KSTA: 1
NEEDRESTART-SVC: systemd-journald.service
NEEDRESTART-SVC: systemd-machined.service
NEEDRESTART-CONT: LXC web1
NEEDRESTART-SESS: metabase @ user manager service
NEEDRESTART-SESS: root @ session #28017`,
			want:  false,
			want1: true,
		},
		{
			name: "more service updates",
			output: `NEEDRESTART-VER: 3.6
NEEDRESTART-KCUR: 5.14.0-284.25.1.el9_2.x86_64
NEEDRESTART-KEXP: 5.14.0-284.25.1.el9_2.x86_64
NEEDRESTART-KSTA: 2
NEEDRESTART-UCSTA: 0
NEEDRESTART-SVC: dbus-broker.service
NEEDRESTART-SVC: systemd-logind.service
NEEDRESTART-SVC: virtnetworkd.service`,
			want:  false,
			want1: true,
		},
		{
			name: "wants kernel updates",
			output: `NEEDRESTART-VER: 2.1
NEEDRESTART-KCUR: 3.19.3-tl1+
NEEDRESTART-KEXP: 3.19.3-tl1+
NEEDRESTART-KSTA: 3
NEEDRESTART-CONT: LXC web1
NEEDRESTART-SESS: metabase @ user manager service
NEEDRESTART-SESS: root @ session #28017`,
			want:  true,
			want1: false,
		},
		{
			name: "wants nothing",
			output: `NEEDRESTART-VER: 2.1
NEEDRESTART-KCUR: 3.19.3-tl1+
NEEDRESTART-KEXP: 3.19.3-tl1+
NEEDRESTART-KSTA: 1
NEEDRESTART-CONT: LXC web1
NEEDRESTART-SESS: metabase @ user manager service
NEEDRESTART-SESS: root @ session #28017`,
			want:  false,
			want1: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n, _ := NewNeedrestartChecker()
			got, got1 := n.detectUpdates(tt.output)
			if got != tt.want {
				t.Errorf("detectUpdates() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("detectUpdates() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

type needrestartDummy struct {
	out string
	err error
}

func (n *needrestartDummy) Result(_ context.Context) (string, error) {
	return n.out, n.err
}

func TestNeedrestartChecker_IsHealthy(t *testing.T) {
	type fields struct {
		rebootNeeded bool
		needrestart  Needrestart
	}

	tests := []struct {
		name    string
		fields  fields
		ctx     context.Context
		want    bool
		wantErr bool
	}{
		{
			name: "integration",
			fields: fields{
				rebootNeeded: false,
				needrestart: &needrestartDummy{
					out: `NEEDRESTART-VER: 3.6
NEEDRESTART-KCUR: 5.14.0-284.25.1.el9_2.x86_64
NEEDRESTART-KEXP: 5.14.0-284.25.1.el9_2.x86_64
NEEDRESTART-KSTA: 2
NEEDRESTART-UCSTA: 0
NEEDRESTART-SVC: dbus-broker.service
NEEDRESTART-SVC: systemd-logind.service
NEEDRESTART-SVC: virtnetworkd.service`,
					err: nil,
				},
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "empty output signals all is good",
			fields: fields{
				rebootNeeded: false,
				needrestart: &needrestartDummy{
					out: "",
					err: nil,
				},
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "empty output signals all is good",
			fields: fields{
				rebootNeeded: false,
				needrestart: &needrestartDummy{
					out: "",
					err: errors.New("some error"),
				},
			},
			want:    false,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := &NeedrestartChecker{
				rebootNeeded: tt.fields.rebootNeeded,
				sync:         sync.Mutex{},
				needrestart:  tt.fields.needrestart,
			}
			got, err := n.IsHealthy(tt.ctx)
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
