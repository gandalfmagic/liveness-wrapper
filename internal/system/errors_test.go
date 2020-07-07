package system

import (
	"reflect"
	"testing"
)

func TestNewProcessExitStatusError(t *testing.T) {
	type args struct {
		exitStatus int
	}
	tests := []struct {
		name string
		args args
		want ProcessExitStatusError
	}{
		{
			name: "0",
			args: args{exitStatus: 0},
			want: &processExitStatusError{exitStatus: 0},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewProcessExitStatusError(tt.args.exitStatus); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewProcessExitStatusError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_processExitStatusError_Error(t *testing.T) {
	type fields struct {
		exitStatus byte
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name:   "0",
			fields: fields{exitStatus: 0},
			want:   "the process ended with exit status 0",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &processExitStatusError{
				exitStatus: tt.fields.exitStatus,
			}
			if got := p.Error(); got != tt.want {
				t.Errorf("Error() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_processExitStatusError_ExitStatus(t *testing.T) {
	type fields struct {
		exitStatus byte
	}
	tests := []struct {
		name   string
		fields fields
		want   int
	}{
		{
			name:   "0",
			fields: fields{exitStatus: 0},
			want:   0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &processExitStatusError{
				exitStatus: tt.fields.exitStatus,
			}
			if got := p.ExitStatus(); got != tt.want {
				t.Errorf("ExitStatus() = %v, want %v", got, tt.want)
			}
		})
	}
}
