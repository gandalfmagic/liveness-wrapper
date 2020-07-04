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

type WrapperHandler interface {
	Start(chan<- error) <-chan WrapperStatus
}

type wrapperHandler struct {
	arg          []string
	ctx          context.Context
	failOnStdErr bool
	hideStdErr   bool
	hideStdOut   bool
	path         string
	restart      WrapperRestartMode
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
		arg:          arg,
		ctx:          ctx,
		failOnStdErr: failOnStdErr,
		hideStdErr:   hideStdErr,
		hideStdOut:   hideStdOut,
		path:         path,
		restart:      restart,
	}

	return p
}

// Start executes the wrapped process, an returns the channels
// on which it send events to the main process
// Return values:
//   <-chan WrapperStatus: is a channel sending an event every
//     time the wrapped process changes its status
//   <-chan int: is a channel sending the exit code of the
//     wrapper process when it ends; when we receive a message
//     on this channel the wrapped process finally ended, the
//     main process of the object will returns and all the
//     channels will be closed
func (p *wrapperHandler) Start(exitStatus chan<- error) <-chan WrapperStatus {
	wrapperStatus := make(chan WrapperStatus)

	go p.do(wrapperStatus, exitStatus)

	return wrapperStatus
}

// run executes a new instance of the wrapped process and
// starts the goroutine responsible to wait for it to end
func (p *wrapperHandler) run(runError chan<- error, signalOnErrors bool, loggedErrors chan<- int) {

	// instantiate the new process ans starts it
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
		logger.Error("cannot start the wrapped process %s: %s", p.path, err)
		// send with a separate goroutine to avoid a deadlock
		go func() {
			runError <- err
		}()

		return
	}

	go func() {
		logger.Debug("waiting for the wrapped process %s to exit", p.path)
		runError <- cmd.Wait()
	}()
}

// canRestart return checks if the wrapped process can be restarted,
// based on the current status of the environment
// it takes the state of the context and the exit code of the process
// as input values
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

func (p *wrapperHandler) do(wrapperStatus chan<- WrapperStatus, chanExitStatus chan<- error) {

	var exitStatus int
	defer func() {
		chanExitStatus <- NewProcessExitStatusError(exitStatus)
	}()

	defer close(wrapperStatus)

	var contextIsCanceling bool
	restartInterval := 1 * time.Second

	runError := make(chan error)
	defer close(runError)

	loggedErrors := make(chan int)
	defer close(loggedErrors)

	// we schedule the process to start the first time (without delay)
	restartTimer := time.NewTimer(0)

	for {
		select {
		case <-restartTimer.C:
			// when a signal i received from the restartTimer, the wrapped
			// process is executed
			if !contextIsCanceling {
				p.run(runError, p.failOnStdErr, loggedErrors)
				wrapperStatus <- WrapperStatusRunning
				logger.Info("the wrapped process %s has started", p.path)
			} else {
				logger.Debug("the wrapped process context is canceling, not starting now")
			}

		case <-p.ctx.Done():
			// this channel receives the signal from the context.CancelFunc() method,
			// so when the main process is ending  we don't need to explicitly kill
			// the wrapped process, because it  receives the same signal too, but we
			// have to be sure it won't be started anymore
			logger.Debug("the wrapped process context is canceled")

			// from now, we must avoid to schedule a new execution of the process
			// when runError channel sends a signal
			contextIsCanceling = true

			// if a restart is already scheduled, we stop it now
			if restartTimer.Stop() {
				// if a restart was scheduled, it means that the process is not running and
				// we won't receive anything from runError channel, so we return immediately
				logger.Debug("the wrapped process was scheduled, but not started, exiting...")
				return
			}

		case n := <-loggedErrors:
			wrapperStatus <- WrapperStatusError
			logger.Debug("the wrapped process logged an error: %d bytes", n)

		case err := <-runError:
			// this channel receives the result error from the cmd.Wait() in the run method
			if err != nil {
				if !contextIsCanceling {
					wrapperStatus <- WrapperStatusError
				}

				if exitError, ok := err.(*exec.ExitError); ok {
					if status, ok := exitError.Sys().(syscall.WaitStatus); ok {
						exitStatus = status.ExitStatus()
						logger.Info("wrapped process exited with status: %d", byte(exitStatus))
					}
				} else {
					logger.Error("wrapped process exited with error: %v", err)
				}
			} else {
				if !contextIsCanceling {
					wrapperStatus <- WrapperStatusStopped
				}

				logger.Debug("the wrapped process has ended without errors")
			}

			// check if the process must be restarted or not
			if p.canRestart(contextIsCanceling, exitStatus) {
				logger.Debug("the wrapped process will restart in %d seconds...", restartInterval/time.Second)
				restartTimer = time.NewTimer(restartInterval)
				restartInterval *= 2
			} else {
				logger.Debug("the wrapped process is completed, exiting now...")
				return
			}
		}
	}
}
