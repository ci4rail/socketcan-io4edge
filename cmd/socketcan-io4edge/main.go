package main

import (
	"flag"
	"fmt"
	"os"
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
}
