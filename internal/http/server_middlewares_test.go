package http

import (
	"bytes"
	"fmt"
	"github.com/gandalfmagic/liveness-wrapper/pkg/logger"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func Test_inStringSlice(t *testing.T) {
	type args struct {
		slice []string
		str   string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "true",
			args: args{slice: []string{"aaa", "bbb"}, str: "aaa"},
			want: true,
		},
		{
			name: "false",
			args: args{slice: []string{"bbb"}, str: "aaa"},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := inStringSlice(tt.args.slice, tt.args.str); got != tt.want {
				t.Errorf("inStringSlice() = %v, want %v", got, tt.want)
			}
		})
	}
}
func testGetHandler() http.HandlerFunc {
	fn := func(rw http.ResponseWriter, req *http.Request) {
	}
	return fn
}

type testWriteCloser struct {
	*bytes.Buffer
}

func (mwc *testWriteCloser) Close() error {
	return nil
}

func TestLoggingMiddleware(t *testing.T) {
	t.Run("standard", func(t *testing.T) {
		var buf bytes.Buffer

		wc := &testWriteCloser{&buf}
		defer func() {
			_ = wc.Close()
		}()

		// Redirect the logger to a buffer
		zLogger, _ := logger.NewLogger(wc, "test", "debug")

		// Create test HTTP server
		ts := httptest.NewServer(Log(zLogger, testGetHandler()))
		defer ts.Close()

		// Trigger a request to get output to log
		_, _ = http.Get(fmt.Sprintf("%s/", ts.URL))

		// Test output
		t.Log(buf.String())
		if buf.Len() == 0 {
			t.Error("No information logged to STDOUT")
		}
		if strings.Count(buf.String(), "\n") > 1 {
			t.Error("Expected only a single line of log output")
		}
		if !strings.Contains(buf.String(), "Go-http-client/1.1") {
			t.Error("The output mus contains the go agent string: Go-http-client/1.1")
		}
	})
}

func TestMethodsMiddleware(t *testing.T) {
	type args struct {
		methods []string
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{
			name: "permitted",
			args: args{[]string{"GET"}},
			want: http.StatusOK,
		},
		{
			name: "permitted",
			args: args{[]string{"GET", "POST"}},
			want: http.StatusOK,
		},
		{
			name: "permitted",
			args: args{[]string{"POST"}},
			want: http.StatusMethodNotAllowed,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test HTTP server
			ts := httptest.NewServer(Methods(tt.args.methods, nil, testGetHandler()))
			defer ts.Close()

			// Trigger a request to get output to log
			r, _ := http.Get(fmt.Sprintf("%s/", ts.URL))

			if r.StatusCode != tt.want {
				t.Errorf("Expected status code %v, got %v", tt.want, r.StatusCode)
			}
		})
	}
}
