package main

import (
	"flag"
	"fmt"
	"github.com/assada/go-worker/config"
	manager "github.com/assada/go-worker/runner"
	"github.com/assada/go-worker/signals"
	"github.com/assada/go-worker/version"
	log "github.com/sirupsen/logrus"
	lSyslog "github.com/sirupsen/logrus/hooks/syslog"
	"io"
	"io/ioutil"
	"log/syslog"
	"os"
	"os/signal"
	"strings"
	"sync"
)

var syslogPriorityMap = map[string]syslog.Priority{
	"DEBUG": syslog.LOG_INFO,
	"INFO":  syslog.LOG_NOTICE,
	"WARN":  syslog.LOG_WARNING,
	"ERROR": syslog.LOG_ERR,
}

var LogLevelMap = map[string]log.Level{
	"DEBUG": log.DebugLevel,
	"INFO":  log.InfoLevel,
	"WARN":  log.WarnLevel,
	"ERROR": log.ErrorLevel,
}

var logger *log.Logger

const (
	ExitCodeOK int = 0
	ExitCodeInterrupt
	ExitCodeParseFlagsError
	ExitCodeRunnerError
	ExitCodeConfigError
)

type Cli struct {
	sync.Mutex

	outStream, errStream io.Writer

	signalCh chan os.Signal

	stopCh chan struct{}

	stopped bool

	logger *log.Logger
}

func NewCli(out, err io.Writer) *Cli {
	return &Cli{
		outStream: out,
		errStream: err,
		signalCh:  make(chan os.Signal, 1),
		stopCh:    make(chan struct{}),
	}
}

func (service *Cli) setup(conf *config.Config) (*config.Config, error) {

	priority, ok := syslogPriorityMap[strings.ToUpper(*conf.LogLevel)]
	if !ok {
		priority = syslog.LOG_INFO
	}
	logLevel, ok := LogLevelMap[strings.ToUpper(*conf.LogLevel)]
	if !ok {
		logLevel = log.InfoLevel
	}

	logger = log.New()
	logger.SetLevel(logLevel)
	logger.SetFormatter(&log.JSONFormatter{})
	hook, err := lSyslog.NewSyslogHook("", "", priority, "")

	if err == nil {
		logger.Hooks.Add(hook)
	}

	return conf, nil
}

func (cli *Cli) Run(args []string) int {
	config, isVersion, err := cli.ParseFlags(args[1:])

	if err != nil {
		if err == flag.ErrHelp {
			_, _ = fmt.Fprintf(cli.errStream, usage, version.Name)
			return 0
		}
		_, _ = fmt.Fprintln(cli.errStream, err.Error())
		return ExitCodeParseFlagsError
	}

	cliConfig := config.Copy()

	config.Finalize()

	config, err = cli.setup(config)
	if err != nil {
		return logError(err, ExitCodeConfigError)
	}

	logger.Info(version.HumanVersion)

	if isVersion {
		logger.Debug("(cli) version flag was given, exiting now")
		logger.Error("%s\n", version.HumanVersion)
		return ExitCodeOK
	}

	runner, err := manager.NewRunner(config, logger)

	if err != nil {
		return logError(err, ExitCodeRunnerError)
	}

	go runner.Start()

	signal.Notify(cli.signalCh)

	for {
		select {
		case err := <-runner.ErrCh:
			code := ExitCodeRunnerError
			if typed, ok := err.(manager.ErrExitable); ok {
				code = typed.ExitStatus()
			}
			return logError(err, code)
		case <-runner.DoneCh:
			logger.Info("(cli) received finish")
			runner.Stop()
			return ExitCodeOK
		case s := <-cli.signalCh:
			logger.Debug("(cli) receiving signal %q", s)

			switch s {
			case *config.ReloadSignal:
				logger.Debug("Reloading configuration...\n")
				runner.Stop()

				config = cliConfig
				if err != nil {
					return logError(err, ExitCodeConfigError)
				}
				config.Finalize()

				config, err = cli.setup(config)
				if err != nil {
					return logError(err, ExitCodeConfigError)
				}

				runner, err = manager.NewRunner(config, logger)
				if err != nil {
					return logError(err, ExitCodeRunnerError)
				}
				go runner.Start()
			case *config.KillSignal:
				logger.Error("Cleaning up...\n")
				runner.Stop()
				return ExitCodeInterrupt
			default:
				runner.Stop()
				return ExitCodeInterrupt
			}
		case <-cli.stopCh:
			return ExitCodeOK
		}
	}
}

func (cli *Cli) ParseFlags(args []string) (*config.Config, bool, error) {
	var isVersion bool

	c := config.DefaultConfig()

	configPaths := make([]string, 0, 6)

	flags := flag.NewFlagSet(version.Name, flag.ContinueOnError)
	flags.SetOutput(ioutil.Discard)
	flags.Usage = func() {}

	flags.Var((funcVar)(func(s string) error {
		configPaths = append(configPaths, s)
		return nil
	}), "config", "")
	flags.Var((funcVar)(func(s string) error {
		sig, err := signals.Parse(s)
		if err != nil {
			return err
		}
		c.KillSignal = config.Signal(sig)
		return nil
	}), "kill-signal", "")

	flags.Var((funcVar)(func(s string) error {
		c.LogLevel = config.String(s)
		return nil
	}), "log-level", "")

	flags.Var((funcVar)(func(s string) error {
		c.PidFile = config.String(s)
		return nil
	}), "pid-file", "")

	flags.Var((funcVar)(func(s string) error {
		sig, err := signals.Parse(s)
		if err != nil {
			return err
		}
		c.ReloadSignal = config.Signal(sig)
		return nil
	}), "reload-signal", "")

	flags.BoolVar(&isVersion, "v", false, "")
	flags.BoolVar(&isVersion, "version", false, "")

	if err := flags.Parse(args); err != nil {
		return nil, false, err
	}

	args = flags.Args()
	if len(args) > 0 {
		return nil, false, fmt.Errorf("cli: extra args: %q", args)
	}

	return c, isVersion, nil
}

func logError(err error, status int) int {
	logger.Error("[ERR] (cli) %s", err)
	return status
}

const usage = `Usage: %s [options]

  Usage description

Options:
  -kill-signal=<signal>
      Signal to listen to gracefully terminate the process
  -log-level=<level>
      Set the logging level - values are "debug", "info", "warn", and "err"
  -pid-file=<path>
      Path on disk to write the PID of the process
  -reload-signal=<signal>
      Signal to listen to reload configuration
  -v, -version
      Print the version of this daemon
`
