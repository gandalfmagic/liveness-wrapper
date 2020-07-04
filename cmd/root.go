package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	"github.com/gandalfmagic/liveness-wrapper/internal"
	"github.com/gandalfmagic/liveness-wrapper/internal/http"
	"github.com/gandalfmagic/liveness-wrapper/internal/system"
	"github.com/gandalfmagic/liveness-wrapper/pkg/logger"

	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	// Used for flags.
	config      string // config file location
	showVersion bool   // whether to print version info or not

	// to be populated by linker
	version = "v0.0.0"
	commit  = "HEAD"

	// HoarderCmd ...
	RootCmd = &cobra.Command{
		Long:             internal.RootDescriptionLong,
		PersistentPreRun: persistentPreRun,
		RunE:             run,
		Short:            internal.RootDescriptionShort,
		SilenceUsage:     true,
		Use:              internal.RootName,
	}
)

func init() {
	// cli flags
	RootCmd.PersistentFlags().StringP("process-path", "p", "", "Path of the wrapped process executable")
	RootCmd.PersistentFlags().BoolP("process-restart-always", "r", false, "Always restart the wrapped process when it ends")
	RootCmd.PersistentFlags().BoolP("process-restart-on-error", "e", false, "Restart the wrapped process only when it fails")
	RootCmd.PersistentFlags().StringSlice("process-args", nil, "Comma separated list of arguments for the wrapped process")
	RootCmd.PersistentFlags().Bool("process-hide-stdout", false, "Hide the stdout of the wrapped process from the logs")
	RootCmd.PersistentFlags().Bool("process-hide-stderr", false, "Hide the stderr of the wrapped process from the logs")
	RootCmd.PersistentFlags().Bool("process-fail-on-stderr", false, "Mark the wrapped process as failed if it writes logs on stderr")
	RootCmd.PersistentFlags().StringP("server-address", "a", ":6060", "Bind address for the http server")
	RootCmd.PersistentFlags().DurationP("server-ping-timeout", "t", 10*time.Minute, "Ping endpoint timeout, use 0 to disable")
	RootCmd.PersistentFlags().String("log-level", "WARN", "Output level of logs (TRACE, DEBUG, INFO, WARN, ERROR, FATAL)")

	// cli-only flags
	RootCmd.Flags().StringVarP(&config, "config", "c", "", "Path to config file (with extension)")
	RootCmd.Flags().BoolVarP(&showVersion, "version", "v", false, "Display the current version of this CLI")

	// bind config to cli flags
	_ = viper.BindPFlag("process.path", RootCmd.PersistentFlags().Lookup("process-path"))
	_ = viper.BindPFlag("process.restart-always", RootCmd.PersistentFlags().Lookup("process-restart-always"))
	_ = viper.BindPFlag("process.restart-on-error", RootCmd.PersistentFlags().Lookup("process-restart-on-error"))
	_ = viper.BindPFlag("process.args", RootCmd.PersistentFlags().Lookup("process-args"))
	_ = viper.BindPFlag("process.hide-stdout", RootCmd.PersistentFlags().Lookup("process-hide-stdout"))
	_ = viper.BindPFlag("process.hide-stderr", RootCmd.PersistentFlags().Lookup("process-hide-stderr"))
	_ = viper.BindPFlag("process.fail-on-stderr", RootCmd.PersistentFlags().Lookup("process-fail-on-stderr"))

	_ = viper.BindPFlag("server.address", RootCmd.PersistentFlags().Lookup("server-address"))
	_ = viper.BindPFlag("server.ping-timeout", RootCmd.PersistentFlags().Lookup("server-ping-timeout"))
	_ = viper.BindPFlag("log.level", RootCmd.PersistentFlags().Lookup("log-level"))
}

func convertToAbsProjectDirectory() {

	dir := viper.GetString("project.directory")
	if dir != "" {
		absDir, err := filepath.Abs(dir)
		logger.CheckFatal("cannot get absolute path", err)

		if absDir != dir {
			viper.Set("project.directory", absDir)
		}
	}
}

func printVersion() {

	if showVersion {
		fmt.Printf("liveness-wrapper %s -- %s\n", version, commit)
		os.Exit(0)
	}
}

func persistentPreRun(_ *cobra.Command, _ []string) {

	printVersion()
	logger.Configure(internal.RootName, viper.GetString("log.level"))
	readConfig()
	convertToAbsProjectDirectory()
}

func readConfig() {

	if config != "" {
		// Use config file from the flag
		viper.SetConfigFile(config)
		logger.Info("reading configuration from %s", config)
	} else {
		// Find home directory
		home, err := homedir.Dir()
		logger.CheckFatal("cannot find home directory", err)

		// Search config in home directory
		viper.SetConfigType("yaml")
		viper.SetConfigName(internal.ConfigurationFile)
		viper.AddConfigPath(home)
		config = filepath.Join(home, internal.ConfigurationFile)
		logger.Info("reading configuration from %s.yaml", config)
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		logger.Info("the configuration file doesn't exist")
	}
}

func getRestartMode() system.WrapperRestartMode {

	if viper.GetBool("process.restart-always") {
		return system.WrapperRestartAlways
	}

	if viper.GetBool("process.restart-on-error") {
		return system.WrapperRestartOnError
	}

	return system.WrapperRestartNever
}

func do(processStatus <-chan system.WrapperStatus, updateAlive chan<- bool) chan<- struct{} {
	done := make(chan struct{})

	go func() {
		defer close(updateAlive)
		for {
			select {
			case s := <-processStatus:
				// change the readiness state based on the process status
				switch s {
				case system.WrapperStatusError:
					updateAlive <- false
				case system.WrapperStatusRunning:
					updateAlive <- true
				case system.WrapperStatusStopped:
					updateAlive <- false
				}

			case <-done:
				return
			}
		}
	}()

	return done
}

func wait(cancelFunc context.CancelFunc, serverDone <-chan struct{}, processDone <-chan error, done chan<- struct{}) error {
	// create the channel to catch SIGINT signal
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	defer close(c)

	defer close(done)
	for {
		select {
		case <-c:
			// if c receives data, then the a SIGINT signal was
			// sent by the system, so we stop the process, and
			// the http server and finally we return
			cancelFunc()
			<-serverDone
			return <-processDone

		case exitStatus := <-processDone:
			// if processDone is closed, then the process has stopped
			// (without being restarted), so we stop the http server
			// and then we return
			cancelFunc()
			<-serverDone
			return exitStatus

		case <-serverDone:
			// if serverDone is closed, then the http server has stopped
			// because of an error, so we stop the wrapped process and
			// then we return
			cancelFunc()
			return <-processDone
		}

	}
}

func run(_ *cobra.Command, _ []string) error {
	ctx, cancelFunc := context.WithCancel(context.Background())

	// create the http server
	server := http.NewServer(ctx, viper.GetString("server.address"), viper.GetDuration("server.ping-timeout"))
	updateAlive, serverDone := server.Start()

	// start the wrapped process
	exitStatus := make(chan error)
	defer close(exitStatus)
	process := system.NewWrapperHandler(ctx, getRestartMode(), viper.GetBool("process.hide-stdout"),
		viper.GetBool("process.hide-stderr"), viper.GetBool("process.fail-on-stderr"),
		viper.GetString("process.path"), viper.GetStringSlice("process.args")...)
	processStatus := process.Start(exitStatus)

	done := do(processStatus, updateAlive)

	return wait(cancelFunc, serverDone, exitStatus, done)
}
