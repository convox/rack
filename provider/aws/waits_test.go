package aws

import (
	"net"
	"net/http"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/convox/rack/pkg/structs"
	"github.com/convox/rack/pkg/test/awsutil"
	"github.com/stretchr/testify/require"
)

func TestWaitForCertificateListedDeadline(t *testing.T) {
	fastCertificateListWait(t, 100*time.Millisecond)

	p := stubProviderHandler(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-time.After(1500 * time.Millisecond):
		case <-r.Context().Done():
		}
	}))

	start := time.Now()
	p.waitForCertificateListed("arn:aws:acm:us-test-1:123456789012:certificate/x")
	require.Less(t, time.Since(start), time.Second)
}

func TestDeleteSyslogInterfacesNeverDetaches(t *testing.T) {
	timeout := syslogDetachTimeout
	syslogDetachTimeout = 0
	t.Cleanup(func() { syslogDetachTimeout = timeout })

	attached := awsutil.Cycle{
		Request:  awsutil.Request{RequestURI: "/", Body: "/Action=DescribeNetworkInterfaces/"},
		Response: awsutil.Response{StatusCode: 200, Body: `<DescribeNetworkInterfacesResponse xmlns="http://ec2.amazonaws.com/doc/2016-11-15/"><networkInterfaceSet><item><networkInterfaceId>eni-1</networkInterfaceId><attachment><attachmentId>eni-attach-1</attachmentId></attachment></item></networkInterfaceSet></DescribeNetworkInterfacesResponse>`},
	}

	p, bodies := stubProvider(t,
		awsutil.Cycle{
			Request:  awsutil.Request{RequestURI: "/", Body: "/Action=ListStackResources/"},
			Response: awsutil.Response{StatusCode: 200, Body: `<ListStackResourcesResponse xmlns="http://cloudformation.amazonaws.com/doc/2010-05-15/"><ListStackResourcesResult><StackResourceSummaries><member><LogicalResourceId>Function</LogicalResourceId><PhysicalResourceId>convox-syslog-Function</PhysicalResourceId><ResourceType>AWS::Lambda::Function</ResourceType><ResourceStatus>CREATE_COMPLETE</ResourceStatus></member></StackResourceSummaries></ListStackResourcesResult></ListStackResourcesResponse>`},
		},
		attached,
		awsutil.Cycle{
			Request:  awsutil.Request{RequestURI: "/", Body: "/Action=DetachNetworkInterface/"},
			Response: awsutil.Response{StatusCode: 200, Body: `<DetachNetworkInterfaceResponse xmlns="http://ec2.amazonaws.com/doc/2016-11-15/"><return>true</return></DetachNetworkInterfaceResponse>`},
		},
		attached,
	)

	err := p.deleteSyslogInterfaces(&structs.Resource{Name: "syslog"})
	require.EqualError(t, err, "network interface did not detach: eni-1")
	require.Len(t, *bodies, 4)
}

func TestRegistryAddAuthTimeout(t *testing.T) {
	ln, err := net.Listen("unix", filepath.Join(t.TempDir(), "d.sock"))
	require.NoError(t, err)

	var mu sync.Mutex
	var conns []net.Conn

	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			mu.Lock()
			conns = append(conns, c)
			mu.Unlock()
		}
	}()

	t.Cleanup(func() {
		_ = ln.Close()
		mu.Lock()
		defer mu.Unlock()
		for _, c := range conns {
			_ = c.Close()
		}
	})

	host, timeout := registryDockerHost, registryAuthTimeout
	registryDockerHost, registryAuthTimeout = "unix://"+ln.Addr().String(), 100*time.Millisecond
	t.Cleanup(func() { registryDockerHost, registryAuthTimeout = host, timeout })

	p := &Provider{Region: "us-test-1"}

	start := time.Now()
	_, err = p.RegistryAdd("registry.example.com", "user", "pass")
	require.EqualError(t, err, "unable to authenticate")
	require.Less(t, time.Since(start), time.Second)
}
