package release

import "github.com/convox/kernel/web/provider"

func Create(cluster, app, ami string, options map[string]string) error {
	return provider.ReleaseCreate(cluster, app, ami, options)
}

func Deploy(cluster, app, ami string) error {
	return provider.ReleaseDeploy(cluster, app, ami)
}
