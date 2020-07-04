package main

import (
	"os"

	"github.com/gandalfmagic/liveness-wrapper/cmd"
	"github.com/gandalfmagic/liveness-wrapper/internal/system"
)

func main() {
	if err := cmd.RootCmd.Execute(); err != nil {
		if e, ok := err.(system.ProcessExitStatusError); ok {
			os.Exit(e.ExitStatus())
		} else {
			os.Exit(1)
		}
	}
}
