package main

import (
	"fmt"
	"os"
	"time"

	"github.com/ci4rail/socketcan-io4edge/pkg/socketcan"
)

func fromSocketCAN(s *socketcan.RawInterface) {
	// create a queue to buffer the received CAN frames from socketcan
	frameQ := make(chan *socketcan.CANFrame, 128)

	// Go routine to read from socketcan
	go func() {
		for {
			f, err := s.Receive()
			if err != nil {
				fmt.Printf("Error reading from socketcan: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("received %s\n", f.String())
			frameQ <- f
		}
	}()

	// Go routine to write to io4edge device
	go func() {
		for {
			rxFrames := readFrameQ(frameQ)
			fmt.Printf("Sending to io4edge device\n")
			for _, f := range rxFrames {
				fmt.Printf(" %s\n", f.String())
			}
			time.Sleep(time.Second * 5) // Simulate slow io4edge device
		}
	}()

}

func readFrameQ(frameQ chan *socketcan.CANFrame) []*socketcan.CANFrame {
	rxFrames := []*socketcan.CANFrame{}
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
