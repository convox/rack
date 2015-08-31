package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/convox/cli/Godeps/_workspace/src/github.com/codegangsta/cli"
	"github.com/convox/cli/stdcli"
)

type Datapoint struct {
	Average     float64
	Maximum     float64
	Minimum     float64
	SampleCount float64
	Sum         float64
	Timestamp   time.Time
	Unit        string
}

type MetricStatistics struct {
	Datapoints []Datapoint
	Label      string
}

func init() {
	stdcli.RegisterCommand(cli.Command{
		Name:        "top",
		Action:      cmdTop,
		Description: "resource utilization stats",
		Usage:       "",
	})
}

func cmdTop(c *cli.Context) {
	metrics := []string{"CPUUtilization", "MemoryUtilization"}
	labels := []string{"CPU", "MEM"}

	fmt.Println("     MAX    AVG    MIN    UPDATED")

	for i := 0; i < len(metrics); i++ {
		dp, err := getMetrics(metrics[i])

		if err != nil {
			stdcli.Error(err)
			return
		}

		ps := "%.1f%%"
		fmt.Printf("%-4s %-5s  %-5s  %-5s  %v\n", labels[i],
			fmt.Sprintf(ps, dp.Maximum),
			fmt.Sprintf(ps, dp.Average),
			fmt.Sprintf(ps, dp.Minimum),
			dp.Timestamp)
	}
}

func getMetrics(name string) (*Datapoint, error) {
	data, err := ConvoxGet(fmt.Sprintf("/top/%s", name))

	if err != nil {
		return nil, err
	}

	var ms MetricStatistics

	err = json.Unmarshal(data, &ms)

	if err != nil {
		return nil, err
	}

	if len(ms.Datapoints) == 0 {
		return nil, fmt.Errorf("No %s data available", name)
	}

	return &ms.Datapoints[0], nil
}
