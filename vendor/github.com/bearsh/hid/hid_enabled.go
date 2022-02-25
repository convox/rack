// hid - Gopher Interface Devices (USB HID)
// Copyright (c) 2017 Péter Szilágyi. All rights reserved.
//
// This file is released under the 3-clause BSD license. Note however that Linux
// support depends on libusb, released under LGNU GPL 2.1 or later.

// +build linux,cgo darwin,!ios,cgo windows,cgo

package hid

/*
#cgo CFLAGS: -I./hidapi/hidapi

#cgo !hidraw,linux CFLAGS: -I. -I./libusb/libusb -DDEFAULT_VISIBILITY="" -DOS_LINUX -D_GNU_SOURCE -DPLATFORM_POSIX
#cgo !hidraw,linux,!android LDFLAGS: -lrt
#cgo !hidraw,linux,noiconv CFLAGS: -DNO_ICONV
#cgo hidraw,linux CFLAGS: -DOS_LINUX -D_GNU_SOURCE -DHIDRAW
#cgo hidraw,linux,!android pkg-config: libudev
#cgo darwin CFLAGS: -DOS_DARWIN
#cgo darwin LDFLAGS: -framework CoreFoundation -framework IOKit -framework AppKit
#cgo windows CFLAGS: -DOS_WINDOWS
#cgo windows LDFLAGS: -lsetupapi

#ifdef OS_LINUX
	#ifdef HIDRAW
	#include "hidapi/linux/hid.c"
	#else
	#include <poll.h>
	#include "os/events_posix.c"
	#include "os/threads_posix.c"

	#include "os/linux_usbfs.c"
	#include "os/linux_netlink.c"

	#include "core.c"
	#include "descriptor.c"
	#include "hotplug.c"
	#include "io.c"
	#include "strerror.c"
	#include "sync.c"

	#undef _GNU_SOURCE // it's already defined in the c-file
	#include "hidapi/libusb/hid.c"
	#endif
#elif OS_DARWIN
	#include "hidapi/mac/hid.c"
#elif OS_WINDOWS
	#include "hidapi/windows/hid.c"
#endif
*/
import "C"

import (
	"errors"
	"runtime"
	"sync"
	"unsafe"
)

// enumerateLock is a mutex serializing access to USB device enumeration needed
// by the macOS USB HID system calls, which require 2 consecutive method calls
// for enumeration, causing crashes if called concurrently.
//
// For more details, see:
//   https://developer.apple.com/documentation/iokit/1438371-iohidmanagersetdevicematching
//   > "subsequent calls will cause the hid manager to release previously enumerated devices"
var enumerateLock sync.Mutex

// Supported returns whether this platform is supported by the HID library or not.
// The goal of this method is to allow programatically handling platforms that do
// not support USB HID and not having to fall back to build constraints.
func Supported() bool {
	return true
}

// Enumerate returns a list of all the HID devices attached to the system which
// match the vendor and product id:
//  - If the vendor id is set to 0 then any vendor matches.
//  - If the product id is set to 0 then any product matches.
//  - If the vendor and product id are both 0, all HID devices are returned.
func Enumerate(vendorID uint16, productID uint16) []DeviceInfo {
	enumerateLock.Lock()
	defer enumerateLock.Unlock()

	// Gather all device infos and ensure they are freed before returning
	head := C.hid_enumerate(C.ushort(vendorID), C.ushort(productID))
	if head == nil {
		return nil
	}
	defer C.hid_free_enumeration(head)

	// Iterate the list and retrieve the device details
	var infos []DeviceInfo
	for ; head != nil; head = head.next {
		info := DeviceInfo{
			Path:      C.GoString(head.path),
			VendorID:  uint16(head.vendor_id),
			ProductID: uint16(head.product_id),
			Release:   uint16(head.release_number),
			UsagePage: uint16(head.usage_page),
			Usage:     uint16(head.usage),
			Interface: int(head.interface_number),
		}
		if head.serial_number != nil {
			info.Serial, _ = wcharTToString(head.serial_number)
		}
		if head.product_string != nil {
			info.Product, _ = wcharTToString(head.product_string)
		}
		if head.manufacturer_string != nil {
			info.Manufacturer, _ = wcharTToString(head.manufacturer_string)
		}
		infos = append(infos, info)
	}
	return infos
}

