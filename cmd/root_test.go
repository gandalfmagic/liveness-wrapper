package cmd

import (
	"context"
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

		// test ready endpoint
		rsp, _ = http.Get("http://127.0.0.1:6060/ready")
		if rsp.StatusCode != 503 {
			t.Errorf("Expected status code 503 on /alive, got %v", rsp.StatusCode)
		}

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
		time.Sleep(120 * time.Millisecond)

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
		time.Sleep(120 * time.Millisecond)

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

		// wait for the process to restart
		time.Sleep(1000 * time.Millisecond)

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

		// send CTRL-C while the process is running
		c <- os.Interrupt

		// wait for the process to restart
		time.Sleep(10 * time.Millisecond)

		// testing ready endpoint
		rsp, _ = http.Get("http://127.0.0.1:6060/ready")
		if rsp.StatusCode != 503 {
			t.Errorf("Expected status code 503 on /ready, got %v", rsp.StatusCode)
		}

		// testing alive endpoint
		rsp, _ = http.Get("http://127.0.0.1:6060/alive")
		if rsp.StatusCode != 200 {
			t.Errorf("Expected status code 200 on /alive, got %v", rsp.StatusCode)
		}

		err := <-chanErr
		if err != nil {
			t.Errorf("no error was expected, got %s", err)
		}
	})

	t.Run("Test_ping", func(t *testing.T) {
		ctx, cancelFuncHttp := context.WithCancel(context.Background())

		// create the http server
		server := myHttp.NewServer("127.0.0.1:6060", 15*time.Second, 15*time.Millisecond)
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
			t.Errorf("Expected status code 200 on /ready, got %v", rsp.StatusCode)
		}

		// testing alive endpoint
		rsp, _ = http.Get("http://127.0.0.1:6060/alive")
		if rsp.StatusCode != 200 {
			t.Errorf("Expected status code 200 on /alive, got %v", rsp.StatusCode)
		}

		// send request to the /ping endpoint
		rsp, _ = http.Get("http://127.0.0.1:6060/ping")
		if rsp.StatusCode != 200 {
			t.Errorf("Expected status code 200 on /ping, got %v", rsp.StatusCode)
		}

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

		// wait 15ms for the /ping timeout to expire
		time.Sleep(15 * time.Millisecond)

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

		// send request to the /ping endpoint
		rsp, _ = http.Get("http://127.0.0.1:6060/ping")
		if rsp.StatusCode != 200 {
			t.Errorf("Expected status code 200 on /ping, got %v", rsp.StatusCode)
		}

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

		// wait 10ms, 5ms before the ping endpoint expires
		time.Sleep(10 * time.Millisecond)

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

		// wait 5ms for the /ping timeout to expire
		time.Sleep(5 * time.Millisecond)

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

		err := <-chanErr
		if err != nil {
			t.Errorf("no error was expected, got %s", err)
		}
	})
}
