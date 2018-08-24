package manifest1_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
