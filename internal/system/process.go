package system

import (
	"context"
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/gandalfmagic/liveness-wrapper/pkg/logger"
)

type WrapperStatus int

const (
	WrapperStatusStopped WrapperStatus = iota
	WrapperStatusRunning
	WrapperStatusError
)

type WrapperRestartMode int

const (
	WrapperRestartNever WrapperRestartMode = iota
	WrapperRestartOnError
	WrapperRestartAlways
)

type WrapperConfiguration struct {
	RestartMode  WrapperRestartMode
	HideStdOut   bool
	HideStdErr   bool
	FailOnStdErr bool
	Timeout      time.Duration
	Path         string
}

type WrapperData struct {
	WrapperStatus WrapperStatus
	Err           error
	Done          bool
}

type WrapperHandler interface {
	Start(ctx context.Context) (<-chan WrapperData, <-chan struct{})
}

type wrapperHandler struct {
	arg             []string
	failOnStdErr    bool
	hideStdErr      bool
	hideStdOut      bool
	path            string
	restartMode     WrapperRestartMode
	restartInterval time.Duration
	timeout         time.Duration
}

// NewWrapperStatus creates a new process wrapper and returns it
// Parameters:
//   restart WrapperRestartMode: indicates if and when the
//     process must restart automatically
//   hideStdOut bool: if true the stdout logs of the wrapped
//    process are hidden from the logger
//   hideStdErr bool: if true the stderr logs of the wrapped
//     process are hidden from the logger
//   failOnStdErr bool : if true the wrapped process is marked
//     as failed if a log is detected on its stderr
//   timeout time.Duration: indicates how much time we must wait
//     for the wrapped process to exit gracefully; if this time
//     expires and the process is still running, then we send
//     a SIGKILL signal to it
//   path: the path of the process executable
//   arg: a list of arguments for the process
// Return values:
//   system.WrapperHandler
func NewWrapperHandler(config WrapperConfiguration, arg ...string) WrapperHandler {
	p := &wrapperHandler{
		arg:             arg,
		failOnStdErr:    config.FailOnStdErr,
		hideStdErr:      config.HideStdErr,
		hideStdOut:      config.HideStdOut,
		path:            config.Path,
		restartMode:     config.RestartMode,
		restartInterval: 1 * time.Second,
		timeout:         config.Timeout,
	}

	return p
}

// Start executes the wrapped process, an returns the channels
// on which it send events to the main process
// Parameters:
//   ctx context.Context: must be a Context created WithCancel,
//     when the ctx.cancelFunc() method is called, the wrapped
//     process will be terminated
// Return values:
//   <-chan WrapperData: is a channel sending an event every
//     time the wrapped process changes its status
//   <-chan struct{}: this channel will be closed when the
//     wrapped process is completely terminated; the main
//     process should wait for it to be closed, then exit
func (p *wrapperHandler) Start(ctx context.Context) (<-chan WrapperData, <-chan struct{}) {
	chanWrapperData := make(chan WrapperData)
	chanWrapperDone := make(chan struct{})

	go p.do(ctx, chanWrapperData, chanWrapperDone)

	return chanWrapperData, chanWrapperDone
}

// initCmdLogWrappers is an internal method used to initialize
// the behaviour of the wrapped command's logs
// Parameters:
//   cmd *exec.Cmd: the wrapped command
//   signalOnErrors bool: if true, the wrapper will send a
//     signal on the loggedErrors Channel if an error is
//     detected on its stderr
//   loggedErrors chan<- int: the channel where the signal
//     will be sent when an error is detected on the wrapped
//     process' stderr; the channel will receive the number
//     of bytes written on stderr
func (p *wrapperHandler) initCmdLogWrappers(cmd *exec.Cmd, signalOnErrors bool, loggedErrors chan<- int) {
	if !p.hideStdOut {
		cmd.Stdout = logger.NewLogInfoWriter("wrapped log")
	}

	if !p.hideStdErr {
		if signalOnErrors {
			cmd.Stderr = logger.SignalOnWrite(loggedErrors, logger.NewLogErrorWriter("wrapped log"))
		} else {
			cmd.Stderr = logger.NewLogErrorWriter("wrapped log")
		}
	}
}

