package manifest1_test

import (
	"testing"

	"github.com/convox/rack/manifest1"
	"github.com/stretchr/testify/assert"
)

func TestPortsBadData(t *testing.T) {
	tests := map[string]string{
		"badport1": `error loading manifest: invalid portmapping 534a`,
		"badport2": `error loading manifest: invalid portmapping 534b:5000`,
		"badport3": `error loading manifest: invalid portmapping 5000:534c`,
		"badport4": `error loading manifest: invalid portmapping 5000:9000:1000`,
	}

	for fixture, message := range tests {
		m, err := manifestFixture(fixture)

		assert.Nil(t, m)

		if assert.NotNil(t, err) {
			assert.Equal(t, message, err.Error())
		}
	}
}

func TestPortsShift(t *testing.T) {
	m, err := manifestFixture("shift")
	m.Shift(5000)

	if assert.NoError(t, err) {
		web := m.Services["web"]

		if assert.NotNil(t, web) {

			if assert.Equal(t, len(web.Ports), 2) {
				assert.Equal(t, web.Ports[0].Balancer, 5000)
				assert.Equal(t, web.Ports[0].Container, 5000)
				assert.Equal(t, web.Ports[1].Balancer, 11000)
				assert.Equal(t, web.Ports[1].Container, 7000)
			}
		}
	}
}

func TestPortsShiftWithSSL(t *testing.T) {
	m, err := manifestFixture("shift-with-ssl")
	// Shift the whole manifest by 2; this is evaluated in addition to any per-service convox.start.shift labels.
	m.Shift(2)

	if assert.NoError(t, err) {
		// Web has a convox.start.shift label, for a total shift of 4.
		web := m.Services["web"]

		if assert.NotNil(t, web) {

			if assert.Equal(t, len(web.Ports), 2) {
				assert.Equal(t, web.Ports[0].Balancer, 84)
				assert.Equal(t, web.Ports[0].Container, 4001)
				assert.Equal(t, web.Ports[1].Balancer, 447)
				assert.Equal(t, web.Ports[1].Container, 4001)
			}

			assert.Equal(t, web.Labels["convox.start.shift"], "2")

			// The label should have been changed from 443 to 445 with --shift,
			// and from 445 to 447 with convox.start.shift.
			assert.Equal(t, web.Labels["convox.port.443.protocol"], "")
			assert.Equal(t, web.Labels["convox.port.445.protocol"], "")
			assert.Equal(t, web.Labels["convox.port.447.protocol"], "https")
		}
	}
}

func TestPortsString(t *testing.T) {
	assert.Equal(t, "5000:9000/tcp", manifest1.Port{Balancer: 5000, Container: 9000, Public: true, Protocol: manifest1.TCP}.String())
	assert.Equal(t, "9000/tcp", manifest1.Port{Container: 9000, Protocol: manifest1.TCP}.String())
}
