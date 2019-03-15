package router

import (
	"crypto/tls"
	"net"
)

type Backend interface {
	CA() (*tls.Certificate, error)
	ExternalIP(remote net.Addr) string
	IdleGet(host string) (bool, error)
	IdleSet(host string, idle bool) error
}

type BackendRouter interface {
	IdleSet(host string, idle bool) error
	TargetAdd(host, target string) error
	TargetList(host string) ([]string, error)
	TargetRemove(host, target string) error
}
