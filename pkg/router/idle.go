package router

import (
	"fmt"
	"time"

	ae "k8s.io/api/extensions/v1beta1"
	am "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (r *Router) HostIdleStatus(host string) (bool, error) {
	idle := true

	ts, err := r.storage.TargetList(host)
	if err != nil {
		return false, err
	}

	for _, t := range ts {
		if service, namespace, ok := parseTarget(t); ok {
			s, err := r.Cluster.ExtensionsV1beta1().Deployments(namespace).GetScale(service, am.GetOptions{})
			if err != nil {
				return false, err
			}

			if s.Spec.Replicas > 0 {
				idle = false
				break
			}
		}
	}

	return idle, nil
}

func (r *Router) HostIdle(host string) error {
	idle, err := r.storage.IdleGet(host)
	if err != nil {
		return err
	}
	if idle {
		return nil
	}

	fmt.Printf("ns=router at=idle host=%q\n", host)

	if err := r.storage.IdleSet(host, true); err != nil {
		return err
	}

	ts, err := r.storage.TargetList(host)
	if err != nil {
		return err
	}

	for _, t := range ts {
		if service, namespace, ok := parseTarget(t); ok {
			scale := &ae.Scale{
				ObjectMeta: am.ObjectMeta{
					Namespace: namespace,
					Name:      service,
				},
				Spec: ae.ScaleSpec{Replicas: 0},
			}

			if _, err := r.Cluster.ExtensionsV1beta1().Deployments(namespace).UpdateScale(service, scale); err != nil {
				fmt.Printf("ns=router at=idle host=%q error=%q\n", host, err)
			}
		}
	}

	return nil
}

func (r *Router) HostUnidle(host string) error {
	idle, err := r.storage.IdleGet(host)
	if err != nil {
		return err
	}
	if !idle {
		return nil
	}

	fmt.Printf("ns=router at=unidle host=%q state=unidling\n", host)

	ts, err := r.storage.TargetList(host)
	if err != nil {
		return err
	}

	for _, t := range ts {
		if service, namespace, ok := parseTarget(t); ok {
			scale := &ae.Scale{
				ObjectMeta: am.ObjectMeta{
					Namespace: namespace,
					Name:      service,
				},
				Spec: ae.ScaleSpec{Replicas: 1},
			}

			if _, err := r.Cluster.ExtensionsV1beta1().Deployments(namespace).UpdateScale(service, scale); err != nil {
				fmt.Printf("ns=router at=unidle host=%q error=%q\n", host, err)
			}

			for {
				time.Sleep(200 * time.Millisecond)
				if rs, err := r.Cluster.AppsV1().Deployments(namespace).Get(service, am.GetOptions{}); err == nil {
					if rs.Status.ReadyReplicas > 0 {
						time.Sleep(500 * time.Millisecond)
						break
					}
				}
			}
		}
	}

	if err := r.storage.IdleSet(host, false); err != nil {
		return err
	}

	fmt.Printf("ns=router at=unidle host=%q state=ready\n", host)

	return nil
}

func (r *Router) idleTicker() {
	for range time.Tick(idleCheck) {
		if err := r.idleTick(); err != nil {
			fmt.Printf("ns=router at=idle.ticker error=%v\n", err)
		}
	}
}

func (r *Router) idleTick() error {
	hs, err := r.storage.IdleReady(time.Now().UTC().Add(-1 * idleTimeout))
	if err != nil {
		return err
	}

	for _, h := range hs {
		if err := r.HostIdle(h); err != nil {
			return err
		}
	}

	return nil
}
