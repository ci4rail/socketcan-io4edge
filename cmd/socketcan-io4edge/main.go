package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/ci4rail/socketcan-io4edge/pkg/socketcan"
)

func usage() {
	fmt.Printf("Usage: %s [OPTIONS] <io4edge-device-address> <socketcan-instance-name>\n", os.Args[0])
	flag.PrintDefaults()
	os.Exit(1)
}

func main() {
	flag.Usage = usage
	bitratePtr := flag.Int("bitrate", 1000000, "CAN Bitrate")
	flag.Parse()
	if flag.NArg() < 2 || flag.NArg() > 2 {
		flag.Usage()
		return
	}
	io4edgeAddress := flag.Arg(0)
	socketCANInstance := flag.Arg(1)

	fmt.Printf("io4edge-device-address: %s, socketcan-instance %s\n", io4edgeAddress, socketCANInstance)
	fmt.Printf("bitrate: %d\n", *bitratePtr)

	socketCAN, err := socketcan.NewRawInterface(socketCANInstance)
	if err != nil {
		fmt.Printf("Error creating socketcan interface: %v\n", err)
		os.Exit(1)
	}
	defer socketCAN.Close()

	// start gateway
	toSocketCAN(socketCAN)
	fromSocketCAN(socketCAN)

	waitForSignal()
}

func waitForSignal() {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigs
	fmt.Println()
	fmt.Println(sig)
}
