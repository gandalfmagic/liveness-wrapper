package http

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func Test_server_ReadyHandler(t *testing.T) {
	type fields struct {
		externalAlive chan bool
		isAlive       bool
		isReady       bool
		pingChannel   chan bool
		pingInterval  time.Duration
		updateReady   chan bool
		server        *http.Server
	}
	type args struct {
		method string
		path   string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   int
	}{
		{
			name:   "ReadyHandler_IsReady",
			fields: fields{isReady: true},
			args:   args{method: "GET", path: "/ready"},
			want:   http.StatusOK,
		},
		{
			name:   "ReadyHandler_IsNotReady",
			fields: fields{isReady: false},
			args:   args{method: "GET", path: "/ready"},
			want:   http.StatusServiceUnavailable,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &server{
				externalAlive: tt.fields.externalAlive,
				isAlive:       tt.fields.isAlive,
				isReady:       tt.fields.isReady,
				pingChannel:   tt.fields.pingChannel,
				pingInterval:  tt.fields.pingInterval,
				updateReady:   tt.fields.updateReady,
				server:        tt.fields.server,
			}

			req, err := http.NewRequest(tt.args.method, tt.args.path, nil)
			if err != nil {
				t.Fatal(err)
			}

			rr := httptest.NewRecorder()
			handler := http.HandlerFunc(s.ReadyHandler)
			handler.ServeHTTP(rr, req)

			if status := rr.Code; status != tt.want {
				t.Errorf("handler returned wrong status code: got %v want %v", status, tt.want)
			}
		})
	}
}

func Test_server_AliveHandler(t *testing.T) {
	type fields struct {
		externalAlive chan bool
		isAlive       bool
		isReady       bool
		pingChannel   chan bool
		pingInterval  time.Duration
		updateReady   chan bool
		server        *http.Server
	}
	type args struct {
		method string
		path   string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   int
	}{
		{
			name:   "AliveHandler_IsAlive",
			fields: fields{isAlive: true},
			args:   args{method: "GET", path: "/alive"},
			want:   http.StatusOK,
		},
		{
			name:   "AliveHandler_IsNotAlive",
			fields: fields{isAlive: false},
			args:   args{method: "GET", path: "/alive"},
			want:   http.StatusServiceUnavailable,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &server{
				externalAlive: tt.fields.externalAlive,
				isAlive:       tt.fields.isAlive,
				isReady:       tt.fields.isReady,
				pingChannel:   tt.fields.pingChannel,
				pingInterval:  tt.fields.pingInterval,
				updateReady:   tt.fields.updateReady,
				server:        tt.fields.server,
			}

			req, err := http.NewRequest(tt.args.method, tt.args.path, nil)
			if err != nil {
				t.Fatal(err)
			}

			rr := httptest.NewRecorder()
			handler := http.HandlerFunc(s.AliveHandler)
			handler.ServeHTTP(rr, req)

			if status := rr.Code; status != tt.want {
				t.Errorf("handler returned wrong status code: got %v want %v", status, tt.want)
			}
		})
	}
}

func Test_server_PingHandler(t *testing.T) {
	type fields struct {
		externalAlive chan bool
		isAlive       bool
		isReady       bool
		pingChannel   chan bool
		pingInterval  time.Duration
		updateReady   chan bool
		server        *http.Server
	}
	type args struct {
		method string
		path   string
	}
	type want struct {
		statusCode int
		channel    bool
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   want
	}{
		{
			name:   "PingHandler",
			fields: fields{pingChannel: make(chan bool)},
			args:   args{method: "GET", path: "/ping"},
			want:   want{statusCode: http.StatusOK, channel: true},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &server{
				externalAlive: tt.fields.externalAlive,
				isAlive:       tt.fields.isAlive,
				isReady:       tt.fields.isReady,
				pingChannel:   tt.fields.pingChannel,
				pingInterval:  tt.fields.pingInterval,
				updateReady:   tt.fields.updateReady,
				server:        tt.fields.server,
			}

			done := make(chan struct{})
			go func() {
				defer close(done)
				if channel := <-s.pingChannel; channel != tt.want.channel {
					t.Errorf("ping channel returned wrong value: got %v want %v", channel, tt.want.channel)
				}
			}()

			req, err := http.NewRequest(tt.args.method, tt.args.path, nil)
			if err != nil {
				t.Fatal(err)
			}

			rr := httptest.NewRecorder()
			handler := http.HandlerFunc(s.PingHandler)
			handler.ServeHTTP(rr, req)

			if status := rr.Code; status != tt.want.statusCode {
				t.Errorf("handler returned wrong status code: got %v want %v", status, tt.want.statusCode)
			}

			<-done
		})
	}
}
