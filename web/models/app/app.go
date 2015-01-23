package app

import (
	"fmt"
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

func Get(cluster, name string) (*models.App, error) {
	pa, err := provider.AppShow(cluster, name)

	if err != nil {
		return nil, err
	}

	app := &models.App{
		Name:       pa.Name,
		Status:     pa.Status,
		Repository: pa.Repository,
		Release:    pa.Release,
		Cluster:    &models.Cluster{Name: pa.Cluster.Name},
	}

	for _, b := range pa.Builds {
		app.Builds = append(app.Builds, models.Build{
			Id:        b.Id,
			Status:    b.Status,
			Release:   b.Release,
			CreatedAt: b.CreatedAt,
			EndedAt:   b.EndedAt,
			Logs:      b.Logs,
		})
	}

	fmt.Printf("pa.Processes %+v\n", pa.Processes)

	for _, p := range pa.Processes {
		app.Processes = append(app.Processes, models.Process{
			Name:     p.Name,
			Count:    p.Count,
			Balancer: (p.Name == "web"),
		})
	}

	for _, r := range pa.Releases {
		app.Releases = append(app.Releases, models.Release{
			Id:        r.Id,
			Ami:       r.Ami,
			CreatedAt: r.CreatedAt,
		})
	}

	return app, nil
}

func Create(cluster, name string, options map[string]string) error {
	return provider.AppCreate(cluster, name, options)
}

func Delete(cluster, name string) error {
	return provider.AppDelete(cluster, name)
}
