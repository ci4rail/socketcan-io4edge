package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/ci4rail/io4edge-client-go/canl2"
	"github.com/ci4rail/socketcan-io4edge/pkg/socketcan"
)

func main() {
	flag.Usage = func() {
		fmt.Printf("Usage: %s [OPTIONS] <io4edge-device-address> <socketcan-instance-name>\n", os.Args[0])
		flag.PrintDefaults()
		os.Exit(1)
	}

	flag.Parse()
	if flag.NArg() != 2 {
		flag.Usage()
		return
	}
	io4edgeAddress := flag.Arg(0)
	socketCANInstance := flag.Arg(1)

	fmt.Printf("io4edge-device-address: %s, socketcan-instance %s\n", io4edgeAddress, socketCANInstance)

	socketCAN, err := socketcan.NewRawInterface(socketCANInstance)
	if err != nil {
		log.Fatalf("Error creating socketcan interface: %v\n", err)
		os.Exit(1)
	}
	defer socketCAN.Close()

	io4edgeCANClient, err := canl2.NewClientFromUniversalAddress(io4edgeAddress, 0)
	if err != nil {
		log.Fatalf("Failed to create canl2 client: %v\n", err)
	}
	fmt.Printf("connected to io4edge CAN at %s\n", io4edgeAddress)
	// start gateway
	toSocketCAN(socketCAN, io4edgeCANClient)
	fromSocketCAN(socketCAN, io4edgeCANClient)

	waitForSignal()
}

func waitForSignal() {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigs
	fmt.Println()
	fmt.Println(sig)
}
