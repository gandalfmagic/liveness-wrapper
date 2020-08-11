package testconsole

import (
	"testing"
	"time"
)

func Test_testConsole_Write(t *testing.T) {
	type fields struct {
		lineIterator chan string
		enabled      bool
	}

	type args struct {
		p []byte
	}

	tests := []struct {
		name        string
		fields      fields
		args        args
		want        int
		wantChannel string
		wantErr     bool
	}{
		{
			name:   "no_channel",
			fields: fields{lineIterator: make(chan string)},
			args:   args{[]byte{'t', 'e', 's', 't'}},
			want:   4,
		},
		{
			name:        "channel",
			fields:      fields{lineIterator: make(chan string), enabled: true},
			args:        args{[]byte{'t', 'e', 's', 't'}},
			want:        4,
			wantChannel: "test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &TestConsole{
				lineIterator: tt.fields.lineIterator,
				enabled:      tt.fields.enabled,
			}

			if tt.fields.enabled {
				go func() {
					gotChannel := <-tt.fields.lineIterator
					if gotChannel != tt.wantChannel {
						t.Errorf("Write() got = %v, want %v", gotChannel, tt.wantChannel)
					}
				}()
			}

			got, err := c.Write(tt.args.p)
			if (err != nil) != tt.wantErr {
				t.Errorf("Write() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Write() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_testConsole_Close(t *testing.T) {
	type fields struct {
		lineIterator chan string
		enabled      bool
	}

	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name:    "nil",
			fields:  fields{lineIterator: make(chan string)},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &TestConsole{
				lineIterator: tt.fields.lineIterator,
				enabled:      tt.fields.enabled,
			}
			if err := c.Close(); (err != nil) != tt.wantErr {
				t.Errorf("Close() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_testConsole_run(t *testing.T) {
	type fields struct {
		lineIterator chan string
		enabled      bool
	}

	type args struct {
		expectedLine string
		timeout      time.Duration
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "empty_expected_line",
			args: args{
				expectedLine: "",
				timeout:      10 * time.Millisecond,
			},
		},
		{
			name: "timeout",
			args: args{
				expectedLine: "test",
				timeout:      10 * time.Millisecond,
			},
			wantErr: true,
		},
		{
			name: "test",
			args: args{
				expectedLine: "test",
				timeout:      10 * time.Millisecond,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lineIterator := make(chan string)
			c := &TestConsole{
				lineIterator: lineIterator,
				enabled:      tt.fields.enabled,
			}

			ch := make(chan error)

			done1 := make(chan struct{})
			go func() {
				got := <-ch
				if tt.wantErr && got == nil {
					t.Fatal("run() wants an error, got nil")
				}
				if !tt.wantErr && got != nil {
					t.Fatalf("run() doesn't want an error, got %s", got)
				}
				close(done1)
			}()

			done2 := make(chan struct{})
			go func() {
				c.run(tt.args.expectedLine, ch, tt.args.timeout)
				close(done2)
			}()

			if tt.args.expectedLine != "" && !tt.wantErr {
				lineIterator <- tt.args.expectedLine
			}

			<-done1
			<-done2
		})
	}
}
