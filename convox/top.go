package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/convox/cli/Godeps/_workspace/src/github.com/codegangsta/cli"
	"github.com/convox/cli/stdcli"
	"github.com/dustin/go-humanize"
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

	t := stdcli.NewTable("", "MIN", "AVG", "MAX", "UPDATED")

	for i := 0; i < len(metrics); i++ {
		dp, err := getMetrics(metrics[i])

		if err != nil {
			stdcli.Error(err)
			return
		}

		t.AddRow(labels[i], fmt.Sprintf("%.1f%%", dp.Minimum), fmt.Sprintf("%.1f%%", dp.Average), fmt.Sprintf("%.1f%%", dp.Maximum), humanize.Time(dp.Timestamp))
	}

	t.Print()
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
