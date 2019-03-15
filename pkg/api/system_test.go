package api_test

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"testing"
	"time"

	"github.com/convox/rack/pkg/options"
	"github.com/convox/rack/pkg/structs"
	"github.com/convox/stdsdk"
	"github.com/stretchr/testify/require"
)

var fxSystem = structs.System{
	Count:      1,
	Domain:     "domain",
	Name:       "name",
	Outputs:    map[string]string{"k1": "v1", "k2": "v2"},
	Parameters: map[string]string{"k1": "v1", "k2": "v2"},
	Provider:   "provider",
	Region:     "region",
	Status:     "status",
	Type:       "type",
	Version:    "version",
}

var fxMetric = structs.Metric{
	Name: "metric1",
	Values: structs.MetricValues{
		{
			Time:    time.Date(2018, 9, 1, 0, 0, 0, 0, time.UTC),
			Average: 2.0,
			Minimum: 1.0,
			Maximum: 3.0,
		},
		{
			Time:    time.Date(2018, 9, 1, 1, 0, 0, 0, time.UTC),
			Average: 2.0,
			Minimum: 1.0,
			Maximum: 3.0,
		},
	},
}

func TestSystemGet(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		s1 := fxSystem
		s2 := structs.System{}
		p.On("SystemGet").Return(&s1, nil)
		err := c.Get("/system", stdsdk.RequestOptions{}, &s2)
		require.NoError(t, err)
		require.Equal(t, s1, s2)
	})
}

func TestSystemGetError(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		var s1 *structs.System
		p.On("SystemGet").Return(nil, fmt.Errorf("err1"))
		err := c.Get("/system", stdsdk.RequestOptions{}, s1)
		require.EqualError(t, err, "err1")
		require.Nil(t, s1)
	})
}

func TestSystemLogs(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		d1 := []byte("test")
		r1 := ioutil.NopCloser(bytes.NewReader(d1))
		opts := structs.LogsOptions{Since: options.Duration(2 * time.Minute)}
		p.On("SystemLogs", opts).Return(r1, nil)
		r2, err := c.Websocket("/system/logs", stdsdk.RequestOptions{})
		require.NoError(t, err)
		d2, err := ioutil.ReadAll(r2)
		require.NoError(t, err)
		require.Equal(t, d1, d2)
	})
}

func TestSystemLogsError(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		opts := structs.LogsOptions{Since: options.Duration(2 * time.Minute)}
		p.On("SystemLogs", opts).Return(nil, fmt.Errorf("err1"))
		r1, err := c.Websocket("/system/logs", stdsdk.RequestOptions{})
		require.NoError(t, err)
		require.NotNil(t, r1)
		d1, err := ioutil.ReadAll(r1)
		require.NoError(t, err)
		require.Equal(t, []byte("ERROR: err1\n"), d1)
	})
}

func TestSystemMetrics(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		m1 := structs.Metrics{fxMetric, fxMetric}
		m2 := structs.Metrics{}
		opts := structs.MetricsOptions{
			End:     options.Time(time.Date(2018, 10, 1, 3, 4, 5, 0, time.UTC)),
			Metrics: []string{"foo", "bar"},
			Period:  options.Int64(300),
			Start:   options.Time(time.Date(2018, 9, 1, 2, 3, 4, 0, time.UTC)),
		}
		ro := stdsdk.RequestOptions{
			Query: stdsdk.Query{
				"end":     "20181001.030405.000000000",
				"metrics": "foo,bar",
				"period":  "300",
				"start":   "20180901.020304.000000000",
			},
		}
		p.On("SystemMetrics", opts).Return(m1, nil)
		err := c.Get("/system/metrics", ro, &m2)
		require.NoError(t, err)
		require.Equal(t, m1, m2)
	})
}

func TestSystemMetricsError(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		var m1 structs.Metrics
		p.On("SystemMetrics", structs.MetricsOptions{}).Return(nil, fmt.Errorf("err1"))
		err := c.Get("/system/metrics", stdsdk.RequestOptions{}, &m1)
		require.EqualError(t, err, "err1")
		require.Nil(t, m1)
	})
}

func TestSystemProcesses(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		p1 := structs.Processes{fxProcess, fxProcess}
		p2 := structs.Processes{}
		opts := structs.SystemProcessesOptions{
			All: options.Bool(true),
		}
		ro := stdsdk.RequestOptions{
			Query: stdsdk.Query{
				"all": "true",
			},
		}
		p.On("SystemProcesses", opts).Return(p1, nil)
		err := c.Get("/system/processes", ro, &p2)
		require.NoError(t, err)
		require.Equal(t, p1, p2)
	})
}

func TestSystemProcessesError(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		var p1 structs.Processes
		p.On("SystemProcesses", structs.SystemProcessesOptions{}).Return(nil, fmt.Errorf("erp1"))
		err := c.Get("/system/processes", stdsdk.RequestOptions{}, &p1)
		require.EqualError(t, err, "erp1")
		require.Nil(t, p1)
	})
}

func TestSystemReleases(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		r1 := structs.Releases{fxRelease, fxRelease}
		r2 := structs.Releases{}
		p.On("SystemReleases").Return(r1, nil)
		err := c.Get("/system/releases", stdsdk.RequestOptions{}, &r2)
		require.NoError(t, err)
		require.Equal(t, r1, r2)
	})
}

func TestSystemReleasesError(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		var r1 structs.Releases
		p.On("SystemReleases").Return(nil, fmt.Errorf("err1"))
		err := c.Get("/system/releases", stdsdk.RequestOptions{}, &r1)
		require.EqualError(t, err, "err1")
		require.Nil(t, r1)
	})
}

func TestSystemUpdate(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		opts := structs.SystemUpdateOptions{
			Count:      options.Int(1),
			Parameters: map[string]string{"k1": "v1", "k2": "v2"},
			Type:       options.String("type"),
			Version:    options.String("version"),
		}
		ro := stdsdk.RequestOptions{
			Params: stdsdk.Params{
				"count":      "1",
				"parameters": "k1=v1&k2=v2",
				"type":       "type",
				"version":    "version",
			},
		}
		p.On("SystemUpdate", opts).Return(nil)
		err := c.Put("/system", ro, nil)
		require.NoError(t, err)
	})
}

func TestSystemUpdateError(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		p.On("SystemUpdate", structs.SystemUpdateOptions{}).Return(fmt.Errorf("err1"))
		err := c.Put("/system", stdsdk.RequestOptions{}, nil)
		require.EqualError(t, err, "err1")
	})
}
