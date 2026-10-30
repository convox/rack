package aws

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/convox/rack/pkg/cache"
	"github.com/stretchr/testify/require"
)

func cachingProvider(t *testing.T, h http.HandlerFunc) *Provider {
	p := stubProviderHandler(t, h)
	p.SkipCache = false
	p.Rack = strings.ReplaceAll(t.Name(), "/", "-")

	t.Setenv("PROVIDER", "")

	return p
}

func TestDescribeStacksCoalesced(t *testing.T) {
	var calls int32

	p := cachingProvider(t, func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		time.Sleep(100 * time.Millisecond)
		_, _ = fmt.Fprint(w, `<DescribeStacksResponse><DescribeStacksResult><Stacks><member><StackName>s</StackName></member></Stacks></DescribeStacksResult></DescribeStacksResponse>`)
	})

	stack := p.Rack + "-app"
	t.Cleanup(func() { require.NoError(t, cache.Clear("describeStacks", stack)) })

	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := p.describeStack(stack)
			if err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()

	require.EqualValues(t, 1, atomic.LoadInt32(&calls))
}

func TestCreateStackClearsDescribeStacks(t *testing.T) {
	var describes int32

	p := cachingProvider(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)

		switch {
		case strings.Contains(string(body), "Action=DescribeStacks"):
			atomic.AddInt32(&describes, 1)
			_, _ = fmt.Fprint(w, `<DescribeStacksResponse><DescribeStacksResult><Stacks><member><StackName>s</StackName></member></Stacks></DescribeStacksResult></DescribeStacksResponse>`)
		case strings.Contains(string(body), "Action=CreateStack"):
			_, _ = fmt.Fprint(w, `<CreateStackResponse><CreateStackResult><StackId>id</StackId></CreateStackResult></CreateStackResponse>`)
		}
	})

	stack := p.Rack + "-app"
	t.Cleanup(func() { require.NoError(t, cache.Clear("describeStacks", stack)) })

	_, err := p.describeStack(stack)
	require.NoError(t, err)
	_, err = p.describeStack(stack)
	require.NoError(t, err)
	require.EqualValues(t, 1, atomic.LoadInt32(&describes))

	require.NoError(t, p.createStack(stack, []byte("{}"), nil, nil))

	_, err = p.describeStack(stack)
	require.NoError(t, err)
	require.EqualValues(t, 2, atomic.LoadInt32(&describes))
}

func TestRackInstancesRefetch(t *testing.T) {
	var ec2Calls int32
	var instances atomic.Value
	instances.Store("")

	p := cachingProvider(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/x-amz-json-1.1")

		switch target := r.Header.Get("X-Amz-Target"); {
		case strings.HasSuffix(target, ".DescribeTasks"):
			_, _ = fmt.Fprint(w, `{"tasks":[{"taskArn":"arn:aws:ecs:us-test-1:123456789012:task/cluster-test/abc","taskDefinitionArn":"arn:aws:ecs:us-test-1:123456789012:task-definition/web:1","containerInstanceArn":"arn:aws:ecs:us-test-1:123456789012:container-instance/cluster-test/ci1","lastStatus":"RUNNING","containers":[{"name":"web"}],"overrides":{"containerOverrides":[]}}],"failures":[]}`)
		case strings.HasSuffix(target, ".DescribeTaskDefinition"):
			_, _ = fmt.Fprint(w, `{"taskDefinition":{"containerDefinitions":[{"name":"web","image":"img"}]}}`)
		case strings.HasSuffix(target, ".DescribeContainerInstances"):
			_, _ = fmt.Fprint(w, `{"containerInstances":[{"containerInstanceArn":"arn:aws:ecs:us-test-1:123456789012:container-instance/cluster-test/ci1","ec2InstanceId":"i-new"}],"failures":[]}`)
		case strings.Contains(string(body), "Action=DescribeInstances"):
			atomic.AddInt32(&ec2Calls, 1)
			w.Header().Set("Content-Type", "text/xml")
			_, _ = fmt.Fprintf(w, `<DescribeInstancesResponse xmlns="http://ec2.amazonaws.com/doc/2016-11-15/"><reservationSet><item><instancesSet>%s</instancesSet></item></reservationSet></DescribeInstancesResponse>`, instances.Load())
		}
	})

	t.Cleanup(func() { require.NoError(t, cache.Clear("rackInstances", p.Rack)) })

	task := "arn:aws:ecs:us-test-1:123456789012:task/cluster-test/abc"
	withIP := `<item><instanceId>i-new</instanceId><privateIpAddress>10.0.0.5</privateIpAddress></item>`
	withoutIP := `<item><instanceId>i-new</instanceId></item>`

	run := func(cached, served string) (string, int32) {
		require.NoError(t, cache.Clear("rackInstances", p.Rack))
		atomic.StoreInt32(&ec2Calls, 0)

		if cached != "-" {
			instances.Store(cached)
			_, _, err := p.rackInstances()
			require.NoError(t, err)
			time.Sleep(time.Millisecond)
		}

		instances.Store(served)

		pss, err := p.taskProcesses([]string{task})
		require.NoError(t, err)
		require.Len(t, pss, 1)

		return pss[0].Host, atomic.LoadInt32(&ec2Calls)
	}

	host, calls := run("", withIP)
	require.Equal(t, "10.0.0.5", host)
	require.EqualValues(t, 2, calls)

	host, calls = run("-", "")
	require.Equal(t, "", host)
	require.EqualValues(t, 1, calls)

	host, calls = run("-", withoutIP)
	require.Equal(t, "", host)
	require.EqualValues(t, 1, calls)
}
