package system

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/gandalfmagic/liveness-wrapper/pkg/logger"
	"github.com/gandalfmagic/liveness-wrapper/pkg/testconsole"
)

type testProcess struct {
	mux           sync.Mutex
	wrapperStatus WrapperStatus
	wrapperError  error
}

func (p *testProcess) Start(chanWrapperData chan WrapperData) <-chan struct{} {
	done := make(chan struct{})

	go p.do(chanWrapperData, done)

	return done
}

func (p *testProcess) do(chanWrapperData chan WrapperData, done chan struct{}) {
	defer close(done)
	for wd := range chanWrapperData {
		p.mux.Lock()
		p.wrapperStatus = wd.WrapperStatus
		p.wrapperError = wd.Err
		if wd.Done {
			p.mux.Unlock()
			return
		}
		p.mux.Unlock()
	}
}

func (p *testProcess) WrapperStatus() WrapperStatus {
	p.mux.Lock()
	defer p.mux.Unlock()

	return p.wrapperStatus
}

func (p *testProcess) WrapperError() error {
	p.mux.Lock()
	defer p.mux.Unlock()

	return p.wrapperError
}

func (p *testProcess) AssertStatus(value WrapperStatus) error {
	p.mux.Lock()
	got := p.wrapperStatus
	p.mux.Unlock()

	if got != value {
		return fmt.Errorf("expected wrapperStatus == %v, got %v", value, got)
	}

	return nil
}

