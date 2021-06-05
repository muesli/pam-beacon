package main

// #cgo LDFLAGS: -lpam
// #include <security/pam_appl.h>
// #include <security/pam_modules.h>
import "C"

import (
	"fmt"
	"os"
	"os/user"
	"strings"
	"time"
	"unsafe"

	"github.com/muesli/go-pam"
	"github.com/muka/go-bluetooth/api"
	"github.com/muka/go-bluetooth/bluez/profile/adapter"
	"github.com/muka/go-bluetooth/bluez/profile/device"
	log "github.com/sirupsen/logrus"
)

var (
	logLevel = log.InfoLevel
	beaconCh = make(chan (string))
)

const (
	adapterID = "hci0"
	timeout   = 30 * time.Second
)

func logError(args ...interface{}) {
	log.Error(args...)
	time.Sleep(time.Second)
}

func logErrorf(s string, args ...interface{}) {
	log.Errorf(s, args...)
	time.Sleep(time.Second)
}

func closeBluetooth() {
	if err := api.Exit(); err != nil {
		logError(err)
	}
}

func findDevice(addresses []string) bool {
	log.Debugf("Looking for beacons %s...", addresses)

	a, err := adapter.GetAdapter(adapterID)
	if err != nil {
		logError(err)
		return false
	}
	defer closeBluetooth()

	if p, err := a.GetPowered(); err != nil {
		logError(err)
		return false
	} else if !p {
		if err := a.SetPowered(true); err != nil {
			logError(err)
			return false
		}
	}

	if monitorCachedDevices(a, addresses) {
		return true
	}

	cancel, err := discoverDevices(a, addresses)
	if err != nil {
		logError(err)
		return false
	}
	defer func() {
		log.Debug("Canceling...")
		cancel()
	}()

	select {
	case <-time.After(timeout):
		return false
	case <-beaconCh:
		return true
	}
}

func discoverDevices(a *adapter.Adapter1, addresses []string) (func(), error) {
	if err := a.FlushDevices(); err != nil {
		logError(err)
		return nil, err
	}

	discovery, cancel, err := api.Discover(a, nil)
	if err != nil {
		logError(err)
		return nil, err
	}

	log.Debugf("Discovered devices:")
	go func() {
		for ev := range discovery {
			go func(ev *adapter.DeviceDiscovered) {
				dev, err := device.NewDevice1(ev.Path)
				if err != nil {
					log.Errorf("%s: %s", ev.Path, err)
					return
				}
				if dev == nil {
					log.Errorf("%s: not found", ev.Path)
					return
				}

				if addr, ok := checkDevice(dev, addresses); ok {
					go func(addr string) {
						// we sent this in a go-routine to prevent this function
						// from dead-locking a bluetooth shutdown
						beaconCh <- addr
					}(addr)
				}
			}(ev)
		}
	}()

	return cancel, nil
}

func checkDevice(dev *device.Device1, addresses []string) (string, bool) {
	props, err := dev.GetProperties()
	if err != nil {
		logErrorf("%s: Failed to get properties: %s", dev.Path(), err.Error())
		return "", false
	}
	log.Debugf("name=%s addr=%s rssi=%d trusted=%t tx=%d connected=%t",
		props.Name, props.Address, props.RSSI, props.Trusted, props.TxPower, props.Connected)

	var addr string
	for _, v := range addresses {
		if strings.EqualFold(props.Address, v) {
			addr = props.Address
			break
		}
	}
	if addr == "" {
		return "", false
	}

	if !props.Connected {
		// check we can actually connect to the device
		log.Debugf("Connecting to %s...", props.Address)
		err = dev.Connect()
		if err != nil {
			logErrorf("%s: Failed to connect: %s", dev.Path(), err.Error())
			return "", false
		}

		log.Debugf("Disconnecting from %s...", props.Address)
		err = dev.Disconnect()
		if err != nil {
			logErrorf("%s: Failed to disconnect: %s", dev.Path(), err.Error())
		}
	}

	log.Printf("Beacon %s found!", props.Address)
	return props.Address, true
}

func monitorCachedDevices(api *adapter.Adapter1, addresses []string) bool {
	devices, err := api.GetDevices()
	if err != nil {
		logError(err)
		return false
	}

	log.Debugf("Cached devices:")
	for _, dev := range devices {
		if _, ok := checkDevice(dev, addresses); ok {
			return true
		}
	}

	return false
}

//export goAuthenticate
func goAuthenticate(handle *C.pam_handle_t, flags C.int, argv []string) C.int {
	for _, arg := range argv {
		if strings.ToLower(arg) == "debug" {
			logLevel = log.DebugLevel
		}
	}
	log.SetLevel(logLevel)
	log.Debugf("argv: %+v", argv)

	hdl := pam.Handle{Ptr: unsafe.Pointer(handle)}
	username, err := hdl.GetUser()
	if err != nil {
		return C.PAM_AUTH_ERR
	}
	addrs, err := readUserConfig(username)
	if err != nil {
		logError(err)
		switch err.(type) {
		case user.UnknownUserError:
			return C.PAM_USER_UNKNOWN
		default:
			return C.PAM_AUTHINFO_UNAVAIL
		}
	}

	if findDevice(addrs) {
		return C.PAM_SUCCESS
	}
	return C.PAM_AUTH_ERR
}

//export setCred
func setCred(handle *C.pam_handle_t, flags C.int, argv []string) C.int {
	return C.PAM_SUCCESS
}

// main is for testing purposes only, the PAM module has to be built with:
// go build -buildmode=c-shared
func main() {
	logLevel = log.DebugLevel
	log.SetLevel(logLevel)

	if len(os.Args) < 2 {
		fmt.Println("usage: pam-beacon <mac-addr>")
		os.Exit(2)
	}
	if !findDevice(os.Args[1:]) {
		os.Exit(1)
	}
}
