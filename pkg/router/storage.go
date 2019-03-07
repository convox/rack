package router

import "time"

type Storage interface {
	IdleGet(host string) (bool, error)
	IdleReady(cutoff time.Time) ([]string, error)
	IdleSet(host string, idle bool) error
	RequestBegin(host string) error
	RequestEnd(host string) error
	TargetAdd(host, target string) error
	TargetList(host string) ([]string, error)
	TargetRemove(host, target string) error
}
