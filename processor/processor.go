package processor

import (
	"github.com/assada/go-worker/config"
	log "github.com/sirupsen/logrus"
	"time"
)

const (
	ExitCodeOK    int = 0
	ExitCodeError     = 10 + iota
)

type Processor struct {
	config config.Config
	error  chan error
	done   chan bool
	log    *log.Logger
}

func NewProcessor(config *config.Config, logger *log.Logger, errorCh chan error, doneCh chan bool) (*Processor, error) {
	logger.Info("(processor) creating new processor")

	processor := &Processor{
		config: *config,
		error:  errorCh,
		done:   doneCh,
		log:    logger,
	}

	processor.init()

	return processor, nil
}

func (p *Processor) init() {
	//TODO: Init
}

func (p *Processor) Process() int {
	//TODO: Implement this method

	//SAMPLE Code:
	sum := 0
	for i := 0; i < 50000000; i++ {
		p.log.Info("(processor) %v", time.Now().UnixNano())
		sum++
	}

	if sum >= 50000000 {
		p.done <- true
	}

	return ExitCodeOK
}
