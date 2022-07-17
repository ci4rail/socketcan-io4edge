package socketcan

import (
	"errors"
	"fmt"
	"unsafe"

	"golang.org/x/sys/unix"
)

type ifReq struct {
	Name  [16]byte
	Index int
}

func ifIndex(socket int, ifName string) (int, error) {

	ifNameRaw, err := unix.ByteSliceFromString(ifName)
	if err != nil {
		return 0, err
	}
	if len(ifNameRaw) > 16 {
		return 0, errors.New("maximum ifname length is 16 characters")
	}

	ifReq := ifReq{}
	copy(ifReq.Name[:], ifNameRaw)
	err = ioctlIfreq(socket, &ifReq)
	return ifReq.Index, err
}

func ioctlIfreq(socket int, ifreq *ifReq) (err error) {
	_, _, errno := unix.Syscall(
		unix.SYS_IOCTL,
		uintptr(socket),
		unix.SIOCGIFINDEX,
		uintptr(unsafe.Pointer(ifreq)),
	)
	if errno != 0 {
		return fmt.Errorf("ioctl SIOCGIFINDEX failed: %v", errno)
	}
	return nil
}
