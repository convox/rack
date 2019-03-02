package router

import "time"

type Backend interface {
	// ActivityLatest(host string) (time.Time, error)

	RequestBegin(host string) error
	RequestEnd(host string) error

	IdleGet(host string) (bool, error)
	IdleReady(cutoff time.Time) ([]string, error)
	IdleSet(host string, idle bool) error

	// Route(host string) (string, error)

	TargetAdd(host, target string) error
	TargetList(host string) ([]string, error)
	TargetRemove(host, target string) error
}
