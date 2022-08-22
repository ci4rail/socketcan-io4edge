package socketcan

import (
	"encoding/binary"
	"fmt"

	"log"

	"golang.org/x/sys/unix"
)

// CANFrame represents a CAN frame.
type CANFrame struct {
	ID       uint32
	DLC      uint8
	Data     []byte
	Extended bool
	RTR      bool
}

// CANErrorClass represents athe CAN error class.
type CANErrorClass uint32

// CANCtrlErrorDetails represents of a can controller error (when CANErrCtrl is set in CANErrorClass).
type CANCtrlErrorDetails uint8

const (
	// CANErrTxTimeout flags a TX timeout
	CANErrTxTimeout CANErrorClass = 0x00000001
	// CANErrLostArb flags a lost arbitration
	CANErrLostArb CANErrorClass = 0x00000002
	// CANErrCtrl flags controller problems
	CANErrCtrl CANErrorClass = 0x00000004
	// CANErrProt flags protocol violations
	CANErrProt CANErrorClass = 0x00000008
	// CANErrTrx flags transceiver status
	CANErrTrx CANErrorClass = 0x00000010
	// CANErrAck flags received no ACK on transmission
	CANErrAck CANErrorClass = 0x00000020
	// CANErrBusOff flags bus off
	CANErrBusOff CANErrorClass = 0x00000040
	// CANErrBusError flags bus error
	CANErrBusError CANErrorClass = 0x00000080
	// CANErrRestarted flags controller restarted
	CANErrRestarted CANErrorClass = 0x00000100

	// CANErrCtrlUnspec flags unspecified
	CANErrCtrlUnspec CANCtrlErrorDetails = 0x00
	// CANErrCtrlRxOverflow flags RX buffer overflow
	CANErrCtrlRxOverflow CANCtrlErrorDetails = 0x01
	// CANErrCtrlTxOverflow flags TX buffer overflow
	CANErrCtrlTxOverflow CANCtrlErrorDetails = 0x02
	// CANErrCtrlRxWarning flags reached warning level for RX errors
	CANErrCtrlRxWarning CANCtrlErrorDetails = 0x04
	// CANErrCtrlTxWarning flags reached warning level for TX errors
	CANErrCtrlTxWarning CANCtrlErrorDetails = 0x08
	// CANErrCtrlRxPassive flags reached error passive status RX
	CANErrCtrlRxPassive CANCtrlErrorDetails = 0x10
	// CANErrCtrlTxPassive flags reached error passive status TX
	CANErrCtrlTxPassive CANCtrlErrorDetails = 0x20

	canErrFlag = 0x20000000
	canRTRFlag = 0x40000000
	canEFFFlag = 0x80000000
)

// CANErrorFrame represents a CAN error frame.
type CANErrorFrame struct {
	ErrorClass          CANErrorClass
	CANCtrlErrorDetails CANCtrlErrorDetails
}

func (f *CANFrame) String() string {
	var s string

	if f.Extended {
		s += fmt.Sprintf("extended Frame %08x", f.ID)
	} else {
		s += fmt.Sprintf("standard Frame %03x", f.ID)
	}
	if f.RTR {
		s += " (RTR)"
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
// blocking write
// Use this function for standard and extended frames.
func (i *RawInterface) Send(f *CANFrame) error {
	frameBytes := make([]byte, 16)
	id := f.ID
	if f.RTR {
		id |= canRTRFlag
	}

	if !f.Extended {
		// standard ID
		if f.ID > 0x7FF {
			return fmt.Errorf("ID %x is not a standard ID", f.ID)
		}
		binary.LittleEndian.PutUint32(frameBytes[0:4], id)
	} else {
		// extended ID
		if f.ID > 0x1FFFFFFF {
			return fmt.Errorf("ID %x is not an extended ID", f.ID)
		}
		id |= canEFFFlag
		binary.LittleEndian.PutUint32(frameBytes[0:4], id)
	}

	// byte 4: data length code
	frameBytes[4] = f.DLC
	// data
	copy(frameBytes[8:], f.Data)

	_, err := unix.Write(i.socket, frameBytes)
	if err != nil {
		log.Printf("Error writing to CAN socket: %v", err)
	}
	return err
}

// SendErrorFrame sends a CAN error frame.
func (i *RawInterface) SendErrorFrame(f *CANErrorFrame) error {
	frameBytes := make([]byte, 16)

	id := uint32(f.ErrorClass | canErrFlag)
	// bytes 0-3: ID
	binary.LittleEndian.PutUint32(frameBytes[0:4], id)

	// byte 4: data length code
	frameBytes[4] = 8
	// byte 9: Controller err details
	frameBytes[9] = byte(f.CANCtrlErrorDetails)

	_, err := unix.Write(i.socket, frameBytes)
	if err != nil {
		log.Printf("Error writing to CAN socket: %v", err)
	}
	return err
}

// Receive receives a CAN frame.
// Blocking read
// Handles only standard and extended frames, error frames are ignored
func (i *RawInterface) Receive() (*CANFrame, error) {
	for {
		f := CANFrame{}
		frameBytes := make([]byte, 16)
		_, err := unix.Read(i.socket, frameBytes)
		if err != nil {
			return nil, err
		}

		// bytes 0-3: ID
		id := uint32(binary.LittleEndian.Uint32(frameBytes[0:4]))

		if id&canErrFlag == 0 { // ignore error frames

			if id&canEFFFlag == 0 {
				// standard ID
				f.ID = id & 0x7FF
			} else {
				// extended ID
				f.ID = id & 0x1FFFFFFF
				f.Extended = true
			}
			if id&canRTRFlag != 0 {
				f.RTR = true
			}

			// byte 4: data length code
			f.DLC = frameBytes[4]
			// data
			f.Data = make([]byte, 8)
			copy(f.Data, frameBytes[8:])

			return &f, nil
		}
	}
}