// Open connects to an HID device by its path name.
func (info DeviceInfo) Open() (*Device, error) {
	enumerateLock.Lock()
	defer enumerateLock.Unlock()

	path := C.CString(info.Path)
	defer C.free(unsafe.Pointer(path))

	device := C.hid_open_path(path)
	if device == nil {
		return nil, errors.New("hidapi: failed to open device")
	}
	return &Device{
		DeviceInfo: info,
		device:     device,
	}, nil
}

// Device is a live HID USB connected device handle.
type Device struct {
	DeviceInfo // Embed the infos for easier access

	device *C.hid_device // Low level HID device to communicate through
	lock   sync.Mutex
}

// Close releases the HID USB device handle.
func (dev *Device) Close() error {
	dev.lock.Lock()
	defer dev.lock.Unlock()

	if dev.device != nil {
		C.hid_close(dev.device)
		dev.device = nil
	}
	return nil
}

// Write sends an output report to a HID device.
//
// Write will send the data on the first OUT endpoint, if one exists. If it does
// not, it will send the data through the Control Endpoint (Endpoint 0).
func (dev *Device) Write(b []byte) (int, error) {
	// Abort if nothing to write
	if len(b) == 0 {
		return 0, nil
	}
	// Abort if device closed in between
	dev.lock.Lock()
	device := dev.device
	dev.lock.Unlock()

	if device == nil {
		return 0, ErrDeviceClosed
	}
	// Prepend a HID report ID on Windows, other OSes don't need it
	var report []byte
	if runtime.GOOS == "windows" {
		report = append([]byte{0x00}, b...)
	} else {
		report = b
	}
	// Execute the write operation
	written := int(C.hid_write(device, (*C.uchar)(&report[0]), C.size_t(len(report))))
	if written == -1 {
		// If the write failed, verify if closed or other error
		dev.lock.Lock()
		device = dev.device
		dev.lock.Unlock()

		if device == nil {
			return 0, ErrDeviceClosed
		}
		// Device not closed, some other error occurred
		message := C.hid_error(device)
		if message == nil {
			return 0, errors.New("hidapi: unknown failure")
		}
		failure, _ := wcharTToString(message)
		return 0, errors.New("hidapi: " + failure)
	}
	return written, nil
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
	// Abort if nothing to write
	if len(b) == 0 {
		return 0, nil
	}
	// Abort if device closed in between
	dev.lock.Lock()
	device := dev.device
	dev.lock.Unlock()

	if device == nil {
		return 0, ErrDeviceClosed
	}

	// Send the feature report
	written := int(C.hid_send_feature_report(device, (*C.uchar)(&b[0]), C.size_t(len(b))))
	if written == -1 {
		// If the write failed, verify if closed or other error
		dev.lock.Lock()
		device = dev.device
		dev.lock.Unlock()

		if device == nil {
			return 0, ErrDeviceClosed
		}
		// Device not closed, some other error occurred
		message := C.hid_error(device)
		if message == nil {
			return 0, errors.New("hidapi: unknown failure")
		}
		failure, _ := wcharTToString(message)
		return 0, errors.New("hidapi: " + failure)
	}
	return written, nil
}

// Read is a wrapper to ReadTimeout that will check if device blocking is enabled
// and set timeout accordingly.
//
// This reproduces C.hid_read() behaviour in wrapping hid_read_timeout:
// return hid_read_timeout(dev, data, length, (dev->blocking)? -1: 0);
func (dev *Device) Read(b []byte) (int, error) {
	var timeout int
	if int(dev.device.blocking) == 1 {
		timeout = -1
	}
	return dev.ReadTimeout(b, timeout)
}

