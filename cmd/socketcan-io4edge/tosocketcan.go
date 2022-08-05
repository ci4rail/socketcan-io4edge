package main

import (
	"fmt"
	"os"
	"time"

	"github.com/ci4rail/io4edge-client-go/canl2"
	"github.com/ci4rail/io4edge-client-go/functionblock"
	fspb "github.com/ci4rail/io4edge_api/canL2/go/canL2/v1alpha1"
	"github.com/ci4rail/socketcan-io4edge/pkg/socketcan"
)

const (
	bucketSamples     = 1
	bufferedSamples   = 400
	streamKeepAliveMs = 1000
)

// unified frame for both normal and error frames
type canFrameCombined struct {
	haveNormalFrame bool
	normalFrame     *socketcan.CANFrame
	haveErrorFrame  bool
	errorFrame      *socketcan.CANErrorFrame
}

func toSocketCAN(s *socketcan.RawInterface, io4edgeCANClient *canl2.Client) {
	// create a queue to buffer the received CAN frames from io4edge device
	frameQ := make(chan *canFrameCombined, 128)

	// Go routine to read from io4edge device
	go func() {
		var busState fspb.ControllerState = fspb.ControllerState_CAN_OK

		err := io4edgeCANClient.StartStream(
			canl2.WithFBStreamOption(functionblock.WithBucketSamples(bucketSamples)),
			canl2.WithFBStreamOption(functionblock.WithBufferedSamples(bufferedSamples)),
			canl2.WithFBStreamOption(functionblock.WithKeepaliveInterval(streamKeepAliveMs)),
		)
		if err != nil {
			fmt.Printf("StartStream failed: %v\n", err)
			os.Exit(1)
		}

		for {
			// read next bucket from stream or null bucket
			sd, err := io4edgeCANClient.ReadStream(time.Millisecond * streamKeepAliveMs * 3)
			if err != nil {
				// timeout is a fatal error
				fmt.Printf("Io4Edge ReadStream failed: %v\n", err)
				os.Exit(1)
			}
			frames := sd.FSData.Samples
			for _, f := range frames {
				if f.ControllerState != busState {
					// generate socket CAN error frame in case of bus state changes to BUS_OFF or ERROR_PASSIVE
					scF := busStateChangeToSocketCANErrorFrame(busState, f.ControllerState)
					if scF != nil {
						frameQ <- scF
					}
					busState = f.ControllerState
				}
				scFrame := io4EdgeSampleTosocketCANFrame(f)
				frameQ <- scFrame
			}
		}
	}()

	// Go routine to write to socketcan
	go func() {
		for {
			rxFrames := readCombinedFrameQ(frameQ)
			//fmt.Printf("Sending to socketcan\n")
			for _, f := range rxFrames {

				if f.haveErrorFrame {
					//fmt.Printf(" send errorframe to sc %v\n", f.errorFrame)
					err := s.SendErrorFrame(f.errorFrame)
					if err != nil {
						fmt.Printf("Error writing error frame to CAN socket: %v", err)
					}
				}
				if f.haveNormalFrame {
					//fmt.Printf(" send to sc %s\n", f.normalFrame.String())
					err := s.Send(f.normalFrame)
					if err != nil {
						fmt.Printf("Error writing to CAN socket: %v", err)
					}
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

func io4EdgeSampleTosocketCANFrame(sample *fspb.Sample) *canFrameCombined {
	f := &canFrameCombined{}

	if sample.IsDataFrame {
		f.haveNormalFrame = true
		f.normalFrame = &socketcan.CANFrame{
			ID:       sample.Frame.MessageId,
			DLC:      uint8(len(sample.Frame.Data)),
			Data:     sample.Frame.Data,
			Extended: sample.Frame.ExtendedFrameFormat,
			RTR:      sample.Frame.RemoteFrame,
		}
	}
	// convert error events
	if sample.Error != fspb.ErrorEvent_CAN_NO_ERROR {
		fmt.Printf("Got Error Event %v\n", sample.Error)
		f.haveErrorFrame = true
		f.errorFrame = &socketcan.CANErrorFrame{}
		switch sample.Error {
		case fspb.ErrorEvent_CAN_TX_FAILED:
			f.errorFrame.ErrorClass = socketcan.CANErrTxTimeout | socketcan.CANErrAck
		case fspb.ErrorEvent_CAN_RX_QUEUE_FULL:
			f.errorFrame.ErrorClass = socketcan.CANErrCtrl
			f.errorFrame.CANCtrlErrorDetails = socketcan.CANErrCtrlRxOverflow
		case fspb.ErrorEvent_CAN_ARB_LOST:
			f.errorFrame.ErrorClass = socketcan.CANErrLostArb
		case fspb.ErrorEvent_CAN_BUS_ERROR:
			f.errorFrame.ErrorClass = socketcan.CANErrBusError
		}
	}
	return f
}

func busStateChangeToSocketCANErrorFrame(oldState fspb.ControllerState, newState fspb.ControllerState) *canFrameCombined {
	if newState == fspb.ControllerState_CAN_OK {
		return nil
	}
	if newState == fspb.ControllerState_CAN_BUS_OFF {
		fmt.Printf("GOT BUS OFF\n")
		return &canFrameCombined{
			haveErrorFrame: true,
			errorFrame: &socketcan.CANErrorFrame{
				ErrorClass: socketcan.CANErrBusOff,
			},
		}
	}
	if newState == fspb.ControllerState_CAN_ERROR_PASSIVE {
		fmt.Printf("GOT ERROR PASSIVE\n")
		return &canFrameCombined{
			haveErrorFrame: true,
			errorFrame: &socketcan.CANErrorFrame{
				ErrorClass:          socketcan.CANErrCtrl,
				CANCtrlErrorDetails: socketcan.CANErrCtrlTxPassive | socketcan.CANErrCtrlRxPassive,
			},
		}
	}
	return nil
}
