package api_test

import (
	"fmt"
	"testing"

	"github.com/convox/rack/pkg/structs"
	"github.com/convox/stdsdk"
	"github.com/stretchr/testify/require"
)

var fxCapacity = structs.Capacity{
	ClusterCPU:     1,
	ClusterMemory:  2,
	InstanceCPU:    3,
	InstanceMemory: 4,
	ProcessCount:   5,
	ProcessCPU:     6,
	ProcessMemory:  7,
	ProcessWidth:   8,
}

func TestCapacityGet(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		c1 := fxCapacity
		c2 := structs.Capacity{}
		p.On("CapacityGet").Return(&c1, nil)
		err := c.Get("/system/capacity", stdsdk.RequestOptions{}, &c2)
		require.NoError(t, err)
		require.Equal(t, c1, c2)
	})
}

func TestCapacityGetError(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		var c1 *structs.Capacity
		p.On("CapacityGet").Return(nil, fmt.Errorf("err1"))
		err := c.Get("/system/capacity", stdsdk.RequestOptions{}, c1)
		require.Nil(t, c1)
		require.EqualError(t, err, "err1")
	})
}
