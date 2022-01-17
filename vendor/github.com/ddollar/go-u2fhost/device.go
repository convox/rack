package u2fhost

import "github.com/marshallbrekka/go-u2fhost/hid"

// APDU Commands
const (
	u2fCommandRegister          uint8 = 0x01 // Registration command
	u2fCommandAuthenticate      uint8 = 0x02 // Authenticate/sign command
	u2fCommandVersion           uint8 = 0x03 // Read version string command
	u2fCommandCheckRegister     uint8 = 0x04 // Registration command that incorporates checking key handles
	u2fCommandAuthenticateBatch uint8 = 0x05 // Authenticate/sign command for a batch of key handles
)

// APDU Response Codes
const (
	u2fStatusNoError                uint16 = 0x9000
	u2fStatusWrongData              uint16 = 0x6A80
	u2fStatusConditionsNotSatisfied uint16 = 0x6985
	u2fStatusCommandNotAllowed      uint16 = 0x6986
	u2fStatusInsNotSupported        uint16 = 0x6D00
)

// Authentication control byte
const (
	u2fAuthEnforce   uint8 = 0x03 // Enforce user presence and sign
	u2fAuthCheckOnly uint8 = 0x07 // Check only
)

type HidDevice struct {
	hidDevice hid.Device
}

func newHidDevice(dev hid.Device) *HidDevice {
	return &HidDevice{
		hidDevice: dev,
	}
}

// Returns a list of supported U2F devices as HidDevice pointers.
// If no supported devices are found, the returned list is empty.
func Devices() []*HidDevice {
	hidDevices := hid.Devices()
	devices := make([]*HidDevice, len(hidDevices))
	for i, _ := range hidDevices {
		devices[i] = newHidDevice(hidDevices[i])
	}
	return devices
}

// Opens the device.
// Must be called before calling Register or Authenticate.
func (dev *HidDevice) Open() error {
	return dev.hidDevice.Open()
}

// Closes the device.
func (dev *HidDevice) Close() {
	dev.hidDevice.Close()
}

// Returns the U2F version the device supports.
func (dev *HidDevice) Version() (string, error) {
	status, response, err := dev.hidDevice.SendAPDU(u2fCommandVersion, 0, 0, []byte{})
	if err != nil {
		return "", err
	}
	if status != u2fStatusNoError {
		return "", u2ferror(status)
	}
	return string(response), nil
}
