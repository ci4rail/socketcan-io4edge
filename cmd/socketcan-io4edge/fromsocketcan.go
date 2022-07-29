package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/ci4rail/io4edge-client-go/canl2"
	"github.com/ci4rail/io4edge-client-go/functionblock"
	fspb "github.com/ci4rail/io4edge_api/canL2/go/canL2/v1alpha1"
	"github.com/ci4rail/io4edge_api/io4edge/go/functionblock/v1alpha1"
	"github.com/ci4rail/socketcan-io4edge/pkg/socketcan"
)

const (
	maxFramesPerIo4EdgeCANSend = 10
	sendRetryDelay             = 100 * time.Millisecond
)

func fromSocketCAN(s *socketcan.RawInterface, io4edgeCANClient *canl2.Client) {
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
			rxFrames := readFrameQ(frameQ, maxFramesPerIo4EdgeCANSend)

			// convert socketcan frames to io4edge frames
			io4eFrames := []*fspb.Frame{}
			for _, f := range rxFrames {
				io4eFrames = append(io4eFrames, socketCANToIo4EdgeFrame(f))
			}
			log.Printf("Sending %d frames to io4edge device", len(io4eFrames))

			var err error

			// try to send frames. Retry forever if device queue is full
			for {
				err = io4edgeCANClient.SendFrames(io4eFrames)

				// retry if device's send queue full
				if err != nil && functionblock.HaveResponseStatus(err, v1alpha1.Status_TEMPORARILY_UNAVAILABLE) {
					log.Printf("Retry send frames\n")
					time.Sleep(sendRetryDelay)
				} else {
					break
				}
			}

			if err != nil {
				log.Fatalf("Error sending frames to io4edge device: %v\n", err)
			}
		}
	}()

}

func readFrameQ(frameQ chan *socketcan.CANFrame, maxFrames int) []*socketcan.CANFrame {
	rxFrames := []*socketcan.CANFrame{}
	// wait for first frame
	f := <-frameQ
	rxFrames = append(rxFrames, f)

	// read all other frames, but non-blocking
	numFrames := 0
	for {
		select {
		case f := <-frameQ:
			rxFrames = append(rxFrames, f)
			numFrames++
			if numFrames >= maxFrames {
				return rxFrames
			}
		default: // queue is empty
			return rxFrames
		}
	}
}

func socketCANToIo4EdgeFrame(s *socketcan.CANFrame) *fspb.Frame {
	return &fspb.Frame{
		MessageId:           s.ID,
		Data:                s.Data,
		RemoteFrame:         s.RTR,
		ExtendedFrameFormat: s.Extended,
	}
}
