package main

// #cgo LDFLAGS: -lpam
// #include <security/pam_appl.h>
// #include <security/pam_modules.h>
import "C"

import (
	"fmt"
	"os"
	"time"
	"unsafe"

	"darkdna.net/pam"
	"github.com/muka/go-bluetooth/api"
	"github.com/muka/go-bluetooth/emitter"
	"github.com/muka/go-bluetooth/linux"
	log "github.com/sirupsen/logrus"
)

var beaconCh = make(chan (string))

const adapterID = "hci0"
const timeout = 5 * time.Second
const logLevel = log.DebugLevel

func logError(args ...interface{}) {
	log.Error(args...)
	time.Sleep(time.Second)
}

func logErrorf(s string, args ...interface{}) {
	log.Errorf(s, args...)
	time.Sleep(time.Second)
}

func printEnv() {
	for _, e := range os.Environ() {
		fmt.Println(e)
	}
}

func findDevice(address string) bool {
	log.Debugf("Looking for beacon %s...", address)

	a := linux.NewBtMgmt(adapterID)
	err := a.SetPowered(true)
	if err != nil {
		logError(err)
		return false
	}

	defer api.Exit()
	go discoverDevices(address)

	select {
	case <-time.After(timeout):
		return false
	case beacon := <-beaconCh:
		log.Printf("Beacon %s found!", beacon)
		return true
	}
}

func discoverDevices(beacon string) {
	devices, err := api.GetDevices()
	if err != nil {
		logError(err)
		return
	}

	log.Debugf("Cached devices:")
	for _, dev := range devices {
		showDeviceInfo(dev, beacon)
	}

	log.Debugf("Discovered devices:")
	api.StopDiscovery()
	time.Sleep(500 * time.Millisecond)

	err = api.On("discovery", emitter.NewCallback(func(ev emitter.Event) {
		discoveryEvent := ev.GetData().(api.DiscoveredDeviceEvent)
		dev := discoveryEvent.Device
		showDeviceInfo(*dev, beacon)
	}))
	if err != nil {
		logError(err)
		return
	}

	err = api.StartDiscovery()
	if err != nil {
		logError(err)
		return
	}
}

func showDeviceInfo(dev api.Device, beacon string) {
	props, err := dev.GetProperties()
	if err != nil {
		logErrorf("%s: Failed to get properties: %s", dev.Path, err.Error())
		return
	}
	log.Debugf("name=%s addr=%s rssi=%d trusted=%t tx=%d connected=%t",
		props.Name, props.Address, props.RSSI, props.Trusted, props.TxPower, props.Connected)

	if props.Address != beacon {
		return
	}

	if props.Connected || props.RSSI != 0 {
		beaconCh <- props.Address
		return
	}

	watchDevice(dev, beacon)
}

func watchDevice(dev api.Device, beacon string) {
	err := dev.On("changed", emitter.NewCallback(func(ev emitter.Event) {
		changed := ev.GetData().(api.PropertyChangedEvent)
		// spew.Dump(changed.Properties)
		// spew.Dump(changed.Device.GetAllServicesAndUUID())
		if changed.Properties.Address != beacon {
			return
		}
		log.Debugf("%s: %d %s %+v\n", changed.Properties.Name, changed.Properties.RSSI, changed.Field, changed.Value)
		if (changed.Field == "RSSI" && changed.Value.(int16) != 0) ||
			(changed.Field == "Connected" && changed.Value.(bool)) {
			beaconCh <- changed.Properties.Address
			return
		}
	}))
	if err != nil {
		logError(err)
	}
}

//export goAuthenticate
func goAuthenticate(handle *C.pam_handle_t, flags C.int, argv []string) C.int {
	log.SetLevel(logLevel)

	hdl := pam.Handle{unsafe.Pointer(handle)}
	log.Debugf("argv: %+v", argv)

	username, err := hdl.GetUser()
	if err != nil {
		return C.PAM_AUTH_ERR
	}

	addrs, err := readUserConfig(username)
	if err != nil {
		return C.PAM_USER_UNKNOWN
	}

	if !findDevice(addrs[0]) {
		return C.PAM_AUTH_ERR
	}

	/*
		cmd := exec.Command("dbus-send", "--system", "--dest=org.bluez", "--print-reply", "/org/bluez/hci0", "org.freedesktop.DBus.Properties.Set", "string:org.bluez.Adapter1", "string", ":Discoverable", "variant:boolean:true")
		out, err := cmd.Output()
		if err != nil {
			fmt.Println(err)
		}
	*/
	/*
		cmd := exec.Command("hcitool", "name", beaconAddress)
		out, err := cmd.Output()
		if err != nil {
			fmt.Println(err)
			return C.PAM_AUTH_ERR
		}
		if len(strings.TrimSpace(string(out))) == 0 {
			return C.PAM_AUTH_ERR
		}
		return C.PAM_SUCCESS
	*/

	return C.PAM_SUCCESS
}

//export setCred
func setCred(handle *C.pam_handle_t, flags C.int, argv []string) C.int {
	return C.PAM_SUCCESS
}

// main is for testing purposes only, the PAM module has to be built with:
// go build -buildmode=c-shared
func main() {
	if !findDevice(beaconAddress) {
		os.Exit(1)
	}
}
