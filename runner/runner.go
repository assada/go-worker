package runner

import (
	"encoding/json"
	"fmt"
	"github.com/assada/go-worker/config"
	"github.com/assada/go-worker/processor"
	log "github.com/sirupsen/logrus"
	"io"
	"os"
	"sync"
	"time"
)

type Runner struct {
	ErrCh                chan error
	DoneCh               chan bool
	ticker               *time.Ticker
	config               *config.Config
	outStream, errStream io.Writer
	inStream             io.Reader
	stopLock             sync.Mutex
	stopped              bool
	log                  *log.Logger
}

func NewRunner(config *config.Config, logger *log.Logger) (*Runner, error) {
	logger.Info("(runner) creating new runner")

	runner := &Runner{
		config: config,
		ticker: time.NewTicker(time.Millisecond),
		log: logger,
	}

	if err := runner.init(); err != nil {
		return nil, err
	}

	return runner, nil
}

func (r *Runner) Start() {
	r.log.Info("(runner) starting")

	if err := r.storePid(); err != nil {
		r.ErrCh <- err
		return
	}

	if err := r.Run(); err != nil {
		r.ErrCh <- err
		return
	}

	pr, _ := processor.NewProcessor(r.config, r.log, r.ErrCh, r.DoneCh)

	for {
		pr.Process()
		select {
		case <-r.ErrCh:
			r.log.Error("(runner) received error")
			return
		case <-r.DoneCh:
			r.log.Error("(runner) received finish")
			return
		}
	}

}

func (r *Runner) Stop() {
	r.stopLock.Lock()
	defer r.stopLock.Unlock()

	if r.stopped {
		return
	}

	r.log.Info("(runner) stopping")

	if err := r.deletePid(); err != nil {
		r.log.Warn("(runner) could not remove pid at %q: %s",
			config.StringVal(r.config.PidFile), err)
	}

	r.stopped = true

	close(r.DoneCh)
}

func (r *Runner) Run() error {
	r.log.Debug("(runner) initiating run")

	return nil
}

func (r *Runner) init() error {
	r.config = config.DefaultConfig().Merge(r.config)
	r.config.Finalize()

	result, err := json.Marshal(r.config)
	if err != nil {
		return err
	}
	r.log.Debug("(runner) final config: %s", result)

	r.inStream = os.Stdin
	r.outStream = os.Stdout
	r.errStream = os.Stderr

	r.ErrCh = make(chan error)
	r.DoneCh = make(chan bool)

	return nil
}

func (r *Runner) storePid() error {
	path := config.StringVal(r.config.PidFile)
	if path == "" {
		return nil
	}

	r.log.Infof("creating pid file at %q", path)

	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	if err != nil {
		return fmt.Errorf("runner: could not open pid file: %s", err)
	}
	defer f.Close()

	pid := os.Getpid()
	_, err = f.WriteString(fmt.Sprintf("%d", pid))
	if err != nil {
		return fmt.Errorf("runner: could not write to pid file: %s", err)
	}
	return nil
}

func (r *Runner) deletePid() error {
	path := config.StringVal(r.config.PidFile)
	if path == "" {
		return nil
	}

	r.log.Debug("Removing pid file at %q", path)

	stat, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("runner: could not remove pid file: %s", err)
	}
	if stat.IsDir() {
		return fmt.Errorf("runner: specified pid file path is directory")
	}

	err = os.Remove(path)
	if err != nil {
		return fmt.Errorf("runner: could not remove pid file: %s", err)
	}
	return nil
}

func (r *Runner) SetOutStream(out io.Writer) {
	r.outStream = out
}

func (r *Runner) SetErrStream(err io.Writer) {
	r.errStream = err
}