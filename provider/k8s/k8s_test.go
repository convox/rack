package k8s_test

import (
	"testing"

	"github.com/convox/rack/provider/k8s"
	"github.com/stretchr/testify/require"
	fakek8s "k8s.io/client-go/kubernetes/fake"
)

func testProvider(t *testing.T, fn func(*k8s.Provider, *fakek8s.Clientset)) {
	c := fakek8s.NewSimpleClientset()

	p, err := k8s.FromEnv()
	require.NoError(t, err)

	p.Cluster = c
	p.Rack = "test"

	fn(p, c)
}

func testProviderManual(t *testing.T, fn func(*k8s.Provider, *fakek8s.Clientset)) {
	c := &fakek8s.Clientset{}

	p := &k8s.Provider{
		Cluster: c,
		Rack:    "test",
	}

	fn(p, c)
}
