package main

import (
	"os"

	"github.com/gandalfmagic/liveness-wrapper/cmd"
)

func main() {
	if err := cmd.RootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
