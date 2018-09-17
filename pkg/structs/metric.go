package structs

import "time"

type Metric struct {
	Name   string       `json:"name"`
	Values MetricValues `json:"values"`
}

type Metrics []Metric

type MetricValue struct {
	Time   time.Time `json:"time"`
	Perc95 float64   `json:"p95"`
}

type MetricValues []MetricValue

type MetricsOptions struct {
	Start  *time.Time `query:"start"`
	End    *time.Time `query:"end"`
	Period *int64     `query:"period"`
}
