package aws

import (
	"fmt"
	"time"

	"github.com/convox/rack/pkg/helpers"
)

// workerCertificates paces its ACM reads below the API's, so the monitor and the API together stay
// under the account's rate limits.
func (p *Provider) workerCertificates() {
	acmPaceLock.Lock()
	acmPaceInterval = time.Second / 4
	acmPaceLock.Unlock()

	helpers.Tick(certificateWorkerTick, p.certificatesTick)
}

func (p *Provider) certificatesTick() {
	defer recoverWith(func(err error) {
		fmt.Printf("ns=workers.certificates at=pass error=%q\n", err)
	})

	if _, err := p.certificatePass(nil, time.Now().Add(4*time.Minute), true); err != nil {
		fmt.Printf("ns=workers.certificates at=pass error=%q\n", err)
	}
}
