# socketcan-io4edge
Tools to connect io4edge CAN devices with the Linux socketcan stack.

## Tool socketcan-io4edge

Connects an io4edge CANL2 instance with a virtual socket CAN. All frames from io4edge CAN device will be published on virtual CAN and vice versa.

e.g.
```bash
$ socketcan-io4edge MIO04-1-can vcan0
```

## Tool socketcan-io4edge-runner

Watches the network for io4edge CAN devices and automatically starts `socketcan-io4edge` processes to connect them with a virtual socket CAN network with a matching name, if one exists. It also watches the virtual can link instances for state changes and reacts accordingly (starts and stops `socketcan-io4edge` processes when link changes up/down).

This program is typically started as a systemd-service.

The virtual socket CAN network must be named according to the MDNS instance names of the io4edge CAN device.

E.g. if the io4edge CAN instance name is `MIO04-1-can`, the virtual Socket CAN device must have been named `vcanMIO4-1` (without `-can`). Because network interface names can have only max. 15 characters, but io4edge instance names can be longer, `socketcan-io4edge-runner` strips longer device names to 15 characters, while preserving the beginning and end of the instance name. Examples:

* Instance Name `S101-IOU04-USB-EXT-1-can` -> vcan name `vcanS101xxEXT-1`
* Instance Name `123456789012-can` -> vcan name `vcan1234xx89012`

### Typical usage

#### Create a socketCAN instance

To access a io4edge device via socketCAN, we create a virtual socketCAN network that matches the service name of the io4edge CAN Interface.

The virtual socket CAN network must be named according to io4edge CAN Interface service name. E.g. if the service name is MYDEV-can, the virtual socketCAN device must be named vcanMYDEV (without -can). Because network interface names can have only max. 15 characters, but service names can be longer, there is a rule to map longer service names to socketCAN device names:

vcan`<first-4-chars-of-service-name>`xx`<last-5-chars-of-service-name>`

Examples:

* Service Name S101-IOU04-USB-EXT-1-can -> vcan name vcanS101xxEXT-1
* Service Name 123456789012-can -> vcan name vcan1234xx89012
* Service Name MIO04-1-can -> vcan name vcanMIO04-1

To create a virtual socketCAN network enter:

```
ip link add dev vcanMIO04-1 type vcan
ip link set up vcanMIO04-1
```

#### socketcan-io4edge usage

Assuming you have an io4edge CAN device with instance name `MYDEV-can`.

In a first shell:

```bash
$ sudo socketcan-io4edge-runner /usr/bin/socketcan-io4edge
```

In a second shell:
```
$ sudo ip link add dev vcanMYDEV type vcan
$ sudo ip link set up vcanMYDEV
$ cangen vcanMYDEV
# -> Frames are sent by io4edge device

# dump frames including errors from io4edge device
$./candump vcanMYDEV vcanMYDEV,1FFFFFFF:1FFFFFFF,#FFFFFFFF -e
```
