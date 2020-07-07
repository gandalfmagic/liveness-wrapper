package system

import (
	"context"
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

type WrapperData struct {
	WrapperStatus WrapperStatus
	Err           error
	Done          bool
}

type WrapperHandler interface {
	Start() <-chan WrapperData
}

type wrapperHandler struct {
	arg             []string
	ctx             context.Context
	failOnStdErr    bool
	hideStdErr      bool
	hideStdOut      bool
	path            string
	restart         WrapperRestartMode
	restartInterval time.Duration
}

// NewWrapperStatus creates a new process wrapper and returns it
// Parameters:
//   ctx: a context.Context created WithCancel, when the
//     ctx.cancelFunc() method is called, the wrapped process
//     will be terminated
//   mustRestart: bool value indicating if the process must be
//     restarted automatically in case it stops with an error
//   path: the path of the process executable
//   arg: a list of arguments for the process
// Return values:
//   system.WrapperHandler
func NewWrapperHandler(ctx context.Context, restart WrapperRestartMode, hideStdOut, hideStdErr, failOnStdErr bool,
	path string, arg ...string) WrapperHandler {
	p := &wrapperHandler{
		arg:             arg,
		ctx:             ctx,
		failOnStdErr:    failOnStdErr,
		hideStdErr:      hideStdErr,
		hideStdOut:      hideStdOut,
		path:            path,
		restart:         restart,
		restartInterval: 1 * time.Second,
	}

	return p
}

// Start executes the wrapped process, an returns the channels
// on which it send events to the main process
// Return values:
//   <-chan WrapperStatus: is a channel sending an event every
//     time the wrapped process changes its status
//   <-chan error: is a channel sending the error of the
//     wrapper process when it ends; when we receive a message
//     on this channel the wrapped process finally ended, the
//     main process of the object will returns and all the
//     channels will be closed
func (p *wrapperHandler) Start() <-chan WrapperData {
	chanWrapperData := make(chan WrapperData)
	go p.do(chanWrapperData)

	return chanWrapperData
}

// run executes a new instance of the wrapped process and
// starts the goroutine responsible to wait for it to end.
func (p *wrapperHandler) run(runError chan<- error, signalOnErrors bool, loggedErrors chan<- int) error {
	cmd := exec.CommandContext(p.ctx, p.path, p.arg...)

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

	err := cmd.Start()
	if err != nil {
		go func() {
			logger.Errorf("cannot start the wrapped process %s: %s", p.path, err)
			runError <- err
		}()

		return err
	}

	go func() {
		logger.Debugf("waiting for the wrapped process %s to exit", p.path)
		runError <- cmd.Wait()
	}()

	return nil
}

func (p *wrapperHandler) doRunError(err error) (status WrapperStatus, processExitStatus int, processError error) {
	if err != nil {
		status = WrapperStatusError

		if exitError, ok := err.(*exec.ExitError); ok {
			if status, ok := exitError.Sys().(syscall.WaitStatus); ok {
				processExitStatus = status.ExitStatus()
				logger.Infof("wrapped process exited with status: %d", byte(processExitStatus))
			}
		} else {
			processError = err
		}
	} else {
		status = WrapperStatusStopped

		logger.Debugf("the wrapped process has ended without errors")
	}

	return
}

func (p *wrapperHandler) doRestart(runError chan error, loggedErrors chan int) (status WrapperStatus) {
	// when a signal i received from the restartTimer, the wrapped
	// process is started
	if err := p.run(runError, p.failOnStdErr, loggedErrors); err != nil {
		status = WrapperStatusError

		return
	}

	status = WrapperStatusRunning

	logger.Infof("the wrapped process %s has started", p.path)

	return
}

// canRestart return checks if the wrapped process can be restarted,
// based on the current status of the environment
// it takes the state of the context and the exit code of the process
// as input values.
func (p *wrapperHandler) canRestart(contextIsCanceling bool, exitStatus int) bool {
	switch p.restart {
	case WrapperRestartNever:
		return false
	case WrapperRestartOnError:
		return !contextIsCanceling && exitStatus != 0
	case WrapperRestartAlways:
		return !contextIsCanceling
	}

	return false
}

func (p *wrapperHandler) do(chanWrapperData chan<- WrapperData) {

	defer close(chanWrapperData)

	var processError error

	var processExitStatus int

	var status WrapperStatus

	defer func() {
		if processExitStatus != 0 {
			chanWrapperData <- WrapperData{status, NewProcessExitStatusError(processExitStatus), true}
			return
		}
		chanWrapperData <- WrapperData{status, processError, true}
	}()

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

			status = p.doRestart(runError, loggedErrors)
			chanWrapperData <- WrapperData{status, nil, false}

		case <-p.ctx.Done():
			if contextDone {
				logger.Debugf("the context is already closing")
				continue
			}

			contextDone = true

			logger.Debugf("received the signal to close the wrapped process context")

			if restartTimer.Stop() {
				logger.Debugf("the wrapped process was scheduled, but not started, exiting...")
				return
			}

		case n := <-loggedErrors:
			status = WrapperStatusError
			chanWrapperData <- WrapperData{status, nil, false}

			logger.Debugf("the wrapped process logged an error: %d bytes", n)

		case err := <-runError:
			status, processExitStatus, processError = p.doRunError(err)
			chanWrapperData <- WrapperData{status, nil, false}

			if p.canRestart(contextDone, processExitStatus) {
				logger.Debugf("the wrapped process will restart in %d seconds...", p.restartInterval/time.Second)
				restartTimer = time.NewTimer(p.restartInterval)
				p.restartInterval *= 2
			} else {
				logger.Debugf("the wrapped process is completed, exiting now...")
				return
			}
		}
	}
}
