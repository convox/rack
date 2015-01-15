package app

import (
	"math/rand"
	"sort"

	"github.com/ddollar/convox/kernel/models"
	"github.com/ddollar/convox/kernel/provider"
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
	app := &models.App{
		Name:    name,
		Cluster: &models.Cluster{Name: cluster},
	}

	return app, nil
}

func Create(cluster, name string) error {
	err := provider.AppCreate(cluster, name)

	if err != nil {
		return err
	}

	return nil
}

func Delete(cluster, name string) error {
	return provider.AppDelete(cluster, name)
}
