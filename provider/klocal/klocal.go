package klocal

import (
	"fmt"
	"time"

	"github.com/convox/rack/pkg/structs"
	"github.com/convox/rack/provider/k8s"
)

type Provider struct {
	*k8s.Provider
	// Router *router.Client
}

func FromEnv() (*Provider, error) {
	kp, err := k8s.FromEnv()
	if err != nil {
		return nil, err
	}

	kp.HostFunc = func(app, service string) string {
		return fmt.Sprintf("%s.%s", service, app)
	}

	kp.RepoFunc = func(app string) (string, bool, error) {
		return fmt.Sprintf("%s/%s", kp.Rack, app), false, nil
	}

	// s, err := NewStorage(filepath.Join(kp.Data, "rack.db"))
	// if err != nil {
	//   return nil, err
	// }

	// kp.Storage = s

	p := &Provider{
		Provider: kp,
	}

	// if host := os.Getenv("ROUTER"); host != "" {
	//   p.Router = router.NewClient(host)
	// }

	return p, nil
}

func (p *Provider) Initialize(opts structs.ProviderOptions) error {
	if err := p.Provider.Initialize(opts); err != nil {
		return err
	}

	rc, err := NewResourceController(p.Provider)
	if err != nil {
		return err
	}

	go rc.Run()

	// if p.Router != nil {
	//   go handlerLoop(p.routerRegister)
	//   go handlerLoop(p.ingressController)
	// }

	return nil
}

func handlerLoop(fn func() error) {
	for {
		if err := fn(); err != nil {
			fmt.Printf("err = %+v\n", err)
		}

		time.Sleep(1 * time.Second)
	}
}

// func (p *Provider) routerRegister() error {
//   for {
//     s, err := p.Cluster.CoreV1().Services(p.Rack).Get("rack", am.GetOptions{})
//     if err != nil {
//       return err
//     }

//     if s == nil {
//       return fmt.Errorf("could not find service for rack")
//     }

//     ps := s.Spec.Ports

//     if len(ps) != 1 {
//       return fmt.Errorf("invalid port configuration")
//     }

//     // if err := p.Router.RackCreate(p.Rack, fmt.Sprintf("tls://127.0.0.1:%d", ps[0].Port)); err != nil {
//     //   return nil
//     // }

//     time.Sleep(10 * time.Second)
//   }
// }
