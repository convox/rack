package local

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/pkg/errors"
)

func (p *Provider) Workers() error {
	log := p.logger("Workers")

	terminate := make(chan os.Signal, 1)
	signal.Notify(terminate, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-terminate
		if err := p.shutdown(); err != nil {
			log.Error(errors.WithStack(err))
		}
	}()

	if !p.Test {
		go func() {
			for {
				time.Sleep(10 * time.Second)

				if err := p.workerConverge(); err != nil {
					log.Error(errors.WithStack(err))
				}
			}
		}()
	}

	return nil
}

func (p *Provider) workerConverge() error {
	apps, err := p.AppList()
	if err != nil {
		return err
	}

	for _, a := range apps {
		if err := p.converge(a.Name); err != nil {
			continue
		}
	}

	if err := p.prune(); err != nil {
		return err
	}

	return nil
}
