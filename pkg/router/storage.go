package router

import "time"

type Storage interface {
	RequestBegin(target string) error
	RequestEnd(target string) error
	Stale(cutoff time.Time) ([]string, error)
	TargetAdd(host, target string, idles bool) error
	TargetList(host string) ([]string, error)
	TargetRemove(host, target string) error
}
