package hid

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"github.com/karalabe/hid"
	butil "github.com/marshallbrekka/go-u2fhost/bytes"
)

// The HID message structure is defined at the following url.
// https://fidoalliance.org/specs/fido-u2f-v1.1-id-20160915/fido-u2f-hid-protocol-v1.1-id-20160915.html

const TYPE_INIT uint8 = 0x80
const HID_RPT_SIZE uint16 = 64

const CMD_INIT uint8 = 0x06
const CMD_WINK uint8 = 0x08
const CMD_APDU uint8 = 0x03

const STAT_ERR uint8 = 0xbf

/** Interfaces **/
type Device interface {
	Open() error
	Close()
	SendAPDU(instruction, p1, p2 uint8, data []byte) (uint16, []byte, error)
}

type baseDevice interface {
	Open() error
	Close()
	Write([]byte) (int, error)
	Read([]byte) (int, error)
}

// Returns an array of available HID devices.
func Devices() []*HidDevice {
	u2fDevices := []*HidDevice{}
	devices := hid.Enumerate(0x0, 0x0)
	for i, device := range devices {
		if device.UsagePage == 0xf1d0 && device.Usage == 1 {
			u2fDevices = append(u2fDevices, newHidDevice(newRawHidDevice(&devices[i])))
		}
	}
	return u2fDevices
}

type HidDevice struct {
	device    baseDevice
	channelId uint32
	// Use the crypto/rand reader directly so we can unit test
	randReader io.Reader
}

func newHidDevice(device baseDevice) *HidDevice {
	return &HidDevice{
		device:     device,
		channelId:  0xffffffff,
		randReader: rand.Reader,
	}
}

func (dev *HidDevice) Open() error {
	err := dev.device.Open()
	if err != nil {
		return err
	}
	nonce := make([]byte, 8)
	_, err = io.ReadFull(dev.randReader, nonce)
	if err != nil {
		return err
	}
	channelId, err := initDevice(dev.device, dev.channelId, nonce)
	if err != nil {
		return err
	}
	dev.channelId = channelId
	return nil
}

func (dev *HidDevice) Close() {
	dev.device.Close()
	dev.channelId = 0xffffffff
}

func (dev *HidDevice) SendAPDU(instruction, p1, p2 uint8, data []byte) (uint16, []byte, error) {
	request := butil.Concat(
		// first byte is always zero
		[]byte{0, instruction, p1, p2},
		int24bytes(uint32(len(data))),
		data,
		[]byte{0x04, 0x00},
	)
	resp, err := call(dev.device, dev.channelId, CMD_APDU, request)
	if err != nil {
		return 0, nil, err
	}
	status := resp[len(resp)-2:]
	return bytesint16(status), resp[:len(resp)-2], nil
}

/** Helper Functions **/

func call(dev baseDevice, channelId uint32, command uint8, data []byte) ([]byte, error) {
	err := sendRequest(dev, channelId, command, data)
	if err != nil {
		return nil, err
	}
	return readResponse(dev, channelId, command)
}

func sendRequest(dev baseDevice, channelId uint32, command uint8, data []byte) error {
	copyLength := min(uint16(len(data)), HID_RPT_SIZE-7)
	offset := copyLength
	var sequence uint8 = 0

	fullRequest, err := butil.ConcatInto(
		make([]byte, HID_RPT_SIZE+1),
		// skip first byte
		[]byte{0},
		int32bytes(channelId),
		[]byte{TYPE_INIT | command},
		int16bytes(uint16(len(data))),
		data[0:copyLength],
	)
	if err != nil {
		return err
	}
	_, err = dev.Write(fullRequest)
	if err != nil {
		return err
	}
	for offset < uint16(len(data)) {
		copyLength = min(uint16(len(data)-int(offset)), HID_RPT_SIZE-5)
		fullRequest, err = butil.ConcatInto(
			make([]byte, HID_RPT_SIZE+1),
			// skip first byte
			[]byte{0},
			int32bytes(channelId),
			[]byte{0x7f & sequence},
			data[offset:offset+copyLength],
		)
		if err != nil {
			return err
		}
		_, err := dev.Write(fullRequest)
		if err != nil {
			return err
		}
		sequence += 1
		offset += copyLength
	}
	return nil
}

func readResponse(dev baseDevice, channelId uint32, command uint8) ([]byte, error) {
	header := butil.Concat(int32bytes(channelId), []byte{TYPE_INIT | command})
	response := make([]byte, HID_RPT_SIZE)
	for !bytes.Equal(header, response[:5]) {
		_, err := dev.Read(response)
		if err != nil {
			return nil, err
		}
		if bytes.Equal(response[:4], header[:4]) && response[4] == STAT_ERR {
			return nil, u2fhiderror(response[6])
		}
	}
	dataLength := bytesint16(response[5:7])
	data := make([]byte, dataLength)
	totalRead := min(dataLength, HID_RPT_SIZE-7)
	copy(data, response[7:7+totalRead])
	var sequence uint8 = 0
	for totalRead < dataLength {
		response = make([]byte, HID_RPT_SIZE)
		_, err := dev.Read(response)
		if err != nil {
			return nil, err
		}
		if !bytes.Equal(response[:4], header[:4]) {
			return nil, errors.New("Wrong CID from device!")
		}
		if response[4] != (sequence & 0x7f) {
			return nil, errors.New("Wrong SEQ from device!")
		}
		sequence += 1
		partLength := min(HID_RPT_SIZE-5, dataLength-totalRead)
		copy(data[totalRead:totalRead+partLength], response[5:5+partLength])
		totalRead += partLength
	}
	return data, nil
}

func initDevice(dev baseDevice, channelId uint32, nonce []byte) (uint32, error) {
	resp, err := call(dev, channelId, CMD_INIT, nonce)
	if err != nil {
		return 0, err
	}
	for !bytes.Equal(resp[:8], nonce) {
		resp, err = readResponse(dev, channelId, CMD_INIT)
		if err != nil {
			return 0, err
		}
	}
	return binary.BigEndian.Uint32(resp[8:12]), nil
}

func int32bytes(i uint32) []byte {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, i)
	return b
}

func int24bytes(i uint32) []byte {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, i)
	return b[1:4]
}

func int16bytes(i uint16) []byte {
	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, i)
	return b
}

func bytesint16(b []byte) uint16 {
	return binary.BigEndian.Uint16(b)
}

func min(a uint16, b uint16) uint16 {
	if a > b {
		return b
	} else {
		return a
	}
}

func u2fhiderror(err uint8) error {
	return fmt.Errorf("U2FHIDError: 0x%02x", err)
}
