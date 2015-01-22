package release

import "github.com/convox/kernel/web/provider"

func Create(cluster, app, ami string, options map[string]string) error {
	return provider.ReleaseCreate(cluster, app, ami, options)
}

func Promote(cluster, app, release string) error {
	return provider.ReleasePromote(cluster, app, release)
}

func Copy(cluster, app, release string) (string, error) {
	return provider.ReleaseCopy(cluster, app, release)
}
