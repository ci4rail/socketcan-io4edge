package main

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/ci4rail/socketcan-io4edge/pkg/socketcan"
)

func toSocketCAN(s *socketcan.RawInterface) {
	// create a queue to buffer the received CAN frames from io4edge device
	frameQ := make(chan *socketcan.CANFrame, 128)

	// Go routine to read from io4edge device
	go func() {
		for {
			time.Sleep(time.Duration(rand.Intn(1000)) * time.Millisecond)

			f := &socketcan.CANFrame{
				ID:       uint32(rand.Intn(0x7ff)),
				DLC:      8,
				Data:     []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
				Extended: false,
			}
			fmt.Printf("Generating frame: %s\n", f.String())
			frameQ <- f
		}
	}()

	// Go routine to write to socketcan
	go func() {
		for {
			rxFrames := readFrameQ(frameQ)
			//fmt.Printf("Sending to socketcan\n")
			for _, f := range rxFrames {
				fmt.Printf(" send to sc %s\n", f.String())
				s.Send(f)
			}
		}
	}()
}
