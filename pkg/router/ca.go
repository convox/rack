package router

import (
	"crypto/tls"

	am "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (r *Router) ca() (*tls.Certificate, error) {
	c, err := r.Cluster.CoreV1().Secrets("convox-system").Get("ca", am.GetOptions{})
	if err != nil {
		return nil, err
	}

	ca, err := tls.X509KeyPair(c.Data["tls.crt"], c.Data["tls.key"])
	if err != nil {
		return nil, err
	}

	return &ca, nil
}
