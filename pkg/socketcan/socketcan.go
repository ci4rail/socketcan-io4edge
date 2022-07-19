package socketcan

import (
	"encoding/binary"
	"fmt"

	"golang.org/x/sys/unix"
)

// CANFrame represents a CAN frame.
type CANFrame struct {
	ID       uint32
	DLC      uint8
	Data     []byte
	Extended bool
}

// ErrorClass represents athe CAN error class.
type ErrorClass uint32

// ErrorFrameData0 represents details of arbitration lost error.
type ErrorFrameData0 uint8

const (
	// CANErrTxTimeout flags a TX timeout (by netdevice driver)
	CANErrTxTimeout ErrorClass = 0x00000001
	// CANErrLostArb flags a lost arbitration, details in ErrorFrameData0
	CANErrLostArb ErrorClass = 0x00000002
	// CANErrCtrl flags controller problems
	CANErrCtrl ErrorClass = 0x00000004
	// CANErrProt flags protocol violations
	CANErrProt ErrorClass = 0x00000008
	// CANErrTrx flags transceiver status
	CANErrTrx ErrorClass = 0x00000010
	// CANErrAck flags received no ACK on transmission
	CANErrAck ErrorClass = 0x00000020
	// CANErrBusOff flags bus off
	CANErrBusOff ErrorClass = 0x00000040
	// CANErrBusError flags bus error
	CANErrBusError ErrorClass = 0x00000080
	// CANErrRestarted flags controller restarted
	CANErrRestarted ErrorClass = 0x00000100

	// CANErrCtrlUnspec flags unspecified
	CANErrCtrlUnspec ErrorFrameData0 = 0x00
	// CANErrCtrlRxOverflow flags RX buffer overflow
	CANErrCtrlRxOverflow ErrorFrameData0 = 0x01
	// CANErrCtrlTxOverflow flags TX buffer overflow
	CANErrCtrlTxOverflow ErrorFrameData0 = 0x02
	// CANErrCtrlRxWarning flags reached warning level for RX errors
	CANErrCtrlRxWarning ErrorFrameData0 = 0x04
	// CANErrCtrlTxWarning flags reached warning level for TX errors
	CANErrCtrlTxWarning ErrorFrameData0 = 0x08
	// CANErrCtrlRxPassive flags reached error passive status RX
	CANErrCtrlRxPassive ErrorFrameData0 = 0x10
	// CANErrCtrlTxPassive flags reached error passive status TX
	CANErrCtrlTxPassive ErrorFrameData0 = 0x20
)

func (f *CANFrame) String() string {
	var s string

	if f.Extended {
		s += fmt.Sprintf("extended Frame %08x", f.ID)
	} else {
		s += fmt.Sprintf("standard Frame %03x", f.ID)
	}

	s += fmt.Sprintf(" DLC: %d, Data: ", f.DLC)
	for i, b := range f.Data {
		if i >= int(f.DLC) {
			break
		}
		s += fmt.Sprintf("%02x", b)
	}
	return s
}

// RawInterface represents a raw CAN interface.
type RawInterface struct {
	ifName string
	socket int
}

// NewRawInterface creates a new raw CAN interface.
func NewRawInterface(interfaceName string) (*RawInterface, error) {
	socket, err := unix.Socket(unix.AF_CAN, unix.SOCK_RAW, unix.CAN_RAW)
	if err != nil {
		return nil, err
	}
	ifindex, err := ifIndex(socket, interfaceName)
	if err != nil {
		return nil, err
	}
	addr := &unix.SockaddrCAN{Ifindex: ifindex}
	if err = unix.Bind(socket, addr); err != nil {
		return nil, err
	}
	return &RawInterface{
		ifName: interfaceName,
		socket: socket,
	}, nil
}

// Close closes the raw CAN interface.
func (i *RawInterface) Close() error {
	return unix.Close(i.socket)
}

// Send sends a CAN frame.
func (i *RawInterface) Send(f *CANFrame) error {
	frameBytes := make([]byte, 16)
	// bytes 0-3: arbitration ID
	if f.ID < 0x800 {
		// standard ID
		binary.LittleEndian.PutUint32(frameBytes[0:4], f.ID)
	} else {
		// extended ID
		// set bit 31 (frame format flag (0 = standard 11 bit, 1 = extended 29 bit)
		binary.LittleEndian.PutUint32(frameBytes[0:4], f.ID|1<<31)
	}

	// byte 4: data length code
	frameBytes[4] = f.DLC
	// data
	copy(frameBytes[8:], f.Data)

	_, err := unix.Write(i.socket, frameBytes)
	return err
}

// Receive receives a CAN frame.
func (i *RawInterface) Receive() (*CANFrame, error) {
	f := CANFrame{}
	frameBytes := make([]byte, 16)
	_, err := unix.Read(i.socket, frameBytes)
	if err != nil {
		return nil, err
	}

	// bytes 0-3: arbitration ID
	f.ID = uint32(binary.LittleEndian.Uint32(frameBytes[0:4]))
	// remove bit 31: extended ID flag
	f.ID &= 0x7FFFFFFF
	// byte 4: data length code
	f.DLC = frameBytes[4]
	// data
	f.Data = make([]byte, 8)
	copy(f.Data, frameBytes[8:])

	return &f, nil
}
