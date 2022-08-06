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
	"strings"
	"sync"

	log "github.com/sirupsen/logrus"

	"github.com/ci4rail/io4edge-client-go/client"
	"github.com/ci4rail/socketcan-io4edge/internal/version"
	"github.com/ci4rail/socketcan-io4edge/pkg/drunner"
	"github.com/vishvananda/netlink"
)

type daemonInfo struct {
	runner              *drunner.Runner
	io4edgeInstanceName string
	ipPort              string
}

var (
	mu          sync.Mutex                     // protects daemonMap
	daemonMap   = make(map[string]*daemonInfo) // key: vcan name
	programPath string
)

func main() {
	var err error

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [OPTIONS] <socketcan-io4edge-program-path>\n", os.Args[0])
		flag.PrintDefaults()
		os.Exit(1)
	}

	logLevel := flag.String("loglevel", "info", "loglevel (debug, info, warn, error)")
	showVersion := flag.Bool("version", false, "show version and exit")
	flag.Parse()
	if *showVersion {
		fmt.Printf("%s\n", version.Version)
		os.Exit(0)
	}
	if flag.NArg() != 1 {
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
	// watch for socketcan link status changes
	go netlinkMonitor()
	// watch for mdns service changes
	client.ServiceObserver("_io4edge_canL2._tcp", serviceAdded, serviceRemoved)
}

func serviceAdded(s client.ServiceInfo) error {

	mu.Lock()
	defer mu.Unlock()

	var daemon *daemonInfo
	fmt.Printf("%s: service added info received from mdns\n", s.GetInstanceName())

	name := vcanName(s.GetInstanceName())
	ipPort := s.GetIPAddressPort()

	daemon, ok := daemonMap[name]
	if !ok {
		// instance does not exist.
		daemon = &daemonInfo{}
		daemon.ipPort = ipPort
		daemon.io4edgeInstanceName = s.GetInstanceName()
		daemonMap[name] = daemon
	} else {
		// instance already exists, check if ip or port changed
		if daemon.ipPort == ipPort {
			fmt.Printf("%s: no change in ip/port (nothing to do)\n", name)
			return nil
		}
		// ip or port changed, kill old instance and start new one
		daemon.ipPort = ipPort
		if daemon.runner != nil {
			fmt.Printf("%s: ip/port changed, %s->%s stop old instance\n", name, daemon.ipPort, ipPort)
			daemon.runner.Stop()
		}
	}

	if socketCANIsUp(name) {
		daemon.startProcess(name)
	} else {
		fmt.Printf("%s: socketcan link is down, don't start process\n", name)
	}

	return nil
}

func serviceRemoved(s client.ServiceInfo) error {
	mu.Lock()
	defer mu.Unlock()

	name := vcanName(s.GetInstanceName())
	fmt.Printf("%s: service removed info received from mdns\n", s.GetInstanceName())

	daemon, ok := daemonMap[name]
	if ok {
		daemon.stopProcess(name)
		delDaemon(name)
	} else {
		fmt.Printf("%s: instance not known! (ignoring)\n", name)
	}
	return nil
}

func vcanName(instanceName string) string {
	// remove "can" from instance name
	instanceName = strings.Replace(instanceName, "-can", "", 1)

	if len(instanceName) > 11 {
		instanceName = instanceName[0:4] + "xx" + instanceName[len(instanceName)-5:]
	}

	return "vcan" + instanceName
}

func (d *daemonInfo) startProcess(name string) {
	runner, err := drunner.New(name, programPath, d.io4edgeInstanceName, name)
	if err != nil {
		logErr("%s: start %s failed: %v\n", name, programPath, err)
	}
	d.runner = runner
}

func (d *daemonInfo) stopProcess(name string) {
	if d.runner != nil {
		fmt.Printf("%s: stopping process\n", name)
		d.runner.Stop()
		d.runner = nil
	}
}

func logErr(format string, arg ...any) {
	fmt.Fprintf(os.Stderr, format, arg...)
}

func delDaemon(name string) {
	delete(daemonMap, name)
}

func socketCANIsUp(name string) bool {
	l, err := netlink.LinkByName(name)
	if err != nil {
		return false
	}
	// cant't check for link up, because vcan is never up, but either unknown or down
	return l.Attrs().OperState != netlink.OperDown
}

func netlinkMonitor() {
	ch := make(chan netlink.LinkUpdate)
	if err := netlink.LinkSubscribe(ch, nil); err != nil {
		fmt.Printf("netlink.LinkSubscribe failed: %v\n", err)
		os.Exit(1)
	}
	for update := range ch {
		name := update.Link.Attrs().Name
		down := update.Link.Attrs().OperState == netlink.OperDown

		fmt.Printf("%s: update from netlinkMonitor operstate %v\n", name, update.Link.Attrs().OperState)

		mu.Lock()
		daemon, ok := daemonMap[name]
		if ok {
			if !down {
				daemon.startProcess(name)
			} else {
				daemon.stopProcess(name)
			}
		}
		mu.Unlock()
	}
}
