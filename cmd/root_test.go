package cmd

import (
	"context"
	"github.com/spf13/viper"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"testing"
	"time"

	myHttp "github.com/gandalfmagic/liveness-wrapper/internal/http"
	"github.com/gandalfmagic/liveness-wrapper/internal/system"
)

var testDirectory = "../test"

func Test_readConfig(t *testing.T) {
	t.Run("read_config", func(t *testing.T) {
		config = "../test/config/liveness-wrapper.yaml"

		if err := readConfig(); err != nil {
			t.Errorf("no error was expected, got one: %s", err)
		}

		processPath := viper.GetString("process.path")
		if processPath != "../test/cmd/test_int_no_err.sh" {
			t.Errorf("process.path expected: %v, got %v", "../test/cmd/test_int_no_err.sh", processPath)
		}

		processFailOnStdErr := viper.GetBool("process.fail-on-stderr")
		if !processFailOnStdErr {
			t.Errorf("process.fail-on-stderr expected: %v, got %v", true, processFailOnStdErr)
		}

		processHideStdErr := viper.GetBool("process.hide-stderr")
		if processHideStdErr {
			t.Errorf("process.hide-stderr expected: %v, got %v", false, processHideStdErr)
		}

		processHideStdOut := viper.GetBool("process.hide-stdout")
		if processHideStdOut {
			t.Errorf("process.hide-stdout expected: %v, got %v", false, processHideStdOut)
		}

		processRestartAlways := viper.GetBool("process.restart-always")
		if processRestartAlways {
			t.Errorf("process.restart-always expected: %v, got %v", false, processRestartAlways)
		}

		processRestartOnError := viper.GetBool("process.restart-on-error")
		if !processRestartOnError {
			t.Errorf("process.restart-on-error expected: %v, got %v", true, processRestartOnError)
		}

		processRestartTimeout := viper.GetDuration("process.timeout")
		if processRestartTimeout != 31*time.Second {
			t.Errorf("process.timeout expected: %v, got %v", 31*time.Second, processRestartTimeout)
		}

		serverAddress := viper.GetString("server.address")
		if serverAddress != ":6060" {
			t.Errorf("process.timeout expected: %v, got %v", ":6060", serverAddress)
		}

		serverPingTimeout := viper.GetDuration("server.ping-timeout")
		if serverPingTimeout != 10*time.Minute {
			t.Errorf("process.ping-timeout expected: %v, got %v", 10*time.Minute, serverPingTimeout)
		}

		serverShutdownTimeout := viper.GetDuration("server.shutdown-timeout")
		if serverShutdownTimeout != 15*time.Second {
			t.Errorf("process.shutdown-timeout expected: %v, got %v", 15*time.Second, serverShutdownTimeout)
		}

		logLevel := viper.GetString("log.level")
		if logLevel != "INFO" {
			t.Errorf("log.level expected: %v, got %v", "INFO", logLevel)
		}

		config = ""
	})
}

