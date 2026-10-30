package aws

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	mockaws "github.com/convox/rack/pkg/mock/aws"
	"github.com/convox/rack/pkg/structs"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestCloudwatchMetricsChunks(t *testing.T) {
	type call struct {
		ids   map[string]bool
		exprs []string
		token string
	}

	calls := []call{}
	first, second := time.Unix(1700000000, 0), time.Unix(1700003600, 0)

	cw := &mockaws.CloudWatchAPI{}
	cw.On("GetMetricData", mock.Anything).Return(func(req *cloudwatch.GetMetricDataInput) (*cloudwatch.GetMetricDataOutput, error) {
		c := call{ids: map[string]bool{}, token: aws.StringValue(req.NextToken)}
		out := &cloudwatch.GetMetricDataOutput{}

		ts := first
		if len(calls) == 0 {
			out.NextToken = aws.String("page-2")
		} else if c.token != "" {
			ts = second
		}

		for _, q := range req.MetricDataQueries {
			c.ids[*q.Id] = true
			if q.Expression != nil {
				c.exprs = append(c.exprs, *q.Expression)
			}
			out.MetricDataResults = append(out.MetricDataResults, &cloudwatch.MetricDataResult{
				Id:         q.Id,
				Label:      q.Label,
				Timestamps: []*time.Time{aws.Time(ts)},
				Values:     []*float64{aws.Float64(1)},
			})
		}

		calls = append(calls, c)

		return out, nil
	})

	mdqs := []metricDataQuerier{}
	for i := 0; i < 300; i++ {
		name := fmt.Sprintf("q%d", i)
		mdqs = append(mdqs, metricExpressions{
			Name: name,
			Statistics: []metricStatistics{
				{Name: name, Namespace: "AWS/ECS", Metric: "CPUUtilization", Statistics: []string{"Sum", "Average"}},
			},
			Expressions: []metricExpression{
				{Statistic: "Maximum", Expression: name + "_Sum_##"},
			},
		})
	}

	p := &Provider{CloudWatch: cw}

	ms, err := p.cloudwatchMetrics(mdqs, structs.MetricsOptions{})
	require.NoError(t, err)

	require.Len(t, calls, 3)
	require.Equal(t, "page-2", calls[1].token)
	require.Equal(t, calls[0].ids, calls[1].ids)
	require.Equal(t, 900, len(calls[0].ids)+len(calls[2].ids))

	for _, c := range []call{calls[0], calls[2]} {
		require.LessOrEqual(t, len(c.ids), 500)
		for _, e := range c.exprs {
			require.True(t, c.ids[strings.TrimSpace(e)], e)
		}
	}

	require.Len(t, ms, 300)

	values := map[string]int{}
	for _, m := range ms {
		values[m.Name] = len(m.Values)
	}

	require.Equal(t, 2, values["q0"])
	require.Equal(t, 1, values["q299"])
}
