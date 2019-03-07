package router

import (
	"crypto/tls"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	ae "k8s.io/api/extensions/v1beta1"
	am "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type BackendKubernetes struct {
	cluster kubernetes.Interface
	ip      string
	prefix  string
	router  BackendRouter
	service string
}

func NewBackendKubernetes(router BackendRouter) (*BackendKubernetes, error) {
	b := &BackendKubernetes{router: router}

	c, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}

	kc, err := kubernetes.NewForConfig(c)
	if err != nil {
		return nil, err
	}

	b.cluster = kc

	ic, err := NewIngressController(kc, router)
	if err != nil {
		return nil, err
	}

	go ic.Run()

	if parts := strings.Split(os.Getenv("POD_IP"), "."); len(parts) > 2 {
		b.prefix = fmt.Sprintf("%s.%s.", parts[0], parts[1])
	}

	if host := os.Getenv("SERVICE_HOST"); host != "" {
		for {
			if ips, err := net.LookupIP(host); err == nil && len(ips) > 0 {
				b.service = ips[0].String()
				break
			}

			time.Sleep(1 * time.Second)
		}
	}

	s, err := kc.CoreV1().Services("convox-system").Get("router", am.GetOptions{})
	if err != nil {
		return nil, err
	}

	if len(s.Status.LoadBalancer.Ingress) > 0 && s.Status.LoadBalancer.Ingress[0].Hostname == "localhost" {
		b.ip = "127.0.0.1"
	} else {
		b.ip = s.Spec.ClusterIP
	}

	fmt.Printf("ns=backend.k8s at=new ip=%s prefix=%s service=%s\n", b.ip, b.prefix, b.service)

	return b, nil
}

func (b *BackendKubernetes) CA() (*tls.Certificate, error) {
	c, err := b.cluster.CoreV1().Secrets("convox-system").Get("ca", am.GetOptions{})
	if err != nil {
		return nil, err
	}

	ca, err := tls.X509KeyPair(c.Data["tls.crt"], c.Data["tls.key"])
	if err != nil {
		return nil, err
	}

	return &ca, nil
}

func (b *BackendKubernetes) ExternalIP(remote net.Addr) string {
	if strings.HasPrefix(remote.String(), b.prefix) {
		return b.service
	}

	return b.ip
}

func (b *BackendKubernetes) IdleGet(host string) (bool, error) {
	idle := true

	ts, err := b.router.TargetList(host)
	if err != nil {
		return false, err
	}

	for _, t := range ts {
		if service, namespace, ok := parseTarget(t); ok {
			s, err := b.cluster.ExtensionsV1beta1().Deployments(namespace).GetScale(service, am.GetOptions{})
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

func (b *BackendKubernetes) IdleSet(host string, idle bool) error {
	if idle {
		return b.idle(host)
	} else {
		return b.unidle(host)
	}
}

func (b *BackendKubernetes) idle(host string) error {
	fmt.Printf("ns=backend.k8s at=idle host=%q\n", host)

	ts, err := b.router.TargetList(host)
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

			if _, err := b.cluster.ExtensionsV1beta1().Deployments(namespace).UpdateScale(service, scale); err != nil {
				fmt.Printf("ns=backend.k8s at=idle host=%q error=%q\n", host, err)
			}
		}
	}

	return nil
}

func (b *BackendKubernetes) unidle(host string) error {
	fmt.Printf("ns=backend.k8s at=unidle host=%q state=unidling\n", host)

	ts, err := b.router.TargetList(host)
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

			if _, err := b.cluster.ExtensionsV1beta1().Deployments(namespace).UpdateScale(service, scale); err != nil {
				fmt.Printf("ns=backend.k8s at=unidle host=%q error=%q\n", host, err)
			}

			for {
				time.Sleep(200 * time.Millisecond)
				if rs, err := b.cluster.AppsV1().Deployments(namespace).Get(service, am.GetOptions{}); err == nil {
					if rs.Status.ReadyReplicas > 0 {
						time.Sleep(500 * time.Millisecond)
						break
					}
				}
			}
		}
	}

	fmt.Printf("ns=backend.k8s at=unidle host=%q state=ready\n", host)

	return nil
}
