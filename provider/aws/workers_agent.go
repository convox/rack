package aws

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/convox/logger"
)

func (p *AWSProvider) workerAgent() {
	log := logger.New("ns=workers.agent")

	for {
		time.Sleep(1 * time.Minute)

		if err := p.scaleAgents(); err != nil {
			log.Error(err)
			continue
		}
	}
}

func (p *AWSProvider) scaleAgents() error {
	log := logger.New("ns=workers.agent").At("scaleAgents")

	sys, err := p.SystemGet()
	if err != nil {
		log.Error(err)
		return err
	}

	apps, err := p.AppList()
	if err != nil {
		log.Error(err)
		return err
	}

	for _, a := range apps {
		if a.Outputs["Agents"] == "" {
			continue
		}

		log.Logf("app=%s agents=%s", a.Name, a.Outputs["Agents"])

		for _, agent := range strings.Split(a.Outputs["Agents"], ",") {
			n := fmt.Sprintf("%sFormation", upperName(agent))
			f := a.Parameters[n]
			fp := strings.Split(f, ",")

			fp[0] = strconv.Itoa(sys.Count)
			g := strings.Join(fp, ",")

			if g == f {
				continue
			}

			if err := p.updateStack(p.rackStack(a.Name), "", map[string]string{n: g}); err != nil {
				log.Error(err)
				continue
			}
		}
	}

	log.Success()
	return nil
}
