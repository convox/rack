package k8s_test

import (
	"fmt"
	"testing"

	"github.com/convox/rack/pkg/structs"
	"github.com/convox/rack/provider/k8s"
	"github.com/stretchr/testify/require"
	am "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	fakek8s "k8s.io/client-go/kubernetes/fake"
	testk8s "k8s.io/client-go/testing"
)

func TestAppCancel(t *testing.T) {
	testProvider(t, func(p *k8s.Provider, c *fakek8s.Clientset) {
		err := p.AppCancel("app1")
		require.EqualError(t, err, "unimplemented")
	})
}

func TestAppCreate(t *testing.T) {
	testProvider(t, func(p *k8s.Provider, c *fakek8s.Clientset) {
		a, err := p.AppCreate("app1", structs.AppCreateOptions{})
		require.NoError(t, err)
		require.NotNil(t, a)

		require.Equal(t, "2", a.Generation)
		require.Equal(t, "app1", a.Name)

		ns, err := c.CoreV1().Namespaces().Get("test-app1", am.GetOptions{})
		require.NoError(t, err)
		require.NotNil(t, ns)

		require.Equal(t, "convox", ns.ObjectMeta.Labels["system"])
		require.Equal(t, "test", ns.ObjectMeta.Labels["rack"])
		require.Equal(t, "app", ns.ObjectMeta.Labels["type"])
		require.Equal(t, "app1", ns.ObjectMeta.Labels["name"])
	})
}

func TestAppCreateError(t *testing.T) {
	testProviderManual(t, func(p *k8s.Provider, c *fakek8s.Clientset) {
		c.AddReactor("create", "namespaces", func(action testk8s.Action) (bool, runtime.Object, error) {
			return true, nil, fmt.Errorf("err1")
		})

		a, err := p.AppCreate("app1", structs.AppCreateOptions{})
		require.EqualError(t, err, "err1")
		require.Nil(t, a)
	})
}

func TestAppDelete(t *testing.T) {
	testProvider(t, func(p *k8s.Provider, c *fakek8s.Clientset) {
		a, err := p.AppCreate("app1", structs.AppCreateOptions{})
		require.NoError(t, err)
		require.NotNil(t, a)

		ns, err := c.CoreV1().Namespaces().Get("test-app1", am.GetOptions{})
		require.NoError(t, err)
		require.NotNil(t, ns)

		err = p.AppDelete("app1")
		require.NoError(t, err)

		ns, err = c.CoreV1().Namespaces().Get("test-app1", am.GetOptions{})
		require.Error(t, err)
		require.Nil(t, ns)
	})
}

func TestAppDeleteError(t *testing.T) {
	testProviderManual(t, func(p *k8s.Provider, c *fakek8s.Clientset) {
		c.AddReactor("delete", "namespaces", func(action testk8s.Action) (bool, runtime.Object, error) {
			return true, nil, fmt.Errorf("err1")
		})

		err := p.AppDelete("app1")
		require.EqualError(t, err, "err1")
	})
}
