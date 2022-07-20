package main

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/ci4rail/socketcan-io4edge/pkg/socketcan"
)

type canFrameCombined struct {
	isErrorFrame bool
	normalFrame  *socketcan.CANFrame
	errorFrame   *socketcan.CANErrorFrame
}

func toSocketCAN(s *socketcan.RawInterface) {
	// create a queue to buffer the received CAN frames from io4edge device
	frameQ := make(chan *canFrameCombined, 128)

	// Go routine to read from io4edge device
	go func() {
		for {
			time.Sleep(time.Duration(rand.Intn(1000)) * time.Millisecond)

			f := &canFrameCombined{
				isErrorFrame: false,
				normalFrame: &socketcan.CANFrame{
					ID:       uint32(rand.Intn(0x7ff)),
					DLC:      8,
					Data:     []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
					Extended: false,
					RTR:      false,
				},
			}

			fmt.Printf("Generating standard frame: %s\n", f.normalFrame.String())
			frameQ <- f

			time.Sleep(time.Duration(rand.Intn(1000)) * time.Millisecond)

			f = &canFrameCombined{
				isErrorFrame: false,
				normalFrame: &socketcan.CANFrame{
					ID:       uint32(rand.Intn(0x1fffffff)),
					DLC:      4,
					Data:     []byte{0xAA, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
					Extended: true,
					RTR:      false,
				},
			}

			fmt.Printf("Generating extended frame: %s\n", f.normalFrame.String())
			frameQ <- f

			time.Sleep(time.Duration(rand.Intn(1000)) * time.Millisecond)

			f = &canFrameCombined{
				isErrorFrame: false,
				normalFrame: &socketcan.CANFrame{
					ID:       uint32(rand.Intn(0x1fffffff)),
					DLC:      4,
					Data:     []byte{},
					Extended: true,
					RTR:      true,
				},
			}

			fmt.Printf("Generating extended RTR frame: %s\n", f.normalFrame.String())
			frameQ <- f

			time.Sleep(time.Duration(rand.Intn(1000)) * time.Millisecond)

			f = &canFrameCombined{
				isErrorFrame: true,
				errorFrame: &socketcan.CANErrorFrame{
					ErrorClass: socketcan.CANErrBusOff,
				},
			}

			fmt.Printf("Generating bus error frame: %v\n", f.errorFrame)
			frameQ <- f

			time.Sleep(time.Duration(rand.Intn(1000)) * time.Millisecond)

			f = &canFrameCombined{
				isErrorFrame: true,
				errorFrame: &socketcan.CANErrorFrame{
					ErrorClass:          socketcan.CANErrCtrl,
					CANCtrlErrorDetails: socketcan.CANErrCtrlRxPassive,
				},
			}

			fmt.Printf("Generating rx passive error frame: %v\n", f.errorFrame)
			frameQ <- f

		}
	}()

	// Go routine to write to socketcan
	go func() {
		for {
			rxFrames := readCombinedFrameQ(frameQ)
			//fmt.Printf("Sending to socketcan\n")
			for _, f := range rxFrames {

				if f.isErrorFrame {
					//fmt.Printf(" send errorframe to sc %v\n", f.errorFrame)
					s.SendErrorFrame(f.errorFrame)
				} else {
					//fmt.Printf(" send to sc %s\n", f.normalFrame.String())
					s.Send(f.normalFrame)
				}
			}
		}
	}()
}

func readCombinedFrameQ(frameQ chan *canFrameCombined) []*canFrameCombined {
	rxFrames := []*canFrameCombined{}
	// wait for first frame
	f := <-frameQ
	rxFrames = append(rxFrames, f)

	// read all other frames, but non-blocking
	for {
		select {
		case f := <-frameQ:
			rxFrames = append(rxFrames, f)
		default: // queue is empty
			return rxFrames
		}
	}
}
