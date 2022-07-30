/*
Copyright © 2022 Ci4Rail GmbH <engineering@ci4rail.com>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"

	log "github.com/sirupsen/logrus"

	"github.com/ci4rail/io4edge-client-go/client"
	"github.com/ci4rail/socketcan-io4edge/pkg/drunner"
)

type daemonInfo struct {
	runner *drunner.Runner
	ipPort string
}

var (
	daemonMap   = make(map[string]*daemonInfo) // key: mdns instance name
	programPath string
)

func serviceAdded(s client.ServiceInfo) error {
	var info *daemonInfo

	fmt.Println("Added service", s.GetInstanceName())

	name := vcanName(s.GetInstanceName())
	ipPort := s.GetIPAddressPort()

	info, ok := daemonMap[name]
	if ok {
		// instance already exists, check if ip or port changed
		if info.ipPort == ipPort {
			fmt.Printf("no change in ip/port for instance %s\n", name)
			return nil
		}
		// ip or port changed, kill old instance and start new one
		fmt.Printf("ip/port changed for instance %s, %s->%s stop old instance\n", name, info.ipPort, ipPort)
		info.runner.Stop()
	} else {
		// instance does not exist. start new instance
		info = &daemonInfo{}
		info.ipPort = ipPort

		err := createSocketCanDevice(name)
		if err != nil {
			log.Errorf("%v", err)
			return nil
		}
		fmt.Printf("start process for instance (%s)\n", name)
		daemonMap[name] = info
	}

	runner, err := drunner.New(name, programPath, s.GetInstanceName(), name)

	if err != nil {
		logErr("Start %s (%s) failed: %v\n", programPath, name, err)
		delInfo(name)
	}
	info.runner = runner

	return nil
}

func serviceRemoved(s client.ServiceInfo) error {
	name := vcanName(s.GetInstanceName())
	fmt.Println("Removed service", s.GetInstanceName())

	info, ok := daemonMap[name]
	if ok {
		fmt.Printf("Stopping instance for %s\n", name)
		info.runner.Stop()
		delInfo(name)
		deleteSocketCanDevice(name)
	} else {
		log.Warnf("instance for %s not in map", name)
	}
	return nil
}

func main() {
	var err error

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [OPTIONS] <ttynvt-program-path> <driver-major-number>\n", os.Args[0])
		flag.PrintDefaults()
		os.Exit(1)
	}

	logLevel := flag.String("loglevel", "info", "loglevel (debug, info, warn, error)")
	// parse command line arguments
	flag.Parse()
	if flag.NArg() != 2 {
		flag.Usage()
	}

	level, err := log.ParseLevel(*logLevel)

	if err != nil {
		log.Fatalf("Invalid log level: %v", err)
	}
	log.SetLevel(level)

	programPath = flag.Arg(0)
	_, err = os.Stat(programPath)
	if err != nil {
		if os.IsNotExist(err) {
			log.Fatalf("error: %s: path not exists!", os.Args[0])
		} else {
			log.Fatalf("error: %v", err)
		}
	}
	client.ServiceObserver("_io4edge_canl2._tcp", serviceAdded, serviceRemoved)
}

func createSocketCanDevice(socketCANInstance string) error {
	cmd := fmt.Sprintf("ip link add dev %s type vcan && ip link set up %s", socketCANInstance, socketCANInstance)
	fmt.Println(cmd)
	out, err := exec.Command("sh", "-c", cmd).CombinedOutput()
	if err != nil {
		return fmt.Errorf("error creating socketcan instance: %v: %s", err, out)
	}
	return nil
}

func deleteSocketCanDevice(socketCANInstance string) error {
	cmd := fmt.Sprintf("ip link delete %s", socketCANInstance)
	fmt.Println(cmd)
	out, err := exec.Command("sh", "-c", cmd).CombinedOutput()
	if err != nil {
		return fmt.Errorf("error deleting socketcan instance: %v: %s", err, out)
	}
	return nil
}

func vcanName(instanceName string) string {
	return "vcan-" + instanceName
}

func logErr(format string, arg ...any) {
	fmt.Fprintf(os.Stderr, format, arg...)
}

func delInfo(name string) {
	delete(daemonMap, name)
}
