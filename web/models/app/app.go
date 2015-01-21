package app

import (
	"math/rand"
	"sort"

	"github.com/convox/kernel/web/models"
	"github.com/convox/kernel/web/provider"
)

func List(cluster string) (models.Apps, error) {
	pa, err := provider.AppList(cluster)

	if err != nil {
		return nil, err
	}

	apps := make(models.Apps, len(pa))

	for i, c := range pa {
		apps[i].Name = c.Name

		apps[i].CpuUsed = (rand.Int() % 100)
		apps[i].CpuTotal = 100
		apps[i].MemoryUsed = (rand.Int() % 100)
		apps[i].MemoryTotal = 100
		apps[i].DiskUsed = (rand.Int() % 100)
		apps[i].DiskTotal = 100
	}

	sort.Sort(apps)

	return apps, nil
}

func Show(cluster, name string) (*models.App, error) {
	pa, err := provider.AppShow(cluster, name)

	if err != nil {
		return nil, err
	}

	app := &models.App{
		Name:     pa.Name,
		Cluster:  &models.Cluster{Name: pa.Cluster.Name},
		Releases: make(models.Releases, 0),
	}

	for _, r := range pa.Releases {
		app.Releases = append(app.Releases, models.Release{
			Ami:       r.Ami,
			CreatedAt: r.CreatedAt,
		})
	}

	return app, nil
}

func Create(cluster, name string, options map[string]string) error {
	err := provider.AppCreate(cluster, name, options)

	if err != nil {
		return err
	}

	return nil
}

func Delete(cluster, name string) error {
	return provider.AppDelete(cluster, name)
}
