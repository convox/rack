// hid - Gopher Interface Devices (USB HID)
// Copyright (c) 2017 Péter Szilágyi. All rights reserved.
//
// This file is released under the 3-clause BSD license. Note however that Linux
// support depends on libusb, released under GNU LGPL 2.1 or later.

// +build !linux,!darwin,!windows ios !cgo

package hid

// Supported returns whether this platform is supported by the HID library or not.
// The goal of this method is to allow programatically handling platforms that do
// not support USB HID and not having to fall back to build constraints.
func Supported() bool {
	return false
}

// Enumerate returns a list of all the HID devices attached to the system which
// match the vendor and product id. On platforms that this file implements the
// function is a noop and returns an empty list always.
func Enumerate(vendorID uint16, productID uint16) []DeviceInfo {
	return nil
}

// Device is a live HID USB connected device handle. On platforms that this file
// implements the type lacks the actual HID device and all methods are noop.
type Device struct {
	DeviceInfo // Embed the infos for easier access
}

// Open connects to an HID device by its path name. On platforms that this file
// implements the method just returns an error.
func (info DeviceInfo) Open() (*Device, error) {
	return nil, ErrUnsupportedPlatform
}

// Close releases the HID USB device handle. On platforms that this file implements
// the method is just a noop.
func (dev *Device) Close() error { return nil }

// Write sends an output report to a HID device. On platforms that this file
// implements the method just returns an error.
func (dev *Device) Write(b []byte) (int, error) {
	return 0, ErrUnsupportedPlatform
}

// SendFeatureReport sends a feature report to a HID device
//
// Feature reports are sent over the Control endpoint as a Set_Report transfer.
// The first byte of b must contain the Report ID. For devices which only
// support a single report, this must be set to 0x0. The remaining bytes
// contain the report data. Since the Report ID is mandatory, calls to
// SendFeatureReport() will always contain one more byte than the report
// contains. For example, if a hid report is 16 bytes long, 17 bytes must be
// passed to SendFeatureReport(): the Report ID (or 0x0, for devices
// which do not use numbered reports), followed by the report data (16 bytes).
// In this example, the length passed in would be 17.
func (dev *Device) SendFeatureReport(b []byte) (int, error) {
	return 0, ErrUnsupportedPlatform
}

// Read retrieves an input report from a HID device. On platforms that this file
// implements the method just returns an error.
func (dev *Device) Read(b []byte) (int, error) {
	return 0, ErrUnsupportedPlatform
}

// GetFeatureReport retreives a feature report from a HID device
//
// Set the first byte of []b to the Report ID of the report to be read. Make
// sure to allow space for this extra byte in []b. Upon return, the first byte
// will still contain the Report ID, and the report data will start in b[1].
func (dev *Device) GetFeatureReport(b []byte) (int, error) {
	return 0, ErrUnsupportedPlatform
}
