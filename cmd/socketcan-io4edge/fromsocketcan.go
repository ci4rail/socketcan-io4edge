package main

import (
	"fmt"
	"os"

	"github.com/ci4rail/io4edge-client-go/canl2"
	fspb "github.com/ci4rail/io4edge_api/canL2/go/canL2/v1alpha1"
	"github.com/ci4rail/socketcan-io4edge/pkg/socketcan"
)

const (
	maxFramesPerIo4EdgeCANSend = 30
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
			verbosePrint("received %s\n", f.String())
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
			verbosePrint("Sending %d frames to io4edge device\n", len(io4eFrames))

			// try to send frames. Ignore errors if the device is not ready, i.e. because is bus off or queue is full
			err := io4edgeCANClient.SendFrames(io4eFrames)
			if err != nil {
				fmt.Printf("Error sending frames to io4edge device: %v\n", err)
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
	numFrames := 1
	if numFrames >= maxFrames {
		return rxFrames
	}
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
	f := &fspb.Frame{
		MessageId:           s.ID,
		RemoteFrame:         s.RTR,
		ExtendedFrameFormat: s.Extended,
	}
	f.Data = make([]byte, s.DLC)
	copy(f.Data, s.Data[0:s.DLC])
	return f
}