// ReadTimeout retrieves an input report from a HID device with a timeout. If timeout is -1 a
// blocking read is performed, else a non-blocking that waits timeout milliseconds
func (dev *Device) ReadTimeout(b []byte, timeout int) (int, error) {
	// Abort if nothing to read
	if len(b) == 0 {
		return 0, nil
	}
	// Abort if device closed in between
	dev.lock.Lock()
	device := dev.device
	dev.lock.Unlock()

	if device == nil {
		return 0, ErrDeviceClosed
	}
	// Execute the read operation
	read := int(C.hid_read_timeout(device, (*C.uchar)(&b[0]), C.size_t(len(b)), C.int(timeout)))
	if read == -1 {
		// If the read failed, verify if closed or other error
		dev.lock.Lock()
		device = dev.device
		dev.lock.Unlock()

		if device == nil {
			return 0, ErrDeviceClosed
		}
		// Device not closed, some other error occurred
		message := C.hid_error(device)
		if message == nil {
			return 0, errors.New("hidapi: unknown failure")
		}
		failure, _ := wcharTToString(message)
		return 0, errors.New("hidapi: " + failure)
	}
	return read, nil
}

// GetFeatureReport retrieves a feature report from a HID device
//
// Set the first byte of []b to the Report ID of the report to be read. Make
// sure to allow space for this extra byte in []b. Upon return, the first byte
// will still contain the Report ID, and the report data will start in b[1].
func (dev *Device) GetFeatureReport(b []byte) (int, error) {
	// Abort if we don't have anywhere to write the results
	if len(b) == 0 {
		return 0, nil
	}
	// Abort if device closed in between
	dev.lock.Lock()
	device := dev.device
	dev.lock.Unlock()

	if device == nil {
		return 0, ErrDeviceClosed
	}

	// Retrive the feature report
	read := int(C.hid_get_feature_report(device, (*C.uchar)(&b[0]), C.size_t(len(b))))
	if read == -1 {
		// If the read failed, verify if closed or other error
		dev.lock.Lock()
		device = dev.device
		dev.lock.Unlock()

		if device == nil {
			return 0, ErrDeviceClosed
		}

		// Device not closed, some other error occurred
		message := C.hid_error(device)
		if message == nil {
			return 0, errors.New("hidapi: unknown failure")
		}
		failure, _ := wcharTToString(message)
		return 0, errors.New("hidapi: " + failure)
	}

	return read, nil
}

// GetInputReport retrieves a input report from a HID device
//
// Set the first byte of []b to the Report ID of the report to be read. Make
// sure to allow space for this extra byte in []b. Upon return, the first byte
// will still contain the Report ID, and the report data will start in b[1].
func (dev *Device) GetInputReport(b []byte) (int, error) {
	// Abort if we don't have anywhere to write the results
	if len(b) == 0 {
		return 0, nil
	}
	// Abort if device closed in between
	dev.lock.Lock()
	device := dev.device
	dev.lock.Unlock()

	if device == nil {
		return 0, ErrDeviceClosed
	}

	// Retrive the feature report
	read := int(C.hid_get_input_report(device, (*C.uchar)(&b[0]), C.size_t(len(b))))
	if read == -1 {
		// If the read failed, verify if closed or other error
		dev.lock.Lock()
		device = dev.device
		dev.lock.Unlock()

		if device == nil {
			return 0, ErrDeviceClosed
		}

		// Device not closed, some other error occurred
		message := C.hid_error(device)
		if message == nil {
			return 0, errors.New("hidapi: unknown failure")
		}
		failure, _ := wcharTToString(message)
		return 0, errors.New("hidapi: " + failure)
	}

	return read, nil
}

// SetNonblocking sets the device handle to be non-blocking.
//
// In non-blocking mode calls to Read() will return
// immediately with a value of 0 if there is no data to be
// read. In blocking mode, Read() will wait (block) until
// there is data to read before returning.
func (dev *Device) SetNonblocking(b bool) error {
	// Abort if device closed in between
	dev.lock.Lock()
	device := dev.device
	dev.lock.Unlock()

	if device == nil {
		return ErrDeviceClosed
	}

	i := C.int(0)
	if b {
		i = 1
	}
	res := int(C.hid_set_nonblocking(device, i))
	if res == -1 {
		// If the read failed, verify if closed or other error
		dev.lock.Lock()
		device = dev.device
		dev.lock.Unlock()

		if device == nil {
			return ErrDeviceClosed
		}
		// Device not closed, some other error occurred
		message := C.hid_error(device)
		if message == nil {
			return errors.New("hidapi: unknown failure")
		}
		failure, _ := wcharTToString(message)
		return errors.New("hidapi: " + failure)
	}

	return nil
}
