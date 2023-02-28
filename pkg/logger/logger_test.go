package logger

import (
	"bytes"
	"strings"
	"testing"
)

type testWriter struct {
	bytes.Buffer
}

func (w testWriter) Close() error {
	return nil
}

func TestConfigure(t *testing.T) {
	type args struct {
		prefix string
		level  string
	}
	type want struct {
		prefix string
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "empty",
			args: args{
				prefix: "",
				level:  "",
			},
			want: want{
				prefix: "",
			},
		},
		{
			name: "error_level",
			args: args{
				prefix: "",
				level:  "ERROR",
			},
			want: want{
				prefix: "",
			},
		},
		{
			name: "prefix",
			args: args{
				prefix: "prefix",
				level:  "",
			},
			want: want{
				prefix: "[prefix]",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf testWriter
			l, _ := NewLogger(&buf, tt.args.prefix, tt.args.level)

			l.Infof("test")
			if !strings.Contains(buf.String(), tt.want.prefix) {
				t.Errorf("Configure expected prefix: %v, got %v", tt.want.prefix, buf.String())
			}

			l.Close()
		})
	}
}
