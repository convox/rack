package main

import (
	"encoding/json"
	"fmt"
	"math"
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
	data, err := ConvoxGet("/top")

	if err != nil {
		stdcli.Error(err)
		return
	}

	var ms MetricStatistics

	err = json.Unmarshal(data, &ms)

	if err != nil {
		stdcli.Error(err)
		return
	}

	dp := ms.Datapoints[0]

	ps := "%.1f"
	fmt.Println("MAX    AVG    MIN")
	fmt.Printf("%-5s  %-5s  %-5s\n", fmt.Sprintf(ps, dp.Maximum), fmt.Sprintf(ps, dp.Average), fmt.Sprintf(ps, dp.Minimum))
}

func round(f float64) float64 {
	return math.Floor(f + .5)
}
