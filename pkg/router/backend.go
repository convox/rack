package router

import (
	"crypto/tls"
	"net"
)

type Backend interface {
	CA() (*tls.Certificate, error)
	ExternalIP(remote net.Addr) string
	IdleGet(target string) (bool, error)
	IdleSet(target string, idle bool) error
	Start() error
}

type BackendRouter interface {
	TargetAdd(host, target string, idles bool) error
	TargetRemove(host, target string) error
}
