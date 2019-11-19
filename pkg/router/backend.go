package router

import (
	"crypto/tls"
)

type Backend interface {
	CA() (*tls.Certificate, error)
	InternalIP() string
	ExternalIP() string
	IdleGet(target string) (bool, error)
	IdleSet(target string, idle bool) error
	Start() error
}

type BackendRouter interface {
	TargetAdd(host, target string, idles bool) error
	TargetRemove(host, target string) error
}
