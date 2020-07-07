package system

import (
	"context"
	"github.com/gandalfmagic/liveness-wrapper/pkg/logger"
	"os"
	"path/filepath"
	"testing"
	"time"
)

var testDirectory = "../../test"

func Test_wrapperHandler_canRestart(t *testing.T) {
	type fields struct {
		arg          []string
		ctx          context.Context
		failOnStdErr bool
		hideStdErr   bool
		hideStdOut   bool
		path         string
		restart      WrapperRestartMode
	}
	type args struct {
		contextIsCanceling bool
		exitStatus         int
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name:   "canRestart_Never_1",
			fields: fields{restart: WrapperRestartNever},
			args:   args{contextIsCanceling: false, exitStatus: 0},
			want:   false,
		},
		{
			name:   "canRestart_Never_2",
			fields: fields{restart: WrapperRestartNever},
			args:   args{contextIsCanceling: true, exitStatus: 0},
			want:   false,
		},
		{
			name:   "canRestart_Never_3",
			fields: fields{restart: WrapperRestartNever},
			args:   args{contextIsCanceling: false, exitStatus: 1},
			want:   false,
		},
		{
			name:   "canRestart_Never_4",
			fields: fields{restart: WrapperRestartNever},
			args:   args{contextIsCanceling: true, exitStatus: 1},
			want:   false,
		},
		{
			name:   "canRestart_OnError_1",
			fields: fields{restart: WrapperRestartOnError},
			args:   args{contextIsCanceling: false, exitStatus: 0},
			want:   false,
		},
		{
			name:   "canRestart_OnError_2",
			fields: fields{restart: WrapperRestartOnError},
			args:   args{contextIsCanceling: true, exitStatus: 0},
			want:   false,
		},
		{
			name:   "canRestart_OnError_3",
			fields: fields{restart: WrapperRestartOnError},
			args:   args{contextIsCanceling: false, exitStatus: 1},
			want:   true,
		},
		{
			name:   "canRestart_OnError_4",
			fields: fields{restart: WrapperRestartOnError},
			args:   args{contextIsCanceling: true, exitStatus: 1},
			want:   false,
		},
		{
			name:   "canRestart_Always_1",
			fields: fields{restart: WrapperRestartAlways},
			args:   args{contextIsCanceling: false, exitStatus: 0},
			want:   true,
		},
		{
			name:   "canRestart_Always_2",
			fields: fields{restart: WrapperRestartAlways},
			args:   args{contextIsCanceling: true, exitStatus: 0},
			want:   false,
		},
		{
			name:   "canRestart_Always_3",
			fields: fields{restart: WrapperRestartAlways},
			args:   args{contextIsCanceling: false, exitStatus: 1},
			want:   true,
		},
		{
			name:   "canRestart_Always_4",
			fields: fields{restart: WrapperRestartAlways},
			args:   args{contextIsCanceling: true, exitStatus: 1},
			want:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &wrapperHandler{
				arg:          tt.fields.arg,
				ctx:          tt.fields.ctx,
				failOnStdErr: tt.fields.failOnStdErr,
				hideStdErr:   tt.fields.hideStdErr,
				hideStdOut:   tt.fields.hideStdOut,
				path:         tt.fields.path,
				restart:      tt.fields.restart,
			}
			if got := p.canRestart(tt.args.contextIsCanceling, tt.args.exitStatus); got != tt.want {
				t.Errorf("canRestart() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_wrapperHandler_do(t *testing.T) {
	type fields struct {
		arg          []string
		failOnStdErr bool
		hideStdErr   bool
		hideStdOut   bool
		path         string
		restart      WrapperRestartMode
	}
	type want struct {
		statusBeforeStart WrapperStatus
		statusAfterStart  WrapperStatus
		statusAfterDone   WrapperStatus
		error             bool
	}
	tests := []struct {
		name   string
		fields fields
		want   want
	}{
		{
			name: "Command_not_found_error",
			fields: fields{
				arg:          []string{},
				failOnStdErr: false,
				hideStdErr:   false,
				hideStdOut:   false,
				path:         filepath.Join(testDirectory, "command_not_found.sh"),
				restart:      WrapperRestartNever,
			},
			want: want{
				statusBeforeStart: WrapperStatusStopped,
				statusAfterStart:  WrapperStatusError,
				statusAfterDone:   WrapperStatusError,
				error:             true,
			},
		},
		{
			name: "Permission_denied_error",
			fields: fields{
				arg:          []string{},
				failOnStdErr: false,
				hideStdErr:   false,
				hideStdOut:   false,
				path:         filepath.Join(testDirectory, "no_permissions.sh"),
				restart:      WrapperRestartNever,
			},
			want: want{
				statusBeforeStart: WrapperStatusStopped,
				statusAfterStart:  WrapperStatusError,
				statusAfterDone:   WrapperStatusError,
				error:             true,
			},
		},
		{
			name: "Simple_run",
			fields: fields{
				arg:          []string{},
				failOnStdErr: false,
				hideStdErr:   false,
				hideStdOut:   false,
				path:         filepath.Join(testDirectory, "test.sh"),
				restart:      WrapperRestartNever,
			},
			want: want{
				statusBeforeStart: WrapperStatusStopped,
				statusAfterStart:  WrapperStatusRunning,
				statusAfterDone:   WrapperStatusStopped,
				error:             false,
			},
		},
		{
			name: "Simple_run_error",
			fields: fields{
				arg:          []string{},
				failOnStdErr: false,
				hideStdErr:   false,
				hideStdOut:   false,
				path:         filepath.Join(testDirectory, "error_10.sh"),
				restart:      WrapperRestartNever,
			},
			want: want{
				statusBeforeStart: WrapperStatusStopped,
				statusAfterStart:  WrapperStatusRunning,
				statusAfterDone:   WrapperStatusError,
				error:             true,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &wrapperHandler{
				arg:          tt.fields.arg,
				ctx:          context.Background(),
				failOnStdErr: tt.fields.failOnStdErr,
				hideStdErr:   tt.fields.hideStdErr,
				hideStdOut:   tt.fields.hideStdOut,
				path:         tt.fields.path,
				restart:      tt.fields.restart,
			}

			chanWrapperData := make(chan WrapperData)
			done := make(chan struct{})

			var wrapperStatus WrapperStatus
			var wrapperError error
			go func() {
				defer close(done)
				for wd := range chanWrapperData {
					wrapperStatus = wd.WrapperStatus
					wrapperError = wd.Err
					if wd.Done {
						return
					}
				}
			}()

			if wrapperStatus != tt.want.statusBeforeStart {
				t.Errorf("expected wrapperStatus == %v, got %v", tt.want.statusBeforeStart, wrapperStatus)
			}

			t.Log("start the process")
			go p.do(chanWrapperData)

			t.Log("wait to ensure the process is running")
			time.Sleep(10 * time.Millisecond)

			if tt.want.statusAfterStart != wrapperStatus {
				t.Errorf("expected wrapperStatus == %v, got %v", tt.want.statusAfterStart, wrapperStatus)
			}

			<-done
			time.Sleep(100 * time.Millisecond)
			if tt.want.statusAfterDone != wrapperStatus {
				t.Errorf("expected wrapperStatus != %v, got %v", tt.want.statusAfterDone, wrapperStatus)
			}

			if tt.want.error && wrapperError == nil {
				t.Errorf("expected an error, got %v", wrapperError)
			}

			if !tt.want.error && wrapperError != nil {
				t.Errorf("no error expected, got %v", wrapperError)
			}
		})
	}
}

func Test_wrapperHandler_do_With_restart(t *testing.T) {
	type fields struct {
		arg          []string
		failOnStdErr bool
		hideStdErr   bool
		hideStdOut   bool
		path         string
		restart      WrapperRestartMode
	}
	type args struct {
		cancelWhileRunning bool
	}
	type want struct {
		statusAfterFirstExit WrapperStatus
		err                  bool
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   want
	}{
		{
			name: "On_error_Cancel_while_IS_running",
			fields: fields{
				arg:          []string{},
				failOnStdErr: false,
				hideStdErr:   false,
				hideStdOut:   false,
				path:         filepath.Join(testDirectory, "error_10.sh"),
				restart:      WrapperRestartOnError,
			},
			args: args{
				cancelWhileRunning: true,
			},
			want: want{
				statusAfterFirstExit: WrapperStatusError,
				err:                  true,
			},
		},
		{
			name: "On_error_Cancel_while_NOT_running",
			fields: fields{
				arg:          []string{},
				failOnStdErr: false,
				hideStdErr:   false,
				hideStdOut:   false,
				path:         filepath.Join(testDirectory, "error_10.sh"),
				restart:      WrapperRestartOnError,
			},
			args: args{
				cancelWhileRunning: false,
			},
			want: want{
				statusAfterFirstExit: WrapperStatusError,
				err:                  true,
			},
		},
		{
			name: "Always_Get_error_Cancel_while_IS_running",
			fields: fields{
				arg:          []string{},
				failOnStdErr: false,
				hideStdErr:   false,
				hideStdOut:   false,
				path:         filepath.Join(testDirectory, "error_10.sh"),
				restart:      WrapperRestartAlways,
			},
			args: args{
				cancelWhileRunning: true,
			},
			want: want{
				statusAfterFirstExit: WrapperStatusError,
				err:                  true,
			},
		},
		{
			name: "Always_Get_error_Cancel_while_NOT_running",
			fields: fields{
				arg:          []string{},
				failOnStdErr: false,
				hideStdErr:   false,
				hideStdOut:   false,
				path:         filepath.Join(testDirectory, "error_10.sh"),
				restart:      WrapperRestartAlways,
			},
			args: args{
				cancelWhileRunning: false,
			},
			want: want{
				statusAfterFirstExit: WrapperStatusError,
				err:                  true,
			},
		},
		{
			name: "Always_Exit_without_error_Cancel_while_IS_running",
			fields: fields{
				arg:          []string{},
				failOnStdErr: false,
				hideStdErr:   false,
				hideStdOut:   false,
				path:         filepath.Join(testDirectory, "test.sh"),
				restart:      WrapperRestartAlways,
			},
			args: args{
				cancelWhileRunning: true,
			},
			want: want{
				statusAfterFirstExit: WrapperStatusStopped,
				err:                  true,
			},
		},
		{
			name: "Always_Exit_without_error_Cancel_while_NOT_running",
			fields: fields{
				arg:          []string{},
				failOnStdErr: false,
				hideStdErr:   false,
				hideStdOut:   false,
				path:         filepath.Join(testDirectory, "test.sh"),
				restart:      WrapperRestartAlways,
			},
			args: args{
				cancelWhileRunning: false,
			},
			want: want{
				statusAfterFirstExit: WrapperStatusStopped,
				err:                  false,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			// create the context
			ctx, cancel := context.WithCancel(context.Background())

			p := &wrapperHandler{
				arg:             tt.fields.arg,
				ctx:             ctx,
				failOnStdErr:    tt.fields.failOnStdErr,
				hideStdErr:      tt.fields.hideStdErr,
				hideStdOut:      tt.fields.hideStdOut,
				path:            tt.fields.path,
				restart:         tt.fields.restart,
				restartInterval: 50 * time.Millisecond,
			}

			chanWrapperData := make(chan WrapperData)
			done := make(chan struct{})

			var wrapperStatus WrapperStatus
			var wrapperError error
			go func() {
				defer close(done)
				for wd := range chanWrapperData {
					wrapperStatus = wd.WrapperStatus
					wrapperError = wd.Err
					if wd.Done {
						return
					}
				}
			}()

			if wrapperStatus != WrapperStatusStopped {
				t.Errorf("expected wrapperStatus == WrapperStatusStopped, got %v", wrapperStatus)
			}

			t.Log("start the process")
			go p.do(chanWrapperData)

			t.Log("wait to ensure the process is running")
			time.Sleep(10 * time.Millisecond)

			if wrapperStatus != WrapperStatusRunning {
				t.Errorf("expected wrapperStatus == WrapperStatusRunning, got %v", wrapperStatus)
			}

			t.Log("wait for the process exit for the first time")
			time.Sleep(410 * time.Millisecond)

			if wrapperStatus != tt.want.statusAfterFirstExit {
				t.Errorf("expected wrapperStatus == %v, got %v", tt.want.statusAfterFirstExit, wrapperStatus)
			}

			if tt.args.cancelWhileRunning {
				t.Log("wait for the process to restart")
				time.Sleep(50 * time.Millisecond)

				if wrapperStatus != WrapperStatusRunning {
					t.Errorf("expected wrapperStatus == WrapperStatusRunning, got %v", wrapperStatus)
				}
			}

			t.Log("cancel the context to terminate the process")
			cancel()
			<-done
			if !tt.want.err && wrapperStatus != WrapperStatusStopped {
				t.Errorf("expected wrapperStatus != WrapperStatusStopped, got %v", wrapperStatus)
			}

			if tt.want.err && wrapperStatus != WrapperStatusError {
				t.Errorf("expected wrapperStatus != WrapperStatusError, got %v", wrapperStatus)
			}

			if tt.want.err && wrapperError == nil {
				t.Errorf("expected an error, got %v", wrapperError)
			}

			if !tt.want.err && wrapperError != nil {
				t.Errorf("no error expected, got %v", wrapperError)
			}
		})
	}
}

func Test_wrapperHandler_do_Log_error(t *testing.T) {
	type fields struct {
		arg          []string
		failOnStdErr bool
		hideStdErr   bool
		hideStdOut   bool
		path         string
		restart      WrapperRestartMode
	}
	type args struct {
		cancelWhileRunning bool
	}
	type want struct {
		statusAfterFirstExit WrapperStatus
		err                  bool
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   want
	}{
		{
			name: "Restart_on_error_Cancel_while_IS_running",
			fields: fields{
				arg:          []string{},
				failOnStdErr: true,
				hideStdErr:   false,
				hideStdOut:   false,
				path:         filepath.Join(testDirectory, "error_10_error_log.sh"),
				restart:      WrapperRestartOnError,
			},
			args: args{
				cancelWhileRunning: true,
			},
			want: want{
				statusAfterFirstExit: WrapperStatusError,
				err:                  true,
			},
		},
		{
			name: "Restart_on_error_Cancel_while_NOT_running",
			fields: fields{
				arg:          []string{},
				failOnStdErr: true,
				hideStdErr:   false,
				hideStdOut:   false,
				path:         filepath.Join(testDirectory, "error_10_error_log.sh"),
				restart:      WrapperRestartOnError,
			},
			args: args{
				cancelWhileRunning: false,
			},
			want: want{
				statusAfterFirstExit: WrapperStatusError,
				err:                  true,
			},
		},
		{
			name: "Restart_always_Get_error_Cancel_while_IS_running",
			fields: fields{
				arg:          []string{},
				failOnStdErr: true,
				hideStdErr:   false,
				hideStdOut:   false,
				path:         filepath.Join(testDirectory, "error_10_error_log.sh"),
				restart:      WrapperRestartAlways,
			},
			args: args{
				cancelWhileRunning: true,
			},
			want: want{
				statusAfterFirstExit: WrapperStatusError,
				err:                  true,
			},
		},
		{
			name: "Restart_always_Get_error_Cancel_while_NOT_running",
			fields: fields{
				arg:          []string{},
				failOnStdErr: true,
				hideStdErr:   false,
				hideStdOut:   false,
				path:         filepath.Join(testDirectory, "error_10_error_log.sh"),
				restart:      WrapperRestartAlways,
			},
			args: args{
				cancelWhileRunning: false,
			},
			want: want{
				statusAfterFirstExit: WrapperStatusError,
				err:                  true,
			},
		},
		{
			name: "Restart_always_Exit_without_error_Cancel_while_IS_running",
			fields: fields{
				arg:          []string{},
				failOnStdErr: true,
				hideStdErr:   false,
				hideStdOut:   false,
				path:         filepath.Join(testDirectory, "test_error_log.sh"),
				restart:      WrapperRestartAlways,
			},
			args: args{
				cancelWhileRunning: true,
			},
			want: want{
				statusAfterFirstExit: WrapperStatusStopped,
				err:                  true,
			},
		},
		{
			name: "Restart_always_Exit_without_error_Cancel_while_NOT_running",
			fields: fields{
				arg:          []string{},
				failOnStdErr: true,
				hideStdErr:   false,
				hideStdOut:   false,
				path:         filepath.Join(testDirectory, "test_error_log.sh"),
				restart:      WrapperRestartAlways,
			},
			args: args{
				cancelWhileRunning: false,
			},
			want: want{
				statusAfterFirstExit: WrapperStatusStopped,
				err:                  false,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			logger.Configure(os.Stdout, "test", "ERROR")

			// create the context
			ctx, cancel := context.WithCancel(context.Background())

			p := &wrapperHandler{
				arg:             tt.fields.arg,
				ctx:             ctx,
				failOnStdErr:    tt.fields.failOnStdErr,
				hideStdErr:      tt.fields.hideStdErr,
				hideStdOut:      tt.fields.hideStdOut,
				path:            tt.fields.path,
				restart:         tt.fields.restart,
				restartInterval: 50 * time.Millisecond,
			}

			chanWrapperData := make(chan WrapperData)
			done := make(chan struct{})

			var wrapperStatus WrapperStatus
			var wrapperError error
			go func() {
				defer close(done)
				for wd := range chanWrapperData {
					wrapperStatus = wd.WrapperStatus
					wrapperError = wd.Err
					if wd.Done {
						return
					}
				}
			}()

			if wrapperStatus != WrapperStatusStopped {
				t.Errorf("expected wrapperStatus == WrapperStatusStopped, got %v", wrapperStatus)
			}

			t.Log("start the process")
			go p.do(chanWrapperData)

			t.Log("wait to ensure the process is running")
			time.Sleep(10 * time.Millisecond)

			if wrapperStatus != WrapperStatusRunning {
				t.Errorf("expected wrapperStatus == WrapperStatusRunning, got %v", wrapperStatus)
			}

			t.Log("wait for the process error log")
			time.Sleep(50 * time.Millisecond)

			if wrapperStatus != WrapperStatusError {
				t.Errorf("expected wrapperStatus == WrapperStatusError, got %v", wrapperStatus)
			}

			t.Log("wait for the process to exit the first time")
			time.Sleep(50 * time.Millisecond)

			if wrapperStatus != tt.want.statusAfterFirstExit {
				t.Errorf("expected wrapperStatus == %v, got %v", tt.want.statusAfterFirstExit, wrapperStatus)
			}

			if tt.args.cancelWhileRunning {
				t.Log("wait for the process to restart")
				time.Sleep(50 * time.Millisecond)

				if wrapperStatus != WrapperStatusRunning {
					t.Errorf("expected wrapperStatus == WrapperStatusRunning, got %v", wrapperStatus)
				}
			}

			t.Log("cancel the context to terminate the process")
			cancel()
			<-done
			if !tt.want.err && wrapperStatus != WrapperStatusStopped {
				t.Errorf("expected wrapperStatus != WrapperStatusStopped, got %v", wrapperStatus)
			}

			if tt.want.err && wrapperStatus != WrapperStatusError {
				t.Errorf("expected wrapperStatus != WrapperStatusError, got %v", wrapperStatus)
			}

			if tt.want.err && wrapperError == nil {
				t.Errorf("expected an error, got %v", wrapperError)
			}

			if !tt.want.err && wrapperError != nil {
				t.Errorf("no error expected, got %v", wrapperError)
			}
		})
	}
}