// run executes a new instance of the wrapped process and starts
// the goroutine responsible to wait for it to end.
// Parameters:
//   ctx context.Context: must be a Context created WithCancel,
//     when the ctx.cancelFunc() method is called, the wrapped
//     process will be terminated
//   runError chan<- error: this channel will receive the
//     error of the wrapped process when it exit
//   signalOnErrors bool: if true, the wrapper will send a
//     signal on the loggedErrors Channel if an error is
//     detected on its stderr
//   loggedErrors chan<- int: the channel where the signal
//     will be sent when an error is detected on the wrapped
//     process' stderr; the channel will receive the number
//     of bytes written on stderr
// Return values:
//   error: this function will return an error if the wrapped
//     process cannot be started for any reason
func (p *wrapperHandler) run(ctx context.Context, runError chan<- error, signalOnErrors bool, loggedErrors chan<- int) error {
	cmd := exec.Command(p.path, p.arg...)

	p.initCmdLogWrappers(cmd, signalOnErrors, loggedErrors)

	err := cmd.Start()
	if err != nil {
		go func() {
			logger.Errorf("cannot start the wrapped process %s: %s", p.path, err)
			runError <- err
		}()

		return err
	}

	var waitDone chan struct{}

	var waitTimeout *time.Timer

	waitDone = make(chan struct{})

	go func() {
		var done bool
		select {
		case <-ctx.Done():
			_ = cmd.Process.Signal(os.Interrupt)
			waitTimeout = time.NewTimer(p.timeout)
		case <-waitDone:
			done = true
		}

		if !done {
			select {
			case <-waitTimeout.C:
				_ = cmd.Process.Kill()
			case <-waitDone:
				if waitTimeout != nil {
					_ = waitTimeout.Stop()
				}

				done = true
			}
		}

		if !done {
			<-waitDone
		}
	}()

	go func() {
		logger.Debugf("waiting for the wrapped process %s to exit", p.path)
		runError <- cmd.Wait()

		if waitDone != nil {
			close(waitDone)
		}
	}()

	return nil
}

// parseRunError receive the error from the wrapped process, and extract all
// the information needed
// Parameters:
//   processErr error: the error returned from the wrapped process
// Return values:
//   status WrapperStatus: the new status of the wrapped process, based on
//     the value of err
//   processExitStatus int: the error code returned from the wrapped
//     process when it ended
//   err error: an error indicating
func (p *wrapperHandler) parseRunError(processErr error) (status WrapperStatus, processExitStatus int, err error) {
	if processErr != nil {
		status = WrapperStatusError

		if exitError, ok := processErr.(*exec.ExitError); ok {
			if waitStatus, ok := exitError.Sys().(syscall.WaitStatus); ok {
				processExitStatus = waitStatus.ExitStatus()
				err = NewProcessExitStatusError(processExitStatus)
				logger.Errorf("wrapped process exited with status: %d", byte(processExitStatus))

				return
			}

			err = exitError

			return
		}

		err = processErr

		return
	}

	status = WrapperStatusStopped

	logger.Debugf("wrapped process completed without errors")

	return
}

func (p *wrapperHandler) doRestart(ctx context.Context, runError chan error, loggedErrors chan int) (status WrapperStatus) {
	// when a signal i received from the restartTimer, the wrapped
	// process is started
	if err := p.run(ctx, runError, p.failOnStdErr, loggedErrors); err != nil {
		status = WrapperStatusError

		return
	}

	status = WrapperStatusRunning

	logger.Infof("wrapped process %s started", p.path)

	return
}

// canRestart return checks if the wrapped process can be restarted,
// based on the current status of the environment
// it takes the state of the context and the exit code of the process
// as input values.
func (p *wrapperHandler) canRestart(contextIsCanceling bool, exitStatus int) bool {
	switch p.restartMode {
	case WrapperRestartNever:
		return false
	case WrapperRestartOnError:
		return !contextIsCanceling && exitStatus != 0
	case WrapperRestartAlways:
		return !contextIsCanceling
	}

	return false
}

func (p *wrapperHandler) do(ctx context.Context, chanWrapperData chan<- WrapperData, chanWrapperDone chan<- struct{}) {
	defer close(chanWrapperDone)
	defer close(chanWrapperData)

	var processError error

	var processExitStatus int

	var status WrapperStatus

	defer func() { chanWrapperData <- WrapperData{status, processError, true} }()

	runError := make(chan error)
	defer close(runError)

	loggedErrors := make(chan int)
	defer close(loggedErrors)

	restartTimer := time.NewTimer(0)

	var contextDone bool

	for {
		select {
		case <-restartTimer.C:
			if contextDone {
				logger.Debugf("cannot execute the wrapped process, the context is closing")
				return
			}

			status = p.doRestart(ctx, runError, loggedErrors)
			chanWrapperData <- WrapperData{status, nil, false}

		case <-ctx.Done():
			if contextDone {
				continue
			}

			contextDone = true

			logger.Debugf("received the signal to close the wrapped process context")

			if restartTimer.Stop() {
				logger.Debugf("wrapped process is scheduled, but not started yet, exit now")
				return
			}

		case n := <-loggedErrors:
			status = WrapperStatusError
			chanWrapperData <- WrapperData{status, nil, false}

			logger.Debugf("wrapped process logged an error: %d bytes", n)

		case err := <-runError:
			status, processExitStatus, processError = p.parseRunError(err)
			chanWrapperData <- WrapperData{status, nil, false}

			if p.canRestart(contextDone, processExitStatus) {
				logger.Debugf("the wrapped process will restart in %d seconds...", p.restartInterval/time.Second)
				restartTimer = time.NewTimer(p.restartInterval)
				p.restartInterval *= 2
			} else {
				logger.Debugf("wrapped process is completed, exiting now...")
				return
			}
		}
	}
}
