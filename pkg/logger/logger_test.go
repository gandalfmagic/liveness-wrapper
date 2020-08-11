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
		level  int
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
				level:  0,
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
				level:  4,
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
				level:  0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf testWriter
			Configure(&buf, tt.args.prefix, tt.args.level)

			if got := defaultLogger.GetLevel(); got != tt.want.level {
				t.Errorf("Configure expected level: %v, got %v", tt.want.level, got)
			}

			Infof("test")
			if !strings.Contains(buf.String(), tt.want.prefix) {
				t.Errorf("Configure expected prefix: %v, got %v", tt.want.prefix, buf.String())
			}

			defaultLogger = nil
		})
	}
}
