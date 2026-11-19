package main

import (
	"encoding/binary"
	"errors"
	"time"

	"github.com/bearsh/hid"
)

const (
	vendorId  = 0x03f0
	productId = 0x098d

	packetPrefix1   = 0x21
	packetPrefix2   = 0xBB
	batteryQuery = 0x0B
	voiceStateQuery = 0x09
	voiceStateWrite = 0x13

	timeout = time.Millisecond * 500
)

type Command uint32

const (
	StatusRequest        Command = 0x21bb0b00
	Ping                 Command = 0x21bb0c00
	GetIdentifier        Command = 0x21bb0d00
	ConnectionState      Command = 0x21bb0300
	MicrophoneState      Command = 0x21bb0500
	VoiceState           Command = 0x21bb0900
	VoicePromptsOn       Command = 0x21bb1301
	VoicePromptsOff      Command = 0x21bb1300
	MicMonitorState      Command = 0x21bb0a00
	MicrophoneMonitorOn  Command = 0x21bb1001
	MicrophoneMonitorOff Command = 0x21bb1000
	SleepState           Command = 0x21bb0700
	SleepTimer10         Command = 0x21bb120a
	SleepTimer20         Command = 0x21bb1214
	SleepTimer30         Command = 0x21bb121e
	BatteryLevel         Command = 0x21bb0b00
)


type Headset struct {
	device *hid.Device
}


func Open() (*Headset, error){
	if !hid.Supported() {
		return nil, errors.New("hid not supported")
	}

	device := findDevice()
	if device == nil {
		return nil, errors.New("Device not found")
	}

	d, err := device.Open()
	if err != nil {
		return nil, errors.New("Could not open device, check permissions")
	}

	return &Headset{device: d}, nil
}
func (h *Headset) Close() {
	h.device.Close()
}

// findDevice returns the first matching headset, or nil.
func findDevice() *hid.DeviceInfo {
	devices := hid.Enumerate(vendorId, productId)
	if len(devices) > 0 {
		return &devices[0]
	}
	return nil
}

// read reads packets from the device until the device is closed, or until it receives
// the battery percentage in a deviceInfoQuery response.
func read(d *hid.Device, c Command) (int , error) {
	var r = make([]byte, 10)
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline){
		n, err := d.ReadTimeout(r, int(timeout.Milliseconds()))
		if err != nil {
			return 0,err
		}
		if n >=4 && r[2] == c.Cmd(){
			return (int(r[3])),nil
		}
	}
	return 0, errors.New("timeout waiting for response")
}

func (h *Headset) Set(c Command) (int,error) {
	p := make([]byte, 31)
	binary.BigEndian.PutUint32(p, uint32(c))
	return h.device.Write(p)
}

func (h *Headset) Query(c Command) (int,error) {
	p := make([]byte, 31)
	binary.BigEndian.PutUint32(p, uint32(c))
	_,err := h.device.Write(p)
	if err != nil {
		return 0, err
	}
	return read(h.device, c)
}

// Kommando-Byte, z.B. 0x0b
func (c Command) Cmd() byte {
	return byte(c >> 8)
}

// Wert-Byte, z.B. 0x01 bei VoicePromptsOn
func (c Command) Value() byte {
	return byte(c)
}
