package system

import "fmt"

type ProcessExitStatusError interface {
	Error() string
	ExitStatus() int
}

type processExitStatusError struct {
	exitStatus byte
}

func NewProcessExitStatusError(exitStatus int) ProcessExitStatusError {
	return &processExitStatusError{
		exitStatus: byte(exitStatus),
	}
}

func (p *processExitStatusError) Error() string {
	return fmt.Sprintf("the process ended with exit status %d", p.exitStatus)
}

func (p *processExitStatusError) ExitStatus() int {
	return int(p.exitStatus)
}