func Test_getRestartMode(t *testing.T) {
	type args struct {
		restartAlways  bool
		restartOnError bool
	}
	tests := []struct {
		name string
		args args
		want system.WrapperRestartMode
	}{
		{
			name: "always",
			args: args{true, false},
			want: system.WrapperRestartAlways,
		},
		{
			name: "always_and_on_error",
			args: args{true, true},
			want: system.WrapperRestartAlways,
		},
		{
			name: "on_error",
			args: args{false, true},
			want: system.WrapperRestartOnError,
		},
		{
			name: "never",
			args: args{false, false},
			want: system.WrapperRestartNever,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getRestartMode(tt.args.restartAlways, tt.args.restartOnError); got != tt.want {
				t.Errorf("getRestartMode() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_runner_wait(t *testing.T) {
	t.Run("Exit_without_error", func(t *testing.T) {
		ctx, cancelServer := context.WithCancel(context.Background())

		// create the http server
		server := myHttp.NewServer("127.0.0.1:6060", 15*time.Second, 10*time.Minute)
		updateReady, updateAlive, serverDone := server.Start(ctx)

		ctx, cancelWrapper := context.WithCancel(context.Background())

		// start the wrapped process
		process := system.NewWrapperHandler(system.WrapperRestartNever, false, false, false, 1*time.Second, filepath.Join(testDirectory, "cmd/test_int_no_err.sh"))
		wrapperData, wrapperDone := process.Start(ctx)

		r := &runner{
			serverDone:  serverDone,
			updateAlive: updateAlive,
			updateReady: updateReady,
			wrapperData: wrapperData,
			wrapperDone: wrapperDone,
		}

		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)
		defer close(c)

		// execute the process
		chanErr := make(chan error)
		go func() {
			chanErr <- r.wait(cancelWrapper, cancelServer, c)
		}()

		// wait for the process to start
		time.Sleep(20 * time.Millisecond)

		// testing the endpoints
		var rsp *http.Response
		rsp, _ = http.Get("http://127.0.0.1:6060/ready")
		if rsp.StatusCode != 200 {
			t.Errorf("Expected status code 200 on /ready, got %v", rsp.StatusCode)
		}

		rsp, _ = http.Get("http://127.0.0.1:6060/alive")
		if rsp.StatusCode != 200 {
			t.Errorf("Expected status code 200 on /alive, got %v", rsp.StatusCode)
		}

		err := <-chanErr
		if err != nil {
			t.Errorf("no error was expected, got %s", err)
		}
	})

	t.Run("Exit_with_error", func(t *testing.T) {
		ctx, cancelFuncHttp := context.WithCancel(context.Background())

		// create the http server
		server := myHttp.NewServer("127.0.0.1:6060", 15*time.Second, 10*time.Minute)
		updateReady, updateAlive, serverDone := server.Start(ctx)

		ctx, cancelFuncProcess := context.WithCancel(context.Background())

		// start the wrapped process
		process := system.NewWrapperHandler(system.WrapperRestartNever, false, false, false, 1*time.Second, filepath.Join(testDirectory, "cmd/error_10_int_no_err.sh"))
		wrapperData, wrapperDone := process.Start(ctx)

		r := &runner{
			serverDone:  serverDone,
			updateAlive: updateAlive,
			updateReady: updateReady,
			wrapperData: wrapperData,
			wrapperDone: wrapperDone,
		}

		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)
		defer close(c)

		// execute the process
		chanErr := make(chan error)
		go func() {
			chanErr <- r.wait(cancelFuncProcess, cancelFuncHttp, c)
		}()

		// wait for the process to start
		time.Sleep(20 * time.Millisecond)

		// testing the endpoints
		var rsp *http.Response

		rsp, _ = http.Get("http://127.0.0.1:6060/ready")
		if rsp.StatusCode != 200 {
			t.Errorf("Expected status code 200 on /ready, got %v", rsp.StatusCode)
		}

		rsp, _ = http.Get("http://127.0.0.1:6060/alive")
		if rsp.StatusCode != 200 {
			t.Errorf("Expected status code 200 on /alive, got %v", rsp.StatusCode)
		}

		err := <-chanErr
		if err == nil {
			t.Error("an error was expected, got no one")
		}
	})

	t.Run("SIGINT", func(t *testing.T) {
		ctx, cancelServer := context.WithCancel(context.Background())

		// create the http server
		server := myHttp.NewServer("127.0.0.1:6060", 15*time.Second, 10*time.Minute)
		updateReady, updateAlive, serverDone := server.Start(ctx)

		ctx, cancelWrapper := context.WithCancel(context.Background())

		// start the wrapped process
		process := system.NewWrapperHandler(system.WrapperRestartNever, false, false, false, 1*time.Second, filepath.Join(testDirectory, "cmd/test_int_no_err.sh"))
		wrapperData, wrapperDone := process.Start(ctx)

		r := &runner{
			serverDone:  serverDone,
			updateAlive: updateAlive,
			updateReady: updateReady,
			wrapperData: wrapperData,
			wrapperDone: wrapperDone,
		}

		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)
		defer close(c)

		// execute the process
		chanErr := make(chan error)
		go func() {
			chanErr <- r.wait(cancelWrapper, cancelServer, c)
		}()

		// wait for the process to start
		time.Sleep(20 * time.Millisecond)

		// testing the endpoints
		var rsp *http.Response

		rsp, _ = http.Get("http://127.0.0.1:6060/ready")
		if rsp.StatusCode != 200 {
			t.Errorf("Expected status code 200 on /ready, got %v", rsp.StatusCode)
		}

		rsp, _ = http.Get("http://127.0.0.1:6060/alive")
		if rsp.StatusCode != 200 {
			t.Errorf("Expected status code 200 on /alive, got %v", rsp.StatusCode)
		}

		// send CTRL-C
		c <- os.Interrupt

		time.Sleep(10 * time.Millisecond)

		// test alive endpoint
		rsp, _ = http.Get("http://127.0.0.1:6060/alive")
		if rsp.StatusCode != 200 {
			t.Errorf("Expected status code 200 on /alive, got %v", rsp.StatusCode)
		}
		defer func() {
			_ = rsp.Body.Close()
		}()

		// test ready endpoint
		rsp, _ = http.Get("http://127.0.0.1:6060/ready")
		if rsp.StatusCode != 503 {
			t.Errorf("Expected status code 503 on /alive, got %v", rsp.StatusCode)
		}
		defer func() {
			_ = rsp.Body.Close()
		}()

		err := <-chanErr
		if err != nil {
			t.Errorf("no error was expected, got %s", err)
		}
	})

	t.Run("Restart_on_error_Exit_with_error_Kill_while_NOT_running", func(t *testing.T) {
		ctx, cancelServer := context.WithCancel(context.Background())

		// create the http server
		server := myHttp.NewServer("127.0.0.1:6060", 15*time.Second, 10*time.Minute)
		updateReady, updateAlive, serverDone := server.Start(ctx)

		ctx, cancelWrapper := context.WithCancel(context.Background())

		// start the wrapped process
		process := system.NewWrapperHandler(system.WrapperRestartOnError, false, false, false, 1*time.Second, filepath.Join(testDirectory, "cmd/error_10_int_no_err.sh"))
		wrapperData, wrapperDone := process.Start(ctx)

		r := &runner{
			serverDone:  serverDone,
			updateAlive: updateAlive,
			updateReady: updateReady,
			wrapperData: wrapperData,
			wrapperDone: wrapperDone,
		}

		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)
		defer close(c)

		// execute the process
		chanErr := make(chan error)
		go func() {
			chanErr <- r.wait(cancelWrapper, cancelServer, c)
		}()

		// wait for the process to start
		time.Sleep(10 * time.Millisecond)

		var rsp *http.Response

		// testing ready endpoint
		rsp, _ = http.Get("http://127.0.0.1:6060/ready")
		if rsp.StatusCode != 200 {
			t.Errorf("Expected status code 200 on /ready, got %v", rsp.StatusCode)
		}

		// testing alive endpoint
		rsp, _ = http.Get("http://127.0.0.1:6060/alive")
		if rsp.StatusCode != 200 {
			t.Errorf("Expected status code 200 on /alive, got %v", rsp.StatusCode)
		}

		// wait for the process to terminate with error
		time.Sleep(260 * time.Millisecond) // 100ms normal execution + 100ms after exit

		// testing ready endpoint
		rsp, _ = http.Get("http://127.0.0.1:6060/ready")
		if rsp.StatusCode != 200 {
			t.Errorf("Expected status code 200 on /ready, got %v", rsp.StatusCode)
		}

		// testing alive endpoint
		rsp, _ = http.Get("http://127.0.0.1:6060/alive")
		if rsp.StatusCode != 503 {
			t.Errorf("Expected status code 503 on /alive, got %v", rsp.StatusCode)
		}

		// send CTRL-C while the process is not running
		c <- os.Interrupt

		err := <-chanErr
		if err == nil {
			t.Error("an error was expected, got no one")
		}
		log.Printf("%s", err)
	})

	t.Run("Restart_on_error_Exit_with_error_Kill_while_IS_running", func(t *testing.T) {
		ctx, cancelFuncHttp := context.WithCancel(context.Background())

		// create the http server
		server := myHttp.NewServer("127.0.0.1:6060", 15*time.Second, 10*time.Minute)
		updateReady, updateAlive, serverDone := server.Start(ctx)

		ctx, cancelFuncProcess := context.WithCancel(context.Background())

		// start the wrapped process
		process := system.NewWrapperHandler(system.WrapperRestartOnError, false, false, false, 1*time.Second, filepath.Join(testDirectory, "cmd/error_10_int_no_err.sh"))
		wrapperData, wrapperDone := process.Start(ctx)

		r := &runner{
			serverDone:  serverDone,
			updateAlive: updateAlive,
			updateReady: updateReady,
			wrapperData: wrapperData,
			wrapperDone: wrapperDone,
		}

		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)
		defer close(c)

		// execute the process
		chanErr := make(chan error)
		go func() {
			chanErr <- r.wait(cancelFuncProcess, cancelFuncHttp, c)
		}()

		// wait for the process to start
		time.Sleep(20 * time.Millisecond)

		var rsp *http.Response

		// testing ready endpoint
		rsp, _ = http.Get("http://127.0.0.1:6060/ready")
		if rsp.StatusCode != 200 {
			t.Errorf("after start: expected status code 200 on /ready, got %v", rsp.StatusCode)
		}

		// testing alive endpoint
		rsp, _ = http.Get("http://127.0.0.1:6060/alive")
		if rsp.StatusCode != 200 {
			t.Errorf("after start: expected status code 200 on /alive, got %v", rsp.StatusCode)
		}

		// wait for the process to terminate with error
		time.Sleep(260 * time.Millisecond)

		// testing ready endpoint
		rsp, _ = http.Get("http://127.0.0.1:6060/ready")
		if rsp.StatusCode != 200 {
			t.Errorf("after exit: expected status code 200 on /ready, got %v", rsp.StatusCode)
		}

		// testing alive endpoint
		rsp, _ = http.Get("http://127.0.0.1:6060/alive")
		if rsp.StatusCode != 503 {
			t.Errorf("after exit: expected status code 503 on /alive, got %v", rsp.StatusCode)
		}

		// wait for the process to restart
		time.Sleep(1000 * time.Millisecond)

		// testing ready endpoint
		rsp, _ = http.Get("http://127.0.0.1:6060/ready")
		if rsp.StatusCode != 200 {
			t.Errorf("after restart: expected status code 200 on /ready, got %v", rsp.StatusCode)
		}

		// testing alive endpoint
		rsp, _ = http.Get("http://127.0.0.1:6060/alive")
		if rsp.StatusCode != 200 {
			t.Errorf("after restart: expected status code 200 on /alive, got %v", rsp.StatusCode)
		}

		// send CTRL-C while the process is running
		c <- os.Interrupt

		// wait for the process to restart
		time.Sleep(10 * time.Millisecond)

		// testing ready endpoint
		rsp, _ = http.Get("http://127.0.0.1:6060/ready")
		if rsp.StatusCode != 503 {
			t.Errorf("after CRTL+C: expected status code 503 on /ready, got %v", rsp.StatusCode)
		}
		defer func() {
			_ = rsp.Body.Close()
		}()

		// testing alive endpoint
		rsp, _ = http.Get("http://127.0.0.1:6060/alive")
		if rsp.StatusCode != 200 {
			t.Errorf("after CRTL+C: expected status code 200 on /alive, got %v", rsp.StatusCode)
		}
		defer func() {
			_ = rsp.Body.Close()
		}()

		err := <-chanErr
		if err != nil {
			t.Errorf("after done: no error was expected, got %s", err)
		}
	})

	t.Run("Test_ping", func(t *testing.T) {
		ctx, cancelFuncHttp := context.WithCancel(context.Background())

		// create the http server
		server := myHttp.NewServer("127.0.0.1:6060", 15*time.Second, 50*time.Millisecond)
		updateReady, updateAlive, serverDone := server.Start(ctx)

		ctx, cancelFuncProcess := context.WithCancel(context.Background())

		// start the wrapped process
		process := system.NewWrapperHandler(system.WrapperRestartNever, false, false, false, 1*time.Second, filepath.Join(testDirectory, "cmd/test_int_no_err.sh"))
		wrapperData, wrapperDone := process.Start(ctx)

		r := &runner{
			serverDone:  serverDone,
			updateAlive: updateAlive,
			updateReady: updateReady,
			wrapperData: wrapperData,
			wrapperDone: wrapperDone,
		}

		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)
		defer close(c)

		// execute the process
		chanErr := make(chan error)
		go func() {
			chanErr <- r.wait(cancelFuncProcess, cancelFuncHttp, c)
		}()

		// wait for the process to start
		time.Sleep(10 * time.Millisecond)

		var rsp *http.Response

		// testing ready endpoint
		rsp, _ = http.Get("http://127.0.0.1:6060/ready")
		if rsp.StatusCode != 200 {
			t.Errorf("after start: expected status code 200 on /ready, got %v", rsp.StatusCode)
		}

		// testing alive endpoint
		rsp, _ = http.Get("http://127.0.0.1:6060/alive")
		if rsp.StatusCode != 200 {
			t.Errorf("after start: expected status code 200 on /alive, got %v", rsp.StatusCode)
		}

		// send request to the /ping endpoint
		rsp, _ = http.Get("http://127.0.0.1:6060/ping")
		if rsp.StatusCode != 200 {
			t.Errorf("after start: expected status code 200 on /ping, got %v", rsp.StatusCode)
		}

		// testing ready endpoint
		rsp, _ = http.Get("http://127.0.0.1:6060/ready")
		if rsp.StatusCode != 200 {
			t.Errorf("after start: expected status code 200 on /ready, got %v", rsp.StatusCode)
		}

		// testing alive endpoint
		rsp, _ = http.Get("http://127.0.0.1:6060/alive")
		if rsp.StatusCode != 200 {
			t.Errorf("after start: expected status code 200 on /alive, got %v", rsp.StatusCode)
		}

		// wait 60ms for the /ping timeout to expire
		time.Sleep(60 * time.Millisecond)

		// testing ready endpoint
		rsp, _ = http.Get("http://127.0.0.1:6060/ready")
		if rsp.StatusCode != 200 {
			t.Errorf("after 1st expiration: expected status code 200 on /ready, got %v", rsp.StatusCode)
		}

		// testing alive endpoint
		rsp, _ = http.Get("http://127.0.0.1:6060/alive")
		if rsp.StatusCode != 503 {
			t.Errorf("after 1st expiration: expected status code 503 on /alive, got %v", rsp.StatusCode)
		}

		// send request to the /ping endpoint
		rsp, _ = http.Get("http://127.0.0.1:6060/ping")
		if rsp.StatusCode != 200 {
			t.Errorf("after 1st expiration: expected status code 200 on /ping, got %v", rsp.StatusCode)
		}

		// testing ready endpoint
		rsp, _ = http.Get("http://127.0.0.1:6060/ready")
		if rsp.StatusCode != 200 {
			t.Errorf("after 1st expiration: expected status code 200 on /ready, got %v", rsp.StatusCode)
		}

		// testing alive endpoint
		rsp, _ = http.Get("http://127.0.0.1:6060/alive")
		if rsp.StatusCode != 200 {
			t.Errorf("after 1st expiration: expected status code 200 on /alive, got %v", rsp.StatusCode)
		}

		// wait 30ms, 20ms before the ping endpoint expires
		time.Sleep(30 * time.Millisecond)

		// testing ready endpoint
		rsp, _ = http.Get("http://127.0.0.1:6060/ready")
		if rsp.StatusCode != 200 {
			t.Errorf("before 2nd expiration: expected status code 200 on /ready, got %v", rsp.StatusCode)
		}

		// testing alive endpoint
		rsp, _ = http.Get("http://127.0.0.1:6060/alive")
		if rsp.StatusCode != 200 {
			t.Errorf("before 2nd expiration: expected status code 200 on /alive, got %v", rsp.StatusCode)
		}

		// wait 30ms for the /ping timeout to expire
		time.Sleep(30 * time.Millisecond)

		// testing ready endpoint
		rsp, _ = http.Get("http://127.0.0.1:6060/ready")
		if rsp.StatusCode != 200 {
			t.Errorf("after 2nd expiration: expected status code 200 on /ready, got %v", rsp.StatusCode)
		}

		// testing alive endpoint
		rsp, _ = http.Get("http://127.0.0.1:6060/alive")
		if rsp.StatusCode != 503 {
			t.Errorf("after 2nd expiration: expected status code 503 on /alive, got %v", rsp.StatusCode)
		}

		err := <-chanErr
		if err != nil {
			t.Errorf("after done: no error was expected, got %s", err)
		}
	})
}

func Test_run(t *testing.T) {
	t.Run("run", func(t *testing.T) {
		config = "../test/config/liveness-wrapper.yaml"

		if err := readConfig(); err != nil {
			t.Errorf("readConfig: no error was expected, got one: %s", err)
		}

		if err := run(nil, nil); err != nil {
			t.Errorf("run: no error was expected, got one: %s", err)
		}

		config = ""
	})
}