func (p *testProcess) AssertStatusChange(value WrapperStatus, timeout time.Duration) error {
	start := time.Now()

	for {
		time.Sleep(10 * time.Millisecond)

		p.mux.Lock()
		newValue := p.wrapperStatus
		p.mux.Unlock()

		if newValue == value {
			return nil
		}

		if time.Now().After(start.Add(timeout)) {
			return fmt.Errorf("timeout, expected wrapperStatus value %v, got %v", value, newValue)
		}
	}
}

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
				restartMode:  tt.fields.restart,
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

	type waitFor struct {
		afterStart string
	}

	type want struct {
		statusBeforeStart WrapperStatus
		statusAfterStart  WrapperStatus
		statusAfterDone   WrapperStatus
		wantErr           bool
	}

	tests := []struct {
		name    string
		fields  fields
		waitFor waitFor
		want    want
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
			waitFor: waitFor{
				afterStart: "no such file or directory",
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
			waitFor: waitFor{
				afterStart: "wrapped log: 10ms",
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
			waitFor: waitFor{
				afterStart: "wrapped log: 10ms",
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
			console := testconsole.NewTestConsole()
			logger.New(console, "", "INFO")

			p := &wrapperHandler{
				arg:          tt.fields.arg,
				failOnStdErr: tt.fields.failOnStdErr,
				hideStdErr:   tt.fields.hideStdErr,
				hideStdOut:   tt.fields.hideStdOut,
				path:         tt.fields.path,
				restartMode:  tt.fields.restart,
			}

			chanWrapperData := make(chan WrapperData)
			chanWrapperDone := make(chan struct{})

			var tp testProcess
			done := tp.Start(chanWrapperData)

			if err := tp.AssertStatus(tt.want.statusBeforeStart); err != nil {
				t.Errorf("before start: %s", err)
			}

			// prepare a simple context with no cancel
			ctx := context.Background()

			// start the process
			go p.do(ctx, chanWrapperData, chanWrapperDone)

			ch := console.WaitForText(tt.waitFor.afterStart, 1*time.Second)
			if err := <-ch; err != nil {
				t.Fatal(err)
			}

			if err := tp.AssertStatus(tt.want.statusAfterStart); err != nil {
				t.Errorf("after start: %s", err)
			}

			<-done
			<-chanWrapperDone

			if err := tp.AssertStatus(tt.want.statusAfterDone); err != nil {
				t.Errorf("after done: %s", err)
			}

			wrapperError := tp.WrapperError()
			if tt.want.wantErr && wrapperError == nil {
				t.Errorf("expected an wantErr, got %v", wrapperError)
			}

			if !tt.want.wantErr && wrapperError != nil {
				t.Errorf("no wantErr expected, got %v", wrapperError)
			}
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

	type waitFor struct {
		afterStart  string
		afterCancel string
	}

	type want struct {
		statusBeforeStart WrapperStatus
		statusAfterStart  WrapperStatus
		statusAfterCancel WrapperStatus
		statusAfterDone   WrapperStatus
		wantErr           bool
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		waitFor waitFor
		want    want
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
			waitFor: waitFor{
				afterStart:  "wrapped log: 10ms",
				afterCancel: "wrapped log: EXIT 100ms",
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
			waitFor: waitFor{
				afterStart: "wrapped log: 10ms",
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
			waitFor: waitFor{
				afterStart:  "wrapped log: 10ms",
				afterCancel: "wrapped log: EXIT 100ms",
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
			waitFor: waitFor{
				afterStart:  "wrapped log: 10ms",
				afterCancel: "wrapped log: EXIT 100ms",
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
			waitFor: waitFor{
				afterStart: "wrapped log: 10ms",
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
			waitFor: waitFor{
				afterStart:  "wrapped log: 10ms",
				afterCancel: "wrapped log: EXIT 100ms",
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
			console := testconsole.NewTestConsole()
			logger.New(console, "", "INFO")

			p := &wrapperHandler{
				arg:          tt.fields.arg,
				failOnStdErr: tt.fields.failOnStdErr,
				hideStdErr:   tt.fields.hideStdErr,
				hideStdOut:   tt.fields.hideStdOut,
				path:         tt.fields.path,
				restartMode:  tt.fields.restart,
				timeout:      tt.fields.timeout,
			}

			chanWrapperData := make(chan WrapperData)
			chanWrapperDone := make(chan struct{})

			var tp testProcess
			done := tp.Start(chanWrapperData)

			if err := tp.AssertStatus(tt.want.statusBeforeStart); err != nil {
				t.Errorf("before start: %s", err)
			}

			// prepare a context with cancel
			ctx, cancel := context.WithCancel(context.Background())

			// start the process
			go p.do(ctx, chanWrapperData, chanWrapperDone)

			// wait to ensure the process is running
			ch := console.WaitForText(tt.waitFor.afterStart, 1*time.Second)
			if err := <-ch; err != nil {
				t.Fatal(err)
			}

			if err := tp.AssertStatus(tt.want.statusAfterStart); err != nil {
				t.Errorf("after start: %s", err)
			}

			cancel()

			if !tt.args.exitImmediately {
				// wait to ensure the process is running
				ch := console.WaitForText(tt.waitFor.afterCancel, 1*time.Second)
				if err := <-ch; err != nil {
					t.Fatal(err)
				}

				if err := tp.AssertStatus(tt.want.statusAfterCancel); err != nil {
					t.Errorf("after cancel: %s", err)
				}
			}

			<-done
			<-chanWrapperDone

			if err := tp.AssertStatusChange(tt.want.statusAfterDone, 5*time.Millisecond); err != nil {
				t.Errorf("after done: %s", err)
			}

			wrapperError := tp.WrapperError()
			if tt.want.wantErr && wrapperError == nil {
				t.Errorf("expected an wantErr, got %v", wrapperError)
			}

			if !tt.want.wantErr && wrapperError != nil {
				t.Errorf("no wantErr expected, got %v", wrapperError)
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
		timeout      time.Duration
	}

	type args struct {
		cancelWhileRunning bool
		checkTimeout       bool
	}

	type waitFor struct {
		afterStart     string
		afterFirstExit string
		afterRestart   string
		afterCancel    string
	}

	type want struct {
		statusAfterStart     WrapperStatus
		statusAfterFirstExit WrapperStatus
		statusAfterRestart   WrapperStatus
		statusAfterDone      WrapperStatus
		statusError          int
		wantErr              bool
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		waitFor waitFor
		want    want
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
			},
			waitFor: waitFor{
				afterStart:     "wrapped log: 10ms",
				afterFirstExit: "wrapped process exited with status: 10",
				afterRestart:   "wrapped log: 10ms",
				afterCancel:    "wrapped log: EXIT 10ms",
			},
			want: want{
				statusAfterStart:     WrapperStatusRunning,
				statusAfterFirstExit: WrapperStatusError,
				statusAfterRestart:   WrapperStatusRunning,
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
			},
			waitFor: waitFor{
				afterStart:     "wrapped log: 10ms",
				afterFirstExit: "wrapped process exited with status: 10",
				afterRestart:   "wrapped log: 10ms",
				afterCancel:    "wrapped log: EXIT 10ms",
			},
			want: want{
				statusAfterStart:     WrapperStatusRunning,
				statusAfterFirstExit: WrapperStatusError,
				statusAfterRestart:   WrapperStatusRunning,
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
			},
			waitFor: waitFor{
				afterStart:     "wrapped log: 10ms",
				afterFirstExit: "wrapped process exited with status: 10",
				afterRestart:   "",
				afterCancel:    "",
			},
			want: want{
				statusAfterStart:     WrapperStatusRunning,
				statusAfterFirstExit: WrapperStatusError,
				statusAfterRestart:   WrapperStatusRunning,
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
			},
			waitFor: waitFor{
				afterStart:     "wrapped log: 10ms",
				afterFirstExit: "wrapped process exited with status: 10",
				afterRestart:   "",
				afterCancel:    "",
			},
			want: want{
				statusAfterStart:     WrapperStatusRunning,
				statusAfterFirstExit: WrapperStatusError,
				statusAfterRestart:   WrapperStatusRunning,
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
			},
			waitFor: waitFor{
				afterStart:     "wrapped log: 10ms",
				afterFirstExit: "wrapped process exited with status",
				afterRestart:   "wrapped log: 10ms",
				afterCancel:    "wrapped log: TERM SIGNAL",
			},
			want: want{
				statusAfterStart:     WrapperStatusRunning,
				statusAfterFirstExit: WrapperStatusError,
				statusAfterRestart:   WrapperStatusRunning,
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
			},
			waitFor: waitFor{
				afterStart:     "wrapped log: 10ms",
				afterFirstExit: "wrapped process exited with status: 10",
				afterRestart:   "wrapped log: 10ms",
				afterCancel:    "wrapped log: TERM SIGNAL",
			},
			want: want{
				statusAfterStart:     WrapperStatusRunning,
				statusAfterFirstExit: WrapperStatusError,
				statusAfterRestart:   WrapperStatusRunning,
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
			},
			waitFor: waitFor{
				afterStart:     "wrapped log: 10ms",
				afterFirstExit: "wrapped process exited with status",
				afterRestart:   "",
				afterCancel:    "",
			},
			want: want{
				statusAfterStart:     WrapperStatusRunning,
				statusAfterFirstExit: WrapperStatusError,
				statusAfterRestart:   WrapperStatusRunning,
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
			},
			waitFor: waitFor{
				afterStart:     "wrapped log: 10ms",
				afterFirstExit: "wrapped process exited with status",
				afterRestart:   "",
				afterCancel:    "",
			},
			want: want{
				statusAfterStart:     WrapperStatusRunning,
				statusAfterFirstExit: WrapperStatusError,
				statusAfterRestart:   WrapperStatusRunning,
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
			},
			waitFor: waitFor{
				afterStart:     "wrapped log: 10ms",
				afterFirstExit: "wrapped process exited with status",
				afterRestart:   "wrapped log: 10ms",
				afterCancel:    "wrapped log: EXIT 10ms",
			},
			want: want{
				statusAfterStart:     WrapperStatusRunning,
				statusAfterFirstExit: WrapperStatusError,
				statusAfterRestart:   WrapperStatusRunning,
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
			},
			waitFor: waitFor{
				afterStart:     "wrapped log: 10ms",
				afterFirstExit: "wrapped process exited with status",
				afterRestart:   "wrapped log: 10ms",
				afterCancel:    "wrapped log: EXIT 10ms",
			},
			want: want{
				statusAfterStart:     WrapperStatusRunning,
				statusAfterFirstExit: WrapperStatusError,
				statusAfterRestart:   WrapperStatusRunning,
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
			},
			waitFor: waitFor{
				afterStart:     "wrapped log: 10ms",
				afterFirstExit: "wrapped process exited with status",
				afterRestart:   "",
				afterCancel:    "",
			},
			want: want{
				statusAfterStart:     WrapperStatusRunning,
				statusAfterFirstExit: WrapperStatusError,
				statusAfterRestart:   WrapperStatusRunning,
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
			},
			waitFor: waitFor{
				afterStart:     "wrapped log: 10ms",
				afterFirstExit: "wrapped process exited with status",
				afterRestart:   "",
				afterCancel:    "",
			},
			want: want{
				statusAfterStart:     WrapperStatusRunning,
				statusAfterFirstExit: WrapperStatusError,
				statusAfterRestart:   WrapperStatusRunning,
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
			},
			waitFor: waitFor{
				afterStart:     "wrapped log: 10ms",
				afterFirstExit: "wrapped process exited with status",
				afterRestart:   "wrapped log: 10ms",
				afterCancel:    "wrapped log: TERM SIGNAL",
			},
			want: want{
				statusAfterStart:     WrapperStatusRunning,
				statusAfterFirstExit: WrapperStatusError,
				statusAfterRestart:   WrapperStatusRunning,
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
			},
			waitFor: waitFor{
				afterStart:     "wrapped log: 10ms",
				afterFirstExit: "wrapped process exited with status",
				afterRestart:   "wrapped log: 10ms",
				afterCancel:    "wrapped log: TERM SIGNAL",
			},
			want: want{
				statusAfterStart:     WrapperStatusRunning,
				statusAfterFirstExit: WrapperStatusError,
				statusAfterRestart:   WrapperStatusRunning,
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
			},
			waitFor: waitFor{
				afterStart:     "wrapped log: 10ms",
				afterFirstExit: "wrapped process exited with status",
				afterRestart:   "",
				afterCancel:    "",
			},
			want: want{
				statusAfterStart:     WrapperStatusRunning,
				statusAfterFirstExit: WrapperStatusError,
				statusAfterRestart:   WrapperStatusRunning,
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
			},
			waitFor: waitFor{
				afterStart:     "wrapped log: 10ms",
				afterFirstExit: "wrapped process exited with status",
				afterRestart:   "",
				afterCancel:    "",
			},
			want: want{
				statusAfterStart:     WrapperStatusRunning,
				statusAfterFirstExit: WrapperStatusError,
				statusAfterRestart:   WrapperStatusRunning,
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
			},
			waitFor: waitFor{
				afterStart:     "wrapped log: 10ms",
				afterFirstExit: "wrapped log: EXIT 100ms",
				afterRestart:   "wrapped log: 10ms",
				afterCancel:    "wrapped log: EXIT 10ms",
			},
			want: want{
				statusAfterStart:     WrapperStatusRunning,
				statusAfterFirstExit: WrapperStatusStopped,
				statusAfterRestart:   WrapperStatusRunning,
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
			},
			waitFor: waitFor{
				afterStart:     "wrapped log: 10ms",
				afterFirstExit: "wrapped log: EXIT 100ms",
				afterRestart:   "wrapped log: 10ms",
				afterCancel:    "wrapped log: EXIT 10ms",
			},
			want: want{
				statusAfterStart:     WrapperStatusRunning,
				statusAfterFirstExit: WrapperStatusStopped,
				statusAfterRestart:   WrapperStatusRunning,
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
			},
			waitFor: waitFor{
				afterStart:     "wrapped log: 10ms",
				afterFirstExit: "wrapped log: EXIT 100ms",
				afterRestart:   "",
				afterCancel:    "",
			},
			want: want{
				statusAfterStart:     WrapperStatusRunning,
				statusAfterFirstExit: WrapperStatusStopped,
				statusAfterRestart:   WrapperStatusStopped,
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
			},
			waitFor: waitFor{
				afterStart:     "wrapped log: 10ms",
				afterFirstExit: "wrapped log: EXIT 100ms",
				afterRestart:   "",
				afterCancel:    "",
			},
			want: want{
				statusAfterStart:     WrapperStatusRunning,
				statusAfterFirstExit: WrapperStatusStopped,
				statusAfterRestart:   WrapperStatusStopped,
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
			},
			waitFor: waitFor{
				afterStart:     "wrapped log: 10ms",
				afterFirstExit: "wrapped log: 100ms",
				afterRestart:   "wrapped log: 10ms",
				afterCancel:    "wrapped log: TERM SIGNAL",
			},
			want: want{
				statusAfterStart:     WrapperStatusRunning,
				statusAfterFirstExit: WrapperStatusStopped,
				statusAfterRestart:   WrapperStatusRunning,
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
			},
			waitFor: waitFor{
				afterStart:     "wrapped log: 10ms",
				afterFirstExit: "wrapped log: 100ms",
				afterRestart:   "wrapped log: 10ms",
				afterCancel:    "wrapped log: TERM SIGNAL",
			},
			want: want{
				statusAfterStart:     WrapperStatusRunning,
				statusAfterFirstExit: WrapperStatusStopped,
				statusAfterRestart:   WrapperStatusRunning,
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
			},
			waitFor: waitFor{
				afterStart:     "wrapped log: 10ms",
				afterFirstExit: "wrapped log: 100ms",
				afterRestart:   "",
				afterCancel:    "",
			},
			want: want{
				statusAfterStart:     WrapperStatusRunning,
				statusAfterFirstExit: WrapperStatusStopped,
				statusAfterRestart:   WrapperStatusStopped,
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
			},
			waitFor: waitFor{
				afterStart:     "wrapped log: 10ms",
				afterFirstExit: "wrapped log: 100ms",
				afterRestart:   "",
				afterCancel:    "",
			},
			want: want{
				statusAfterStart:     WrapperStatusRunning,
				statusAfterFirstExit: WrapperStatusStopped,
				statusAfterRestart:   WrapperStatusStopped,
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
				timeout:      50 * time.Millisecond,
			},
			args: args{
				cancelWhileRunning: true,
				checkTimeout:       true,
			},
			waitFor: waitFor{
				afterStart:     "wrapped log: 10ms",
				afterFirstExit: "wrapped log: EXIT 100ms",
				afterRestart:   "wrapped log: 10ms",
				afterCancel:    "wrapped log: TERM SIGNAL",
			},
			want: want{
				statusAfterStart:     WrapperStatusRunning,
				statusAfterFirstExit: WrapperStatusStopped,
				statusAfterRestart:   WrapperStatusRunning,
				statusAfterDone:      WrapperStatusError,
				statusError:          255,
				wantErr:              true,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			console := testconsole.NewTestConsole()
			logger.New(console, "", "INFO")

			p := &wrapperHandler{
				arg:             tt.fields.arg,
				failOnStdErr:    tt.fields.failOnStdErr,
				hideStdErr:      tt.fields.hideStdErr,
				hideStdOut:      tt.fields.hideStdOut,
				path:            tt.fields.path,
				restartMode:     tt.fields.restart,
				restartInterval: 50 * time.Millisecond,
				timeout:         tt.fields.timeout,
			}

			chanWrapperData := make(chan WrapperData)
			chanWrapperDone := make(chan struct{})

			var tp testProcess
			done := tp.Start(chanWrapperData)

			if err := tp.AssertStatus(WrapperStatusStopped); err != nil {
				t.Errorf("before start: %s", err)
			}

			// create the context
			ctx, cancel := context.WithCancel(context.Background())

			// start the process
			go p.do(ctx, chanWrapperData, chanWrapperDone)

			ch := console.WaitForText(tt.waitFor.afterStart, 1*time.Second)
			if err := <-ch; err != nil {
				t.Fatal(err)
			}

			if err := tp.AssertStatus(tt.want.statusAfterStart); err != nil {
				t.Errorf("before start: %s", err)
			}

			ch = console.WaitForText(tt.waitFor.afterFirstExit, 1*time.Second)
			if err := <-ch; err != nil {
				t.Fatal(err)
			}

			if err := tp.AssertStatusChange(tt.want.statusAfterFirstExit, 5*time.Millisecond); err != nil {
				t.Errorf("after first exit: %s", err)
			}

			if tt.args.cancelWhileRunning {
				ch = console.WaitForText(tt.waitFor.afterRestart, 1*time.Second)
				if err := <-ch; err != nil {
					t.Fatal(err)
				}

				if err := tp.AssertStatus(tt.want.statusAfterRestart); err != nil {
					t.Errorf("after restart: %s", err)
				}
			}

			// cancel the context to terminate the process
			cancel()

			ch = console.WaitForText(tt.waitFor.afterCancel, 1*time.Second)
			if err := <-ch; err != nil {
				t.Fatal(err)
			}

			var timeoutStart time.Time
			if tt.args.checkTimeout {
				timeoutStart = time.Now()
			}

			<-done
			<-chanWrapperDone

			if tt.args.checkTimeout {
				timeoutDuration := time.Since(timeoutStart)

				if timeoutDuration > tt.fields.timeout+(30*time.Millisecond) {
					t.Errorf("after timeout: expected timeout == %v, got %v", tt.fields.timeout, timeoutDuration)
				}
			}

			if err := tp.AssertStatus(tt.want.statusAfterDone); err != nil {
				t.Errorf("after done: %s", err)
			}

			wrapperError := tp.WrapperError()
			if tt.want.wantErr && wrapperError == nil {
				t.Errorf("after done: expected an error, got %v", wrapperError)
			}

			if !tt.want.wantErr && wrapperError != nil {
				t.Errorf("after done: no error expected, got %v", wrapperError)
			}

			if exitStatusError, ok := wrapperError.(ProcessExitStatusError); tt.want.wantErr && ok {
				if exitStatusError.ExitStatus() != tt.want.statusError {
					t.Errorf("after done: expected exit status == %v, got %v", tt.want.statusError, exitStatusError.ExitStatus())
				}
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
		timeout      time.Duration
	}

	type args struct {
		cancelWhileRunning bool
	}

	type waitFor struct {
		afterStart     string
		afterError     string
		afterFirstExit string
		afterRestart   string
		afterCancel    string
	}

	type want struct {
		statusAfterFirstExit WrapperStatus
		err                  bool
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		waitFor waitFor
		want    want
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
			waitFor: waitFor{
				afterStart:     "wrapped log: 10ms",
				afterError:     "wrapped log: write a line to stderr",
				afterFirstExit: "wrapped process exited with status: 10",
				afterRestart:   "wrapped log: 10ms",
				afterCancel:    "wrapped log: TERM SIGNAL",
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
			waitFor: waitFor{
				afterStart:     "wrapped log: 10ms",
				afterError:     "wrapped log: write a line to stderr",
				afterFirstExit: "wrapped process exited with status: 10",
				afterRestart:   "wrapped log: 10ms",
				afterCancel:    "",
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
			waitFor: waitFor{
				afterStart:     "wrapped log: 10ms",
				afterError:     "wrapped log: write a line to stderr",
				afterFirstExit: "wrapped process exited with status: 10",
				afterRestart:   "wrapped log: 10ms",
				afterCancel:    "wrapped log: TERM SIGNAL",
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
			waitFor: waitFor{
				afterStart:     "wrapped log: 10ms",
				afterError:     "wrapped log: write a line to stderr",
				afterFirstExit: "wrapped process exited with status: 10",
				afterRestart:   "wrapped log: 10ms",
				afterCancel:    "",
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
			waitFor: waitFor{
				afterStart:     "wrapped log: 10ms",
				afterError:     "wrapped log: write a line to stderr",
				afterFirstExit: "wrapped log: 100ms",
				afterRestart:   "wrapped log: 10ms",
				afterCancel:    "wrapped log: TERM SIGNAL",
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
			waitFor: waitFor{
				afterStart:     "wrapped log: 10ms",
				afterError:     "wrapped log: write a line to stderr",
				afterFirstExit: "wrapped log: 100ms",
				afterRestart:   "wrapped log: 10ms",
				afterCancel:    "",
			},
			want: want{
				statusAfterFirstExit: WrapperStatusStopped,
				err:                  false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			console := testconsole.NewTestConsole()
			logger.New(console, "test", "INFO")

			// create the context
			ctx, cancel := context.WithCancel(context.Background())

			p := &wrapperHandler{
				arg:             tt.fields.arg,
				failOnStdErr:    tt.fields.failOnStdErr,
				hideStdErr:      tt.fields.hideStdErr,
				hideStdOut:      tt.fields.hideStdOut,
				path:            tt.fields.path,
				restartMode:     tt.fields.restart,
				restartInterval: 50 * time.Millisecond,
				timeout:         tt.fields.timeout,
			}

			chanWrapperData := make(chan WrapperData)
			chanWrapperDone := make(chan struct{})

			var tp testProcess
			done := tp.Start(chanWrapperData)

			if err := tp.AssertStatus(WrapperStatusStopped); err != nil {
				t.Errorf("before start: %s", err)
			}

			// start the process
			go p.do(ctx, chanWrapperData, chanWrapperDone)

			ch := console.WaitForText(tt.waitFor.afterStart, 1*time.Second)
			if err := <-ch; err != nil {
				t.Fatal(err)
			}

			if err := tp.AssertStatus(WrapperStatusRunning); err != nil {
				t.Errorf("after start: %s", err)
			}

			ch = console.WaitForText(tt.waitFor.afterError, 1*time.Second)
			if err := <-ch; err != nil {
				t.Fatal(err)
			}
			if err := tp.AssertStatusChange(WrapperStatusError, 5*time.Millisecond); err != nil {
				t.Errorf("after error: %s", err)
			}

			ch = console.WaitForText(tt.waitFor.afterFirstExit, 1*time.Second)
			if err := <-ch; err != nil {
				t.Fatal(err)
			}

			if err := tp.AssertStatusChange(tt.want.statusAfterFirstExit, 5*time.Millisecond); err != nil {
				t.Fatalf("after error: %s", err)
			}

			if tt.args.cancelWhileRunning {
				ch = console.WaitForText(tt.waitFor.afterRestart, 1*time.Second)
				if err := <-ch; err != nil {
					t.Fatal(err)
				}

				if err := tp.AssertStatus(WrapperStatusRunning); err != nil {
					t.Errorf("after start: %s", err)
				}
			}

			// cancel the context to terminate the process
			cancel()

			ch = console.WaitForText(tt.waitFor.afterCancel, 1*time.Second)
			if err := <-ch; err != nil {
				t.Fatal(err)
			}

			<-done
			<-chanWrapperDone

			wrapperStatus := tp.WrapperStatus()
			if !tt.want.err && wrapperStatus != WrapperStatusStopped {
				t.Errorf("after done: expected wrapperStatus != WrapperStatusStopped, got %v", wrapperStatus)
			}

			if tt.want.err && wrapperStatus != WrapperStatusError {
				t.Errorf("after done: expected wrapperStatus != WrapperStatusError, got %v", wrapperStatus)
			}

			wrapperError := tp.WrapperError()
			if tt.want.err && wrapperError == nil {
				t.Errorf("after done: expected an error, got %v", wrapperError)
			}

			if !tt.want.err && wrapperError != nil {
				t.Errorf("after done:  no error expected, got %v", wrapperError)
			}
		})
	}
}
