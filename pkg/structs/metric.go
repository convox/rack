package structs

import "time"

type Metric struct {
	Name   string       `json:"name"`
	Values MetricValues `json:"values"`
}

type Metrics []Metric

type MetricValue struct {
	Average float64   `json:"avg"`
	Count   float64   `json:"count"`
	Maximum float64   `json:"max"`
	Minimum float64   `json:"min"`
	P90     float64   `json:"p90"`
	P95     float64   `json:"p95"`
	P99     float64   `json:"p99"`
	Sum     float64   `json:"sum"`
	Time    time.Time `json:"time"`
}

type MetricValues []MetricValue

type MetricsOptions struct {
	End     *time.Time `query:"end"`
	Metrics []string   `query:"metrics"`
	Start   *time.Time `query:"start"`
	Period  *int64     `query:"period"`
}
