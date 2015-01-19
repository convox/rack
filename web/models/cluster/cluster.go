package cluster

import (
	"math/rand"
	"sort"

	"github.com/convox/kernel/web/models"
	"github.com/convox/kernel/web/provider"
)

func List() (models.Clusters, error) {
	pc, err := provider.ClusterList()

	if err != nil {
		return nil, err
	}

	mc := make(models.Clusters, len(pc))

	for i, c := range pc {
		mc[i].Name = c.Name
		mc[i].Status = c.Status

		mc[i].CpuUsed = (rand.Int() % 100)
		mc[i].CpuTotal = 100
		mc[i].MemoryUsed = (rand.Int() % 100)
		mc[i].MemoryTotal = 100
		mc[i].DiskUsed = (rand.Int() % 100)
		mc[i].DiskTotal = 100
	}

	sort.Sort(mc)

	return mc, nil
}

func Show(name string) (*models.Cluster, error) {
	apps, err := provider.AppList(name)

	cluster := &models.Cluster{
		Name: name,
		Apps: make([]models.App, len(apps)),
	}

	if err != nil {
		return nil, err
	}

	for i, app := range apps {
		cluster.Apps[i].Name = app.Name
		cluster.Apps[i].Status = app.Status

		cluster.Apps[i].CpuUsed = (rand.Int() % 100)
		cluster.Apps[i].CpuTotal = 100
		cluster.Apps[i].MemoryUsed = (rand.Int() % 100)
		cluster.Apps[i].MemoryTotal = 100
		cluster.Apps[i].DiskUsed = (rand.Int() % 100)
		cluster.Apps[i].DiskTotal = 100
	}

	sort.Sort(cluster.Apps)

	return cluster, nil
}

func Create(name string) error {
	err := provider.ClusterCreate(name)

	if err != nil {
		return err
	}

	return nil
}

func Delete(name string) error {
	return provider.ClusterDelete(name)
}
