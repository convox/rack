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
	log := p.logger("workerConverge")

	if _, err := p.router.RackGet(p.Rack); err != nil {
		if err := p.routerRegister(); err != nil {
			log.At("register").Error(err)
			return err
		}
	}

	if err := p.idle(); err != nil {
		log.At("idle").Error(err)
		return err
	}

	apps, err := p.AppList()
	if err != nil {
		log.At("list").Error(err)
		return err
	}

	for _, a := range apps {
		if err := p.converge(a.Name); err != nil {
			log.At("converge").Append("app=%s", a.Name).Error(err)
			continue
		}
	}

	return nil
}
