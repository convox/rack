package manifest_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
//	"github.com/convox/rack/manifest"
	"github.com/convox/rack/manifest"
)

func TestBalancer(t *testing.T) {
	m, err := manifestFixture("balancer")

	if assert.NoError(t, err) {
		if assert.Equal(t, len(m.Balancers()), 1) {
			balancer := m.Balancers()[0]

			assert.Equal(t, "TCP", balancer.HealthProtocol())
			assert.Equal(t, "80", balancer.HealthPort())
			assert.Equal(t, "", balancer.HealthPath())
			assert.Equal(t, "3", balancer.HealthTimeout())
		}
	}
}

func TestBalancerLabels(t *testing.T) {
	m, err := manifestFixture("balancer-labels")

	if assert.NoError(t, err) {
		if assert.Equal(t, len(m.Balancers()), 1) {
			balancer := m.Balancers()[0]

			assert.Equal(t, "HTTP", balancer.HealthProtocol())
			assert.Equal(t, "443", balancer.HealthPort())
			assert.Equal(t, "/foo", balancer.HealthPath())
			assert.Equal(t, "20", balancer.HealthTimeout())
			assert.Equal(t, "3", balancer.HealthThresholdUnhealthy())
			assert.Equal(t, "4", balancer.HealthThresholdHealthy())
			assert.Equal(t, true, balancer.UseAppCookieStickiness(manifest.Port{Balancer: 2345}))
			assert.Equal(t, "kevinTest", balancer.AppCookieStickinessName(manifest.Port{Balancer: 2345}))
		}
	}
}

func TestBalancerSecure(t *testing.T) {
	m, err := manifestFixture("balancer-secure")

	if assert.NoError(t, err) {
		if assert.Equal(t, len(m.Balancers()), 1) {
			balancer := m.Balancers()[0]

			assert.Equal(t, "SSL", balancer.HealthProtocol())
			assert.Equal(t, "443", balancer.HealthPort())
		}
	}
}
