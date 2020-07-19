package system

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/gandalfmagic/liveness-wrapper/pkg/logger"
)

var testDirectory = "../../test/system"

func Test_wrapperHandler_canRestart(t *testing.T) {
	type fields struct {
		arg          []string
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

// testing a simple execution of the process, without context cancel.
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
		wantErr           bool
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
				wantErr:           true,
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
				wantErr:           true,
			},
		},
		{
			name: "Simple_run",
			fields: fields{
				arg:          []string{},
				failOnStdErr: false,
				hideStdErr:   false,
				hideStdOut:   false,
				path:         filepath.Join(testDirectory, "test_int_no_err.sh"),
				restart:      WrapperRestartNever,
			},
			want: want{
				statusBeforeStart: WrapperStatusStopped,
				statusAfterStart:  WrapperStatusRunning,
				statusAfterDone:   WrapperStatusStopped,
				wantErr:           false,
			},
		},
		{
			name: "Simple_run_error",
			fields: fields{
				arg:          []string{},
				failOnStdErr: false,
				hideStdErr:   false,
				hideStdOut:   false,
				path:         filepath.Join(testDirectory, "error_10_int_no_err.sh"),
				restart:      WrapperRestartNever,
			},
			want: want{
				statusBeforeStart: WrapperStatusStopped,
				statusAfterStart:  WrapperStatusRunning,
				statusAfterDone:   WrapperStatusError,
				wantErr:           true,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &wrapperHandler{
				arg:          tt.fields.arg,
				failOnStdErr: tt.fields.failOnStdErr,
				hideStdErr:   tt.fields.hideStdErr,
				hideStdOut:   tt.fields.hideStdOut,
				path:         tt.fields.path,
				restart:      tt.fields.restart,
			}

			chanWrapperData := make(chan WrapperData)
			chanWrapperDone := make(chan struct{})
			done := make(chan struct{})

			// start the main process to update wrapperStatus and wrapperError
			// from the internal events of the process
			var mux sync.Mutex
			var wrapperStatus WrapperStatus
			var wrapperError error
			go func() {
				defer close(done)
				for wd := range chanWrapperData {
					mux.Lock()
					wrapperStatus = wd.WrapperStatus
					wrapperError = wd.Err
					if wd.Done {
						mux.Unlock()
						return
					}
					mux.Unlock()
				}
			}()

			mux.Lock()
			if wrapperStatus != tt.want.statusBeforeStart {
				t.Errorf("expected wrapperStatus == %v, got %v", tt.want.statusBeforeStart, wrapperStatus)
			}
			mux.Unlock()

			// prepare a simple context with no cancel
			ctx := context.Background()

			// start the process
			go p.do(ctx, chanWrapperData, chanWrapperDone)

			// wait to ensure the process is running
			time.Sleep(10 * time.Millisecond)

			mux.Lock()
			if wrapperStatus != tt.want.statusAfterStart {
				t.Errorf("after start: expected wrapperStatus == %v, got %v", tt.want.statusAfterStart, wrapperStatus)
			}
			mux.Unlock()

			<-done
			<-chanWrapperDone

			// wait 10ms after the process stopped signal
			time.Sleep(10 * time.Millisecond)
			mux.Lock()
			if tt.want.statusAfterDone != wrapperStatus {
				t.Errorf("after stop: expected wrapperStatus == %v, got %v", tt.want.statusAfterDone, wrapperStatus)
			}
			mux.Unlock()

			mux.Lock()
			if tt.want.wantErr && wrapperError == nil {
				t.Errorf("expected an wantErr, got %v", wrapperError)
			}
			mux.Unlock()

			mux.Lock()
			if !tt.want.wantErr && wrapperError != nil {
				t.Errorf("no wantErr expected, got %v", wrapperError)
			}
			mux.Unlock()
		})
	}
}

