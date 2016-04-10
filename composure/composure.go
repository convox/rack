package composure

import "github.com/convox/rack/composure/provider"

func Start(path, manifestfile string) error {
	return provider.CurrentProvider.ManifestRun(path, manifestfile)
}
