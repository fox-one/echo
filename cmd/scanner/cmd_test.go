package main

import (
	"os/exec"
	"reflect"
	"testing"
)

func Test_parseCmd(t *testing.T) {
	type args struct {
		cmd string
	}
	tests := []struct {
		name  string
		args  args
		want  []string
		want1 bool
	}{
		{
			name:  "string array",
			args:  args{cmd: `["foo","--bar","haha"]`},
			want:  []string{"foo", "--bar", "haha"},
			want1: true,
		},
		{
			name:  "string",
			args:  args{cmd: "foo --bar haha"},
			want:  []string{"foo", "--bar", "haha"},
			want1: true,
		},
		{
			name:  "with quote",
			args:  args{cmd: `./group --debug --name "haha xixi"`},
			want:  []string{"./group", "--debug", "--name", `"haha xixi"`},
			want1: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := parseCmd(tt.args.cmd)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseCmd() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("parseCmd() got1 = %v, want %v", got1, tt.want1)
			}

			if got1 {
				cmd := exec.Command(got[0], got[1:]...)
				t.Log(cmd.String())
			}
		})
	}
}
