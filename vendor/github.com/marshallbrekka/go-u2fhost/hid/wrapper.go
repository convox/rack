package hid

import "github.com/karalabe/hid"

type RawHidDevice struct {
	Device *hid.DeviceInfo
	Handle *hid.Device
}

func newRawHidDevice(dev *hid.DeviceInfo) *RawHidDevice {
	return &RawHidDevice{
		Device: dev,
	}
}

func (dev *RawHidDevice) Open() error {
	handle, err := dev.Device.Open()
	if err != nil {
		return err
	}
	dev.Handle = handle
	return nil
}

func (dev *RawHidDevice) Close() {
	if dev.Handle != nil {
		dev.Handle.Close()
		dev.Handle = nil
	}
}

func (dev *RawHidDevice) Write(data []byte) (int, error) {
	return dev.Handle.Write(data)
}

func (dev *RawHidDevice) Read(response []byte) (int, error) {
	return dev.Handle.Read(response)
}
