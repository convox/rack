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
	IdleSet(target string, idle bool) error
	TargetAdd(host, target string) error
	TargetRemove(host, target string) error
}