func Test_wrapperHandler_do_With_cancel(t *testing.T) {
	type fields struct {
		arg          []string
		failOnStdErr bool
		hideStdErr   bool
		hideStdOut   bool
		path         string
		restart      WrapperRestartMode
		timeout      time.Duration
	}

	type args struct {
		exitImmediately bool
	}

	type want struct {
		statusBeforeStart WrapperStatus
		statusAfterStart  WrapperStatus
		statusAfterCancel WrapperStatus
		statusAfterDone   WrapperStatus
		wantErr           bool
	}

	tests := []struct {
		name   string
		fields fields
		args   args
		want   want
	}{
		{
			name: "Simple_run_exit_ok",
			fields: fields{
				arg:          []string{},
				failOnStdErr: false,
				hideStdErr:   false,
				hideStdOut:   false,
				path:         filepath.Join(testDirectory, "test_int_no_err.sh"),
				restart:      WrapperRestartNever,
				timeout:      6 * time.Second,
			},
			args: args{
				exitImmediately: false,
			},
			want: want{
				statusBeforeStart: WrapperStatusStopped,
				statusAfterStart:  WrapperStatusRunning,
				statusAfterCancel: WrapperStatusRunning,
				statusAfterDone:   WrapperStatusStopped,
				wantErr:           false,
			},
		},
		{
			name: "Simple_run_0s_exit_ok",
			fields: fields{
				arg:          []string{},
				failOnStdErr: false,
				hideStdErr:   false,
				hideStdOut:   false,
				path:         filepath.Join(testDirectory, "test_0s_int_no_err.sh"),
				restart:      WrapperRestartNever,
				timeout:      6 * time.Second,
			},
			args: args{
				exitImmediately: true,
			},
			want: want{
				statusBeforeStart: WrapperStatusStopped,
				statusAfterStart:  WrapperStatusRunning,
				statusAfterCancel: WrapperStatusStopped,
				statusAfterDone:   WrapperStatusStopped,
				wantErr:           false,
			},
		},
		{
			name: "Simple_run_error_exit_ok",
			fields: fields{
				arg:          []string{},
				failOnStdErr: false,
				hideStdErr:   false,
				hideStdOut:   false,
				path:         filepath.Join(testDirectory, "error_10_int_no_err.sh"),
				restart:      WrapperRestartNever,
				timeout:      6 * time.Second,
			},
			args: args{
				exitImmediately: false,
			},
			want: want{
				statusBeforeStart: WrapperStatusStopped,
				statusAfterStart:  WrapperStatusRunning,
				statusAfterCancel: WrapperStatusRunning,
				statusAfterDone:   WrapperStatusStopped,
				wantErr:           false,
			},
		},
		{
			name: "Simple_run_exit_err",
			fields: fields{
				arg:          []string{},
				failOnStdErr: false,
				hideStdErr:   false,
				hideStdOut:   false,
				path:         filepath.Join(testDirectory, "test_int_err.sh"),
				restart:      WrapperRestartNever,
				timeout:      6 * time.Second,
			},
			args: args{
				exitImmediately: false,
			},
			want: want{
				statusBeforeStart: WrapperStatusStopped,
				statusAfterStart:  WrapperStatusRunning,
				statusAfterCancel: WrapperStatusRunning,
				statusAfterDone:   WrapperStatusError,
				wantErr:           true,
			},
		},
		{
			name: "Simple_run_0s_exit_err",
			fields: fields{
				arg:          []string{},
				failOnStdErr: false,
				hideStdErr:   false,
				hideStdOut:   false,
				path:         filepath.Join(testDirectory, "test_0s_int_err.sh"),
				restart:      WrapperRestartNever,
				timeout:      6 * time.Second,
			},
			args: args{
				exitImmediately: true,
			},
			want: want{
				statusBeforeStart: WrapperStatusStopped,
				statusAfterStart:  WrapperStatusRunning,
				statusAfterCancel: WrapperStatusError,
				statusAfterDone:   WrapperStatusError,
				wantErr:           true,
			},
		},
		{
			name: "Simple_run_error_exit_err",
			fields: fields{
				arg:          []string{},
				failOnStdErr: false,
				hideStdErr:   false,
				hideStdOut:   false,
				path:         filepath.Join(testDirectory, "error_10_int_err.sh"),
				restart:      WrapperRestartNever,
				timeout:      6 * time.Second,
			},
			args: args{
				exitImmediately: false,
			},
			want: want{
				statusBeforeStart: WrapperStatusStopped,
				statusAfterStart:  WrapperStatusRunning,
				statusAfterCancel: WrapperStatusRunning,
				statusAfterDone:   WrapperStatusError,
				wantErr:           true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &wrapperHandler{
				arg:          tt.fields.arg,
				failOnStdErr: tt.fields.failOnStdErr,
				hideStdErr:   tt.fields.hideStdErr,
				hideStdOut:   tt.fields.hideStdOut,
				path:         tt.fields.path,
				restart:      tt.fields.restart,
				timeout:      tt.fields.timeout,
			}

			chanWrapperData := make(chan WrapperData)
			chanWrapperDone := make(chan struct{})
			done := make(chan struct{})

			// start the main process to update wrapperStatus and wrapperError
			// from the internal events of the process
			var mux sync.Mutex
			var wrapperStatus WrapperStatus
			var wrapperError error
			go func() {
				defer close(done)
				for wd := range chanWrapperData {
					mux.Lock()
					wrapperStatus = wd.WrapperStatus
					wrapperError = wd.Err
					if wd.Done {
						mux.Unlock()
						return
					}
					mux.Unlock()
				}
			}()

			mux.Lock()
			if wrapperStatus != tt.want.statusBeforeStart {
				t.Errorf("expected wrapperStatus == %v, got %v", tt.want.statusBeforeStart, wrapperStatus)
			}
			mux.Unlock()

			// prepare a context with cancel
			ctx, cancel := context.WithCancel(context.Background())

			// start the process
			go p.do(ctx, chanWrapperData, chanWrapperDone)

			// wait to ensure the process is running
			time.Sleep(10 * time.Millisecond)

			mux.Lock()
			if tt.want.statusAfterStart != wrapperStatus {
				t.Errorf("after start: expected wrapperStatus == %v, got %v", tt.want.statusAfterStart, wrapperStatus)
			}
			mux.Unlock()

			cancel()

			if !tt.args.exitImmediately {
				// wait to 20ms
				time.Sleep(20 * time.Millisecond)

				mux.Lock()
				if tt.want.statusAfterCancel != wrapperStatus {
					t.Errorf("after cancel: expected wrapperStatus == %v, got %v", tt.want.statusAfterCancel, wrapperStatus)
				}
				mux.Unlock()
			}

			<-done
			<-chanWrapperDone

			// wait 10ms after the process stopped signal
			time.Sleep(10 * time.Millisecond)
			mux.Lock()
			if tt.want.statusAfterDone != wrapperStatus {
				t.Errorf("after stop: expected wrapperStatus == %v, got %v", tt.want.statusAfterDone, wrapperStatus)
			}
			mux.Unlock()

			mux.Lock()
			if tt.want.wantErr && wrapperError == nil {
				t.Errorf("expected an wantErr, got %v", wrapperError)
			}
			mux.Unlock()

			mux.Lock()
			if !tt.want.wantErr && wrapperError != nil {
				t.Errorf("no wantErr expected, got %v", wrapperError)
			}
			mux.Unlock()
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
		timeout      time.Duration
	}
	type args struct {
		cancelWhileRunning bool
		checkTimeout       bool
		timeToExit         time.Duration
		timeToRestart      time.Duration
	}
	type want struct {
		statusAfterStart     WrapperStatus
		statusAfterFirstExit WrapperStatus
		statusAfterRestart   WrapperStatus
		statusAfterCancel    WrapperStatus
		statusAfterDone      WrapperStatus
		statusError          int
		wantErr              bool
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
				path:         filepath.Join(testDirectory, "error_10_int_no_err.sh"),
				restart:      WrapperRestartOnError,
				timeout:      6 * time.Second,
			},
			args: args{
				cancelWhileRunning: true,
				timeToExit:         240 * time.Millisecond,
				timeToRestart:      50 * time.Millisecond,
			},
			want: want{
				statusAfterStart:     WrapperStatusRunning,
				statusAfterFirstExit: WrapperStatusError,
				statusAfterRestart:   WrapperStatusRunning,
				statusAfterCancel:    WrapperStatusRunning,
				statusAfterDone:      WrapperStatusStopped,
				wantErr:              false,
			},
		},
		{
			name: "On_error_Cancel_while_IS_running_int_err",
			fields: fields{
				arg:          []string{},
				failOnStdErr: false,
				hideStdErr:   false,
				hideStdOut:   false,
				path:         filepath.Join(testDirectory, "error_10_int_err.sh"),
				restart:      WrapperRestartOnError,
				timeout:      6 * time.Second,
			},
			args: args{
				cancelWhileRunning: true,
				timeToExit:         240 * time.Millisecond,
				timeToRestart:      50 * time.Millisecond,
			},
			want: want{
				statusAfterStart:     WrapperStatusRunning,
				statusAfterFirstExit: WrapperStatusError,
				statusAfterRestart:   WrapperStatusRunning,
				statusAfterCancel:    WrapperStatusRunning,
				statusAfterDone:      WrapperStatusError,
				statusError:          131,
				wantErr:              true,
			},
		},
		{
			name: "On_error_Cancel_while_NOT_running",
			fields: fields{
				arg:          []string{},
				failOnStdErr: false,
				hideStdErr:   false,
				hideStdOut:   false,
				path:         filepath.Join(testDirectory, "error_10_int_no_err.sh"),
				restart:      WrapperRestartOnError,
				timeout:      6 * time.Second,
			},
			args: args{
				cancelWhileRunning: false,
				timeToExit:         240 * time.Millisecond,
				timeToRestart:      50 * time.Millisecond,
			},
			want: want{
				statusAfterStart:     WrapperStatusRunning,
				statusAfterFirstExit: WrapperStatusError,
				statusAfterRestart:   WrapperStatusRunning,
				statusAfterCancel:    WrapperStatusError,
				statusAfterDone:      WrapperStatusError,
				statusError:          10,
				wantErr:              true,
			},
		},
		{
			name: "On_error_Cancel_while_NOT_running_int_err",
			fields: fields{
				arg:          []string{},
				failOnStdErr: false,
				hideStdErr:   false,
				hideStdOut:   false,
				path:         filepath.Join(testDirectory, "error_10_int_err.sh"),
				restart:      WrapperRestartOnError,
				timeout:      6 * time.Second,
			},
			args: args{
				cancelWhileRunning: false,
				timeToExit:         240 * time.Millisecond,
				timeToRestart:      50 * time.Millisecond,
			},
			want: want{
				statusAfterStart:     WrapperStatusRunning,
				statusAfterFirstExit: WrapperStatusError,
				statusAfterRestart:   WrapperStatusRunning,
				statusAfterCancel:    WrapperStatusError,
				statusAfterDone:      WrapperStatusError,
				statusError:          10,
				wantErr:              true,
			},
		},
		{
			name: "On_error_Cancel_while_IS_running_0s",
			fields: fields{
				arg:          []string{},
				failOnStdErr: false,
				hideStdErr:   false,
				hideStdOut:   false,
				path:         filepath.Join(testDirectory, "error_10_0s_int_no_err.sh"),
				restart:      WrapperRestartOnError,
				timeout:      6 * time.Second,
			},
			args: args{
				cancelWhileRunning: true,
				timeToExit:         110 * time.Millisecond,
				timeToRestart:      50 * time.Millisecond,
			},
			want: want{
				statusAfterStart:     WrapperStatusRunning,
				statusAfterFirstExit: WrapperStatusError,
				statusAfterRestart:   WrapperStatusRunning,
				statusAfterCancel:    WrapperStatusRunning,
				statusAfterDone:      WrapperStatusStopped,
				wantErr:              false,
			},
		},
		{
			name: "On_error_Cancel_while_IS_running_0s_int_err",
			fields: fields{
				arg:          []string{},
				failOnStdErr: false,
				hideStdErr:   false,
				hideStdOut:   false,
				path:         filepath.Join(testDirectory, "error_10_0s_int_err.sh"),
				restart:      WrapperRestartOnError,
				timeout:      6 * time.Second,
			},
			args: args{
				cancelWhileRunning: true,
				timeToExit:         110 * time.Millisecond,
				timeToRestart:      50 * time.Millisecond,
			},
			want: want{
				statusAfterStart:     WrapperStatusRunning,
				statusAfterFirstExit: WrapperStatusError,
				statusAfterRestart:   WrapperStatusRunning,
				statusAfterCancel:    WrapperStatusRunning,
				statusAfterDone:      WrapperStatusError,
				statusError:          131,
				wantErr:              true,
			},
		},
		{
			name: "On_error_Cancel_while_NOT_running_0s",
			fields: fields{
				arg:          []string{},
				failOnStdErr: false,
				hideStdErr:   false,
				hideStdOut:   false,
				path:         filepath.Join(testDirectory, "error_10_0s_int_no_err.sh"),
				restart:      WrapperRestartOnError,
				timeout:      6 * time.Second,
			},
			args: args{
				cancelWhileRunning: false,
				timeToExit:         110 * time.Millisecond,
				timeToRestart:      50 * time.Millisecond,
			},
			want: want{
				statusAfterStart:     WrapperStatusRunning,
				statusAfterFirstExit: WrapperStatusError,
				statusAfterRestart:   WrapperStatusRunning,
				statusAfterCancel:    WrapperStatusError,
				statusAfterDone:      WrapperStatusError,
				statusError:          10,
				wantErr:              true,
			},
		},
		{
			name: "On_error_Cancel_while_NOT_running_0s_int_err",
			fields: fields{
				arg:          []string{},
				failOnStdErr: false,
				hideStdErr:   false,
				hideStdOut:   false,
				path:         filepath.Join(testDirectory, "error_10_0s_int_err.sh"),
				restart:      WrapperRestartOnError,
				timeout:      6 * time.Second,
			},
			args: args{
				cancelWhileRunning: false,
				timeToExit:         110 * time.Millisecond,
				timeToRestart:      50 * time.Millisecond,
			},
			want: want{
				statusAfterStart:     WrapperStatusRunning,
				statusAfterFirstExit: WrapperStatusError,
				statusAfterRestart:   WrapperStatusRunning,
				statusAfterCancel:    WrapperStatusError,
				statusAfterDone:      WrapperStatusError,
				statusError:          10,
				wantErr:              true,
			},
		},
		{
			name: "Always_Get_error_Cancel_while_IS_running",
			fields: fields{
				arg:          []string{},
				failOnStdErr: false,
				hideStdErr:   false,
				hideStdOut:   false,
				path:         filepath.Join(testDirectory, "error_10_int_no_err.sh"),
				restart:      WrapperRestartAlways,
				timeout:      6 * time.Second,
			},
			args: args{
				cancelWhileRunning: true,
				timeToExit:         240 * time.Millisecond,
				timeToRestart:      50 * time.Millisecond,
			},
			want: want{
				statusAfterStart:     WrapperStatusRunning,
				statusAfterFirstExit: WrapperStatusError,
				statusAfterRestart:   WrapperStatusRunning,
				statusAfterCancel:    WrapperStatusRunning,
				statusAfterDone:      WrapperStatusStopped,
				wantErr:              false,
			},
		},
		{
			name: "Always_Get_error_Cancel_while_IS_running_int_err",
			fields: fields{
				arg:          []string{},
				failOnStdErr: false,
				hideStdErr:   false,
				hideStdOut:   false,
				path:         filepath.Join(testDirectory, "error_10_int_err.sh"),
				restart:      WrapperRestartAlways,
				timeout:      6 * time.Second,
			},
			args: args{
				cancelWhileRunning: true,
				timeToExit:         240 * time.Millisecond,
				timeToRestart:      50 * time.Millisecond,
			},
			want: want{
				statusAfterStart:     WrapperStatusRunning,
				statusAfterFirstExit: WrapperStatusError,
				statusAfterRestart:   WrapperStatusRunning,
				statusAfterCancel:    WrapperStatusRunning,
				statusAfterDone:      WrapperStatusError,
				statusError:          131,
				wantErr:              true,
			},
		},
		{
			name: "Always_Get_error_Cancel_while_NOT_running",
			fields: fields{
				arg:          []string{},
				failOnStdErr: false,
				hideStdErr:   false,
				hideStdOut:   false,
				path:         filepath.Join(testDirectory, "error_10_int_no_err.sh"),
				restart:      WrapperRestartAlways,
				timeout:      6 * time.Second,
			},
			args: args{
				cancelWhileRunning: false,
				timeToExit:         240 * time.Millisecond,
				timeToRestart:      50 * time.Millisecond,
			},
			want: want{
				statusAfterStart:     WrapperStatusRunning,
				statusAfterFirstExit: WrapperStatusError,
				statusAfterRestart:   WrapperStatusRunning,
				statusAfterCancel:    WrapperStatusError,
				statusAfterDone:      WrapperStatusError,
				statusError:          10,
				wantErr:              true,
			},
		},
		{
			name: "Always_Get_error_Cancel_while_NOT_running_int_err",
			fields: fields{
				arg:          []string{},
				failOnStdErr: false,
				hideStdErr:   false,
				hideStdOut:   false,
				path:         filepath.Join(testDirectory, "error_10_int_err.sh"),
				restart:      WrapperRestartAlways,
				timeout:      6 * time.Second,
			},
			args: args{
				cancelWhileRunning: false,
				timeToExit:         240 * time.Millisecond,
				timeToRestart:      50 * time.Millisecond,
			},
			want: want{
				statusAfterStart:     WrapperStatusRunning,
				statusAfterFirstExit: WrapperStatusError,
				statusAfterRestart:   WrapperStatusRunning,
				statusAfterCancel:    WrapperStatusError,
				statusAfterDone:      WrapperStatusError,
				statusError:          10,
				wantErr:              true,
			},
		},
		{
			name: "Always_Get_error_Cancel_while_IS_running_0s",
			fields: fields{
				arg:          []string{},
				failOnStdErr: false,
				hideStdErr:   false,
				hideStdOut:   false,
				path:         filepath.Join(testDirectory, "error_10_0s_int_no_err.sh"),
				restart:      WrapperRestartAlways,
				timeout:      6 * time.Second,
			},
			args: args{
				cancelWhileRunning: true,
				timeToExit:         110 * time.Millisecond,
				timeToRestart:      50 * time.Millisecond,
			},
			want: want{
				statusAfterStart:     WrapperStatusRunning,
				statusAfterFirstExit: WrapperStatusError,
				statusAfterRestart:   WrapperStatusRunning,
				statusAfterCancel:    WrapperStatusRunning,
				statusAfterDone:      WrapperStatusStopped,
				wantErr:              false,
			},
		},
		{
			name: "Always_Get_error_Cancel_while_IS_running_0s_int_err",
			fields: fields{
				arg:          []string{},
				failOnStdErr: false,
				hideStdErr:   false,
				hideStdOut:   false,
				path:         filepath.Join(testDirectory, "error_10_0s_int_err.sh"),
				restart:      WrapperRestartAlways,
				timeout:      6 * time.Second,
			},
			args: args{
				cancelWhileRunning: true,
				timeToExit:         110 * time.Millisecond,
				timeToRestart:      50 * time.Millisecond,
			},
			want: want{
				statusAfterStart:     WrapperStatusRunning,
				statusAfterFirstExit: WrapperStatusError,
				statusAfterRestart:   WrapperStatusRunning,
				statusAfterCancel:    WrapperStatusRunning,
				statusAfterDone:      WrapperStatusError,
				statusError:          131,
				wantErr:              true,
			},
		},
		{
			name: "Always_Get_error_Cancel_while_NOT_running_0s",
			fields: fields{
				arg:          []string{},
				failOnStdErr: false,
				hideStdErr:   false,
				hideStdOut:   false,
				path:         filepath.Join(testDirectory, "error_10_0s_int_no_err.sh"),
				restart:      WrapperRestartAlways,
				timeout:      6 * time.Second,
			},
			args: args{
				cancelWhileRunning: false,
				timeToExit:         110 * time.Millisecond,
				timeToRestart:      50 * time.Millisecond,
			},
			want: want{
				statusAfterStart:     WrapperStatusRunning,
				statusAfterFirstExit: WrapperStatusError,
				statusAfterRestart:   WrapperStatusRunning,
				statusAfterCancel:    WrapperStatusError,
				statusAfterDone:      WrapperStatusError,
				statusError:          10,
				wantErr:              true,
			},
		},
		{
			name: "Always_Get_error_Cancel_while_NOT_running_0s_int_err",
			fields: fields{
				arg:          []string{},
				failOnStdErr: false,
				hideStdErr:   false,
				hideStdOut:   false,
				path:         filepath.Join(testDirectory, "error_10_0s_int_err.sh"),
				restart:      WrapperRestartAlways,
				timeout:      6 * time.Second,
			},
			args: args{
				cancelWhileRunning: false,
				timeToExit:         110 * time.Millisecond,
				timeToRestart:      50 * time.Millisecond,
			},
			want: want{
				statusAfterStart:     WrapperStatusRunning,
				statusAfterFirstExit: WrapperStatusError,
				statusAfterRestart:   WrapperStatusRunning,
				statusAfterCancel:    WrapperStatusError,
				statusAfterDone:      WrapperStatusError,
				statusError:          10,
				wantErr:              true,
			},
		},
		{
			name: "Always_Exit_without_error_Cancel_while_IS_running",
			fields: fields{
				arg:          []string{},
				failOnStdErr: false,
				hideStdErr:   false,
				hideStdOut:   false,
				path:         filepath.Join(testDirectory, "test_int_no_err.sh"),
				restart:      WrapperRestartAlways,
				timeout:      6 * time.Second,
			},
			args: args{
				cancelWhileRunning: true,
				timeToExit:         240 * time.Millisecond,
				timeToRestart:      50 * time.Millisecond,
			},
			want: want{
				statusAfterStart:     WrapperStatusRunning,
				statusAfterFirstExit: WrapperStatusStopped,
				statusAfterRestart:   WrapperStatusRunning,
				statusAfterCancel:    WrapperStatusRunning,
				statusAfterDone:      WrapperStatusStopped,
				wantErr:              false,
			},
		},
		{
			name: "Always_Exit_without_error_Cancel_while_IS_running_int_err",
			fields: fields{
				arg:          []string{},
				failOnStdErr: false,
				hideStdErr:   false,
				hideStdOut:   false,
				path:         filepath.Join(testDirectory, "test_int_err.sh"),
				restart:      WrapperRestartAlways,
				timeout:      6 * time.Second,
			},
			args: args{
				cancelWhileRunning: true,
				timeToExit:         240 * time.Millisecond,
				timeToRestart:      50 * time.Millisecond,
			},
			want: want{
				statusAfterStart:     WrapperStatusRunning,
				statusAfterFirstExit: WrapperStatusStopped,
				statusAfterRestart:   WrapperStatusRunning,
				statusAfterCancel:    WrapperStatusRunning,
				statusAfterDone:      WrapperStatusError,
				statusError:          131,
				wantErr:              true,
			},
		},
		{
			name: "Always_Exit_without_error_Cancel_while_NOT_running",
			fields: fields{
				arg:          []string{},
				failOnStdErr: false,
				hideStdErr:   false,
				hideStdOut:   false,
				path:         filepath.Join(testDirectory, "test_int_no_err.sh"),
				restart:      WrapperRestartAlways,
				timeout:      6 * time.Second,
			},
			args: args{
				cancelWhileRunning: false,
				timeToExit:         240 * time.Millisecond,
				timeToRestart:      50 * time.Millisecond,
			},
			want: want{
				statusAfterStart:     WrapperStatusRunning,
				statusAfterFirstExit: WrapperStatusStopped,
				statusAfterRestart:   WrapperStatusStopped,
				statusAfterCancel:    WrapperStatusStopped,
				statusAfterDone:      WrapperStatusStopped,
				wantErr:              false,
			},
		},
		{
			name: "Always_Exit_without_error_Cancel_while_NOT_running_int_err",
			fields: fields{
				arg:          []string{},
				failOnStdErr: false,
				hideStdErr:   false,
				hideStdOut:   false,
				path:         filepath.Join(testDirectory, "test_int_err.sh"),
				restart:      WrapperRestartAlways,
				timeout:      6 * time.Second,
			},
			args: args{
				cancelWhileRunning: false,
				timeToExit:         240 * time.Millisecond,
				timeToRestart:      50 * time.Millisecond,
			},
			want: want{
				statusAfterStart:     WrapperStatusRunning,
				statusAfterFirstExit: WrapperStatusStopped,
				statusAfterRestart:   WrapperStatusStopped,
				statusAfterCancel:    WrapperStatusStopped,
				statusAfterDone:      WrapperStatusStopped,
				wantErr:              false,
			},
		},
		{
			name: "Always_Exit_without_error_Cancel_while_IS_running_0s",
			fields: fields{
				arg:          []string{},
				failOnStdErr: false,
				hideStdErr:   false,
				hideStdOut:   false,
				path:         filepath.Join(testDirectory, "test_0s_int_no_err.sh"),
				restart:      WrapperRestartAlways,
				timeout:      6 * time.Second,
			},
			args: args{
				cancelWhileRunning: true,
				timeToExit:         110 * time.Millisecond,
				timeToRestart:      50 * time.Millisecond,
			},
			want: want{
				statusAfterStart:     WrapperStatusRunning,
				statusAfterFirstExit: WrapperStatusStopped,
				statusAfterRestart:   WrapperStatusRunning,
				statusAfterCancel:    WrapperStatusRunning,
				statusAfterDone:      WrapperStatusStopped,
				wantErr:              false,
			},
		},
		{
			name: "Always_Exit_without_error_Cancel_while_IS_running_0s_int_err",
			fields: fields{
				arg:          []string{},
				failOnStdErr: false,
				hideStdErr:   false,
				hideStdOut:   false,
				path:         filepath.Join(testDirectory, "test_0s_int_err.sh"),
				restart:      WrapperRestartAlways,
				timeout:      6 * time.Second,
			},
			args: args{
				cancelWhileRunning: true,
				timeToExit:         110 * time.Millisecond,
				timeToRestart:      50 * time.Millisecond,
			},
			want: want{
				statusAfterStart:     WrapperStatusRunning,
				statusAfterFirstExit: WrapperStatusStopped,
				statusAfterRestart:   WrapperStatusRunning,
				statusAfterCancel:    WrapperStatusRunning,
				statusAfterDone:      WrapperStatusError,
				statusError:          131,
				wantErr:              true,
			},
		},
		{
			name: "Always_Exit_without_error_Cancel_while_NOT_running_0s",
			fields: fields{
				arg:          []string{},
				failOnStdErr: false,
				hideStdErr:   false,
				hideStdOut:   false,
				path:         filepath.Join(testDirectory, "test_0s_int_no_err.sh"),
				restart:      WrapperRestartAlways,
				timeout:      6 * time.Second,
			},
			args: args{
				cancelWhileRunning: false,
				timeToExit:         110 * time.Millisecond,
				timeToRestart:      50 * time.Millisecond,
			},
			want: want{
				statusAfterStart:     WrapperStatusRunning,
				statusAfterFirstExit: WrapperStatusStopped,
				statusAfterRestart:   WrapperStatusStopped,
				statusAfterCancel:    WrapperStatusStopped,
				statusAfterDone:      WrapperStatusStopped,
				wantErr:              false,
			},
		},
		{
			name: "Always_Exit_without_error_Cancel_while_NOT_running_0s_int_err",
			fields: fields{
				arg:          []string{},
				failOnStdErr: false,
				hideStdErr:   false,
				hideStdOut:   false,
				path:         filepath.Join(testDirectory, "test_0s_int_err.sh"),
				restart:      WrapperRestartAlways,
				timeout:      6 * time.Second,
			},
			args: args{
				cancelWhileRunning: false,
				timeToExit:         110 * time.Millisecond,
				timeToRestart:      50 * time.Millisecond,
			},
			want: want{
				statusAfterStart:     WrapperStatusRunning,
				statusAfterFirstExit: WrapperStatusStopped,
				statusAfterRestart:   WrapperStatusStopped,
				statusAfterCancel:    WrapperStatusStopped,
				statusAfterDone:      WrapperStatusStopped,
				wantErr:              false,
			},
		},
		{
			name: "Always_Exit_without_error_Cancel_while_running_timeout",
			fields: fields{
				arg:          []string{},
				failOnStdErr: false,
				hideStdErr:   false,
				hideStdOut:   false,
				path:         filepath.Join(testDirectory, "test_int_no_err.sh"),
				restart:      WrapperRestartAlways,
				timeout:      10 * time.Millisecond,
			},
			args: args{
				cancelWhileRunning: true,
				checkTimeout:       true,
				timeToExit:         240 * time.Millisecond,
				timeToRestart:      50 * time.Millisecond,
			},
			want: want{
				statusAfterStart:     WrapperStatusRunning,
				statusAfterFirstExit: WrapperStatusStopped,
				statusAfterRestart:   WrapperStatusRunning,
				statusAfterCancel:    WrapperStatusRunning,
				statusAfterDone:      WrapperStatusError,
				statusError:          255,
				wantErr:              true,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			p := &wrapperHandler{
				arg:             tt.fields.arg,
				failOnStdErr:    tt.fields.failOnStdErr,
				hideStdErr:      tt.fields.hideStdErr,
				hideStdOut:      tt.fields.hideStdOut,
				path:            tt.fields.path,
				restart:         tt.fields.restart,
				restartInterval: 50 * time.Millisecond,
				timeout:         tt.fields.timeout,
			}

			chanWrapperData := make(chan WrapperData)
			chanWrapperDone := make(chan struct{})
			done := make(chan struct{})

			// start the main process to update wrapperStatus and wrapperError
			// from the internal events of the process
			var mux sync.Mutex
			var wrapperStatus WrapperStatus
			var wrapperError error
			go func() {
				defer close(done)
				for wd := range chanWrapperData {
					mux.Lock()
					wrapperStatus = wd.WrapperStatus
					wrapperError = wd.Err
					if wd.Done {
						mux.Unlock()
						return
					}
					mux.Unlock()
				}
			}()

			mux.Lock()
			if wrapperStatus != WrapperStatusStopped {
				t.Errorf("before start: expected wrapperStatus == WrapperStatusStopped, got %v", wrapperStatus)
			}
			mux.Unlock()

			// create the context
			ctx, cancel := context.WithCancel(context.Background())

			// start the process
			go p.do(ctx, chanWrapperData, chanWrapperDone)

			// wait to ensure the process is running
			time.Sleep(10 * time.Millisecond)

			mux.Lock()
			if wrapperStatus != tt.want.statusAfterStart {
				t.Errorf("after start: expected wrapperStatus == %v, got %v", tt.want.statusAfterStart, wrapperStatus)
			}
			mux.Unlock()

			// wait for the process exit for the first time
			time.Sleep(tt.args.timeToExit)

			mux.Lock()
			if wrapperStatus != tt.want.statusAfterFirstExit {
				t.Errorf("after first exit: expected wrapperStatus == %v, got %v", tt.want.statusAfterFirstExit, wrapperStatus)
			}
			mux.Unlock()

			if tt.args.cancelWhileRunning {
				// wait for the process to restart
				time.Sleep(tt.args.timeToRestart)

				mux.Lock()
				if wrapperStatus != tt.want.statusAfterRestart {
					t.Errorf("after restart: expected wrapperStatus == %v, got %v", tt.want.statusAfterRestart, wrapperStatus)
				}
				mux.Unlock()
			}

			// cancel the context to terminate the process
			cancel()

			// wait 0.5ms
			time.Sleep(500 * time.Microsecond)

			mux.Lock()
			if wrapperStatus != tt.want.statusAfterCancel {
				t.Errorf("after cancel: expected wrapperStatus == %v, got %v", tt.want.statusAfterCancel, wrapperStatus)
			}
			mux.Unlock()

			var timeoutStart time.Time
			if tt.args.checkTimeout {
				time.Sleep(1 * time.Millisecond) // GitHub hack, not needed
				timeoutStart = time.Now()
			}

			<-done
			<-chanWrapperDone

			var timeoutDuration time.Duration
			if tt.args.checkTimeout {
				timeoutDuration = time.Since(timeoutStart)

				if timeoutDuration > tt.fields.timeout {
					t.Errorf("after timeout: expected timeout == %v, got %v", tt.fields.timeout, timeoutDuration)
				}
			}

			// wait 1ms
			time.Sleep(1 * time.Millisecond)

			mux.Lock()
			if wrapperStatus != tt.want.statusAfterDone {
				t.Errorf("after done: expected wrapperStatus == %v, got %v", tt.want.statusAfterDone, wrapperStatus)
			}
			mux.Unlock()

			mux.Lock()
			if tt.want.wantErr && wrapperError == nil {
				t.Errorf("after done: expected an error, got %v", wrapperError)
			}
			mux.Unlock()

			mux.Lock()
			if !tt.want.wantErr && wrapperError != nil {
				t.Errorf("after done: no error expected, got %v", wrapperError)
			}
			mux.Unlock()

			mux.Lock()
			if exitStatusError, ok := wrapperError.(ProcessExitStatusError); tt.want.wantErr && ok {
				if exitStatusError.ExitStatus() != tt.want.statusError {
					t.Errorf("after done: expected exit status == %v, got %v", tt.want.statusError, exitStatusError.ExitStatus())
				}
			}
			mux.Unlock()
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
		timeout      time.Duration
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
				timeout:      6 * time.Second,
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
				timeout:      6 * time.Second,
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
				timeout:      6 * time.Second,
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
				timeout:      6 * time.Second,
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
				timeout:      6 * time.Second,
			},
			args: args{
				cancelWhileRunning: true,
			},
			want: want{
				statusAfterFirstExit: WrapperStatusStopped,
				err:                  false,
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
				timeout:      6 * time.Second,
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
				failOnStdErr:    tt.fields.failOnStdErr,
				hideStdErr:      tt.fields.hideStdErr,
				hideStdOut:      tt.fields.hideStdOut,
				path:            tt.fields.path,
				restart:         tt.fields.restart,
				restartInterval: 50 * time.Millisecond,
				timeout:         tt.fields.timeout,
			}

			chanWrapperData := make(chan WrapperData)
			chanWrapperDone := make(chan struct{})
			done := make(chan struct{})

			// start the main process to update wrapperStatus and wrapperError
			// from the internal events of the process
			var mux sync.Mutex
			var wrapperStatus WrapperStatus
			var wrapperError error
			go func() {
				defer close(done)
				for wd := range chanWrapperData {
					mux.Lock()
					wrapperStatus = wd.WrapperStatus
					wrapperError = wd.Err
					if wd.Done {
						mux.Unlock()
						return
					}
					mux.Unlock()
				}
			}()

			mux.Lock()
			if wrapperStatus != WrapperStatusStopped {
				t.Errorf("expected wrapperStatus == WrapperStatusStopped, got %v", wrapperStatus)
			}
			mux.Unlock()

			// start the process
			go p.do(ctx, chanWrapperData, chanWrapperDone)

			// wait to ensure the process is running
			time.Sleep(10 * time.Millisecond)

			mux.Lock()
			if wrapperStatus != WrapperStatusRunning {
				t.Errorf("expected wrapperStatus == WrapperStatusRunning, got %v", wrapperStatus)
			}
			mux.Unlock()

			// wait for the process wantErr log
			time.Sleep(110 * time.Millisecond)

			mux.Lock()
			if wrapperStatus != WrapperStatusError {
				t.Errorf("expected wrapperStatus == WrapperStatusError, got %v", wrapperStatus)
			}
			mux.Unlock()

			// wait for the process to exit the first time
			time.Sleep(100 * time.Millisecond) // Hack for Github, should be 100

			mux.Lock()
			if wrapperStatus != tt.want.statusAfterFirstExit {
				t.Errorf("expected wrapperStatus == %v, got %v", tt.want.statusAfterFirstExit, wrapperStatus)
			}
			mux.Unlock()

			if tt.args.cancelWhileRunning {
				// wait for the process to restart
				time.Sleep(60 * time.Millisecond)

				mux.Lock()
				if wrapperStatus != WrapperStatusRunning {
					t.Errorf("expected wrapperStatus == WrapperStatusRunning, got %v", wrapperStatus)
				}
				mux.Unlock()
			}

			// cancel the context to terminate the process
			cancel()

			<-done
			<-chanWrapperDone

			mux.Lock()
			if !tt.want.err && wrapperStatus != WrapperStatusStopped {
				t.Errorf("expected wrapperStatus != WrapperStatusStopped, got %v", wrapperStatus)
			}
			mux.Unlock()

			mux.Lock()
			if tt.want.err && wrapperStatus != WrapperStatusError {
				t.Errorf("expected wrapperStatus != WrapperStatusError, got %v", wrapperStatus)
			}
			mux.Unlock()

			mux.Lock()
			if tt.want.err && wrapperError == nil {
				t.Errorf("expected an wantErr, got %v", wrapperError)
			}
			mux.Unlock()

			mux.Lock()
			if !tt.want.err && wrapperError != nil {
				t.Errorf("no wantErr expected, got %v", wrapperError)
			}
			mux.Unlock()
		})
	}
}
