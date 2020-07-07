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
		ctx, cancelFuncHttp := context.WithCancel(context.Background())

		// create the http server
		server := myHttp.NewServer(ctx, "127.0.0.1:6060", 10*time.Minute)
		updateReady, updateAlive, serverDone := server.Start()

		ctx, cancelFuncProcess := context.WithCancel(context.Background())

		// start the wrapped process
		process := system.NewWrapperHandler(ctx, system.WrapperRestartNever, false, false, false, filepath.Join(testDirectory, "test.sh"))
		wrapperData := process.Start()

		r := &runner{
			serverDone:  serverDone,
			updateAlive: updateAlive,
			updateReady: updateReady,
			wrapperData: wrapperData,
		}

		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)
		defer close(c)

		t.Log("execute the process")
		chanErr := make(chan error)
		go func() {
			chanErr <- r.wait(cancelFuncProcess, cancelFuncHttp, c)
		}()

		t.Log("wait for the process to start")
		time.Sleep(5 * time.Millisecond)

		t.Log("testing the endpoints again")
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
		server := myHttp.NewServer(ctx, "127.0.0.1:6060", 10*time.Minute)
		updateReady, updateAlive, serverDone := server.Start()

		ctx, cancelFuncProcess := context.WithCancel(context.Background())

		// start the wrapped process
		process := system.NewWrapperHandler(ctx, system.WrapperRestartNever, false, false, false, filepath.Join(testDirectory, "error_10.sh"))
		wrapperData := process.Start()

		r := &runner{
			serverDone:  serverDone,
			updateAlive: updateAlive,
			updateReady: updateReady,
			wrapperData: wrapperData,
		}

		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)
		defer close(c)

		t.Log("execute the process")
		chanErr := make(chan error)
		go func() {
			chanErr <- r.wait(cancelFuncProcess, cancelFuncHttp, c)
		}()

		t.Log("wait for the process to start")
		time.Sleep(5 * time.Millisecond)

		t.Log("testing the endpoints")
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
		log.Printf("%s", err)
	})

	t.Run("SIGINT", func(t *testing.T) {
		ctx, cancelFuncHttp := context.WithCancel(context.Background())

		// create the http server
		server := myHttp.NewServer(ctx, "127.0.0.1:6060", 10*time.Minute)
		updateReady, updateAlive, serverDone := server.Start()

		ctx, cancelFuncProcess := context.WithCancel(context.Background())

		// start the wrapped process
		process := system.NewWrapperHandler(ctx, system.WrapperRestartNever, false, false, false, filepath.Join(testDirectory, "test.sh"))
		wrapperData := process.Start()

		r := &runner{
			serverDone:  serverDone,
			updateAlive: updateAlive,
			updateReady: updateReady,
			wrapperData: wrapperData,
		}

		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)
		defer close(c)

		t.Log("execute the process")
		chanErr := make(chan error)
		go func() {
			chanErr <- r.wait(cancelFuncProcess, cancelFuncHttp, c)
		}()

		t.Log("wait for the process to start")
		time.Sleep(5 * time.Millisecond)

		t.Log("testing the endpoints")
		var rsp *http.Response

		rsp, _ = http.Get("http://127.0.0.1:6060/ready")
		if rsp.StatusCode != 200 {
			t.Errorf("Expected status code 200 on /ready, got %v", rsp.StatusCode)
		}

		rsp, _ = http.Get("http://127.0.0.1:6060/alive")
		if rsp.StatusCode != 200 {
			t.Errorf("Expected status code 200 on /alive, got %v", rsp.StatusCode)
		}

		t.Log("send CTRL-C")
		c <- os.Interrupt

		t.Log("testing the endpoints again")
		rsp, _ = http.Get("http://127.0.0.1:6060/ready")
		if rsp.StatusCode != 503 {
			t.Errorf("Expected status code 503 on /ready, got %v", rsp.StatusCode)
		}

		rsp, _ = http.Get("http://127.0.0.1:6060/alive")
		if rsp.StatusCode != 200 {
			t.Errorf("Expected status code 200 on /alive, got %v", rsp.StatusCode)
		}

		err := <-chanErr
		if err == nil {
			t.Error("an error was expected, got no one")
		}
		log.Printf("%s", err)
	})

	t.Run("Restart_on_error_Exit_with_error_Kill_while_NOT_running", func(t *testing.T) {
		ctx, cancelFuncHttp := context.WithCancel(context.Background())

		// create the http server
		server := myHttp.NewServer(ctx, "127.0.0.1:6060", 10*time.Minute)
		updateReady, updateAlive, serverDone := server.Start()

		ctx, cancelFuncProcess := context.WithCancel(context.Background())

		// start the wrapped process
		process := system.NewWrapperHandler(ctx, system.WrapperRestartOnError, false, false, false, filepath.Join(testDirectory, "error_10.sh"))
		wrapperData := process.Start()

		r := &runner{
			serverDone:  serverDone,
			updateAlive: updateAlive,
			updateReady: updateReady,
			wrapperData: wrapperData,
		}

		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)
		defer close(c)

		t.Log("execute the process")
		chanErr := make(chan error)
		go func() {
			chanErr <- r.wait(cancelFuncProcess, cancelFuncHttp, c)
		}()

		t.Log("wait for the process to start")
		time.Sleep(5 * time.Millisecond)

		t.Log("testing the endpoints")
		var rsp *http.Response

		rsp, _ = http.Get("http://127.0.0.1:6060/ready")
		if rsp.StatusCode != 200 {
			t.Errorf("Expected status code 200 on /ready, got %v", rsp.StatusCode)
		}

		rsp, _ = http.Get("http://127.0.0.1:6060/alive")
		if rsp.StatusCode != 200 {
			t.Errorf("Expected status code 200 on /alive, got %v", rsp.StatusCode)
		}

		t.Log("wait for the process to terminate with error")
		time.Sleep(100 * time.Millisecond)

		t.Log("testing the endpoints")
		rsp, _ = http.Get("http://127.0.0.1:6060/ready")
		if rsp.StatusCode != 200 {
			t.Errorf("Expected status code 200 on /ready, got %v", rsp.StatusCode)
		}

		rsp, _ = http.Get("http://127.0.0.1:6060/alive")
		if rsp.StatusCode != 503 {
			t.Errorf("Expected status code 503 on /alive, got %v", rsp.StatusCode)
		}

		t.Log("send CTRL-C while the process is not running")
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
		server := myHttp.NewServer(ctx, "127.0.0.1:6060", 10*time.Minute)
		updateReady, updateAlive, serverDone := server.Start()

		ctx, cancelFuncProcess := context.WithCancel(context.Background())

		// start the wrapped process
		process := system.NewWrapperHandler(ctx, system.WrapperRestartOnError, false, false, false, filepath.Join(testDirectory, "error_10.sh"))
		wrapperData := process.Start()

		r := &runner{
			serverDone:  serverDone,
			updateAlive: updateAlive,
			updateReady: updateReady,
			wrapperData: wrapperData,
		}

		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)
		defer close(c)

		t.Log("execute the process")
		chanErr := make(chan error)
		go func() {
			chanErr <- r.wait(cancelFuncProcess, cancelFuncHttp, c)
		}()

		t.Log("wait for the process to start")
		time.Sleep(5 * time.Millisecond)

		t.Log("testing the endpoints")
		var rsp *http.Response

		rsp, _ = http.Get("http://127.0.0.1:6060/ready")
		if rsp.StatusCode != 200 {
			t.Errorf("Expected status code 200 on /ready, got %v", rsp.StatusCode)
		}

		rsp, _ = http.Get("http://127.0.0.1:6060/alive")
		if rsp.StatusCode != 200 {
			t.Errorf("Expected status code 200 on /alive, got %v", rsp.StatusCode)
		}

		t.Log("wait for the process to terminate with error")
		time.Sleep(110 * time.Millisecond)

		t.Log("testing the endpoints")
		rsp, _ = http.Get("http://127.0.0.1:6060/ready")
		if rsp.StatusCode != 200 {
			t.Errorf("Expected status code 200 on /ready, got %v", rsp.StatusCode)
		}

		rsp, _ = http.Get("http://127.0.0.1:6060/alive")
		if rsp.StatusCode != 503 {
			t.Errorf("Expected status code 503 on /alive, got %v", rsp.StatusCode)
		}

		t.Log("wait for the process to restart")
		time.Sleep(1010 * time.Millisecond)

		t.Log("testing the endpoints")
		rsp, _ = http.Get("http://127.0.0.1:6060/ready")
		if rsp.StatusCode != 200 {
			t.Errorf("Expected status code 200 on /ready, got %v", rsp.StatusCode)
		}

		rsp, _ = http.Get("http://127.0.0.1:6060/alive")
		if rsp.StatusCode != 200 {
			t.Errorf("Expected status code 200 on /alive, got %v", rsp.StatusCode)
		}

		t.Log("send CTRL-C while the process is running")
		c <- os.Interrupt

		err := <-chanErr
		if err == nil {
			t.Error("an error was expected, got no one")
		}
		log.Printf("%s", err)
	})

	t.Run("Test_ping", func(t *testing.T) {
		ctx, cancelFuncHttp := context.WithCancel(context.Background())

		// create the http server
		server := myHttp.NewServer(ctx, "127.0.0.1:6060", 20*time.Millisecond)
		updateReady, updateAlive, serverDone := server.Start()

		ctx, cancelFuncProcess := context.WithCancel(context.Background())

		// start the wrapped process
		process := system.NewWrapperHandler(ctx, system.WrapperRestartNever, false, false, false, filepath.Join(testDirectory, "test.sh"))
		wrapperData := process.Start()

		r := &runner{
			serverDone:  serverDone,
			updateAlive: updateAlive,
			updateReady: updateReady,
			wrapperData: wrapperData,
		}

		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)
		defer close(c)

		t.Log("execute the process")
		chanErr := make(chan error)
		go func() {
			chanErr <- r.wait(cancelFuncProcess, cancelFuncHttp, c)
		}()

		t.Log("wait for the process to start")
		time.Sleep(5 * time.Millisecond)

		t.Log("testing the endpoints")
		var rsp *http.Response

		rsp, _ = http.Get("http://127.0.0.1:6060/ready")
		if rsp.StatusCode != 200 {
			t.Errorf("Expected status code 200 on /ready, got %v", rsp.StatusCode)
		}

		rsp, _ = http.Get("http://127.0.0.1:6060/alive")
		if rsp.StatusCode != 200 {
			t.Errorf("Expected status code 200 on /alive, got %v", rsp.StatusCode)
		}

		t.Log("send request to the /ping endpoint")
		rsp, _ = http.Get("http://127.0.0.1:6060/ping")
		if rsp.StatusCode != 200 {
			t.Errorf("Expected status code 200 on /ping, got %v", rsp.StatusCode)
		}

		rsp, _ = http.Get("http://127.0.0.1:6060/ready")
		if rsp.StatusCode != 200 {
			t.Errorf("Expected status code 200 on /ready, got %v", rsp.StatusCode)
		}

		rsp, _ = http.Get("http://127.0.0.1:6060/alive")
		if rsp.StatusCode != 200 {
			t.Errorf("Expected status code 200 on /alive, got %v", rsp.StatusCode)
		}

		t.Log("wait for the /ping timeout to expire")
		time.Sleep(20 * time.Millisecond)

		rsp, _ = http.Get("http://127.0.0.1:6060/ready")
		if rsp.StatusCode != 200 {
			t.Errorf("Expected status code 200 on /ready, got %v", rsp.StatusCode)
		}

		rsp, _ = http.Get("http://127.0.0.1:6060/alive")
		if rsp.StatusCode != 503 {
			t.Errorf("Expected status code 503 on /alive, got %v", rsp.StatusCode)
		}

		t.Log("send request to the /ping endpoint")
		rsp, _ = http.Get("http://127.0.0.1:6060/ping")
		if rsp.StatusCode != 200 {
			t.Errorf("Expected status code 200 on /ping, got %v", rsp.StatusCode)
		}

		rsp, _ = http.Get("http://127.0.0.1:6060/ready")
		if rsp.StatusCode != 200 {
			t.Errorf("Expected status code 200 on /ready, got %v", rsp.StatusCode)
		}

		rsp, _ = http.Get("http://127.0.0.1:6060/alive")
		if rsp.StatusCode != 200 {
			t.Errorf("Expected status code 200 on /alive, got %v", rsp.StatusCode)
		}

		t.Log("wait 18ms for the /ping timeout to expire")
		time.Sleep(18 * time.Millisecond)

		rsp, _ = http.Get("http://127.0.0.1:6060/ready")
		if rsp.StatusCode != 200 {
			t.Errorf("Expected status code 200 on /ready, got %v", rsp.StatusCode)
		}

		rsp, _ = http.Get("http://127.0.0.1:6060/alive")
		if rsp.StatusCode != 200 {
			t.Errorf("Expected status code 200 on /alive, got %v", rsp.StatusCode)
		}

		t.Log("wait 2ms for the /ping timeout to expire")
		time.Sleep(2 * time.Millisecond)

		rsp, _ = http.Get("http://127.0.0.1:6060/ready")
		if rsp.StatusCode != 200 {
			t.Errorf("Expected status code 200 on /ready, got %v", rsp.StatusCode)
		}

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
