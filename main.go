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
	"github.com/muka/go-bluetooth/emitter"
	"github.com/muka/go-bluetooth/linux/btmgmt"
	log "github.com/sirupsen/logrus"
)

var (
	logLevel = log.InfoLevel
	beaconCh = make(chan (string))
)

const (
	adapterID = "hci0"
	timeout   = 10 * time.Second
)

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

func closeBluetooth() {
	api.StopDiscovery()
	api.Exit()
}

func findDevice(address string) bool {
	log.Debugf("Looking for beacon %s...", address)

	a := btmgmt.NewBtMgmt(adapterID)
	err := a.SetPowered(true)
	if err != nil {
		logError(err)
		return false
	}

	defer closeBluetooth()
	go discoverDevices(address)

	select {
	case <-time.After(timeout):
		return false
	case beacon := <-beaconCh:
		log.Printf("Beacon %s found!", beacon)
		return true
	}
}

func monitorCachedDevices(beacon string) bool {
	devices, err := api.GetDevices()
	if err != nil {
		logError(err)
		return false
	}

	log.Debugf("Cached devices:")
	for _, dev := range devices {
		if checkDevice(dev, beacon) {
			return true
		}
	}

	return false
}

func discoverDevices(beacon string) {
	if monitorCachedDevices(beacon) {
		return
	}

	log.Debugf("Discovered devices:")

	api.StopDiscovery()
	time.Sleep(100 * time.Millisecond)

	err := api.On("discovery", emitter.NewCallback(func(ev emitter.Event) {
		discoveryEvent := ev.GetData().(api.DiscoveredDeviceEvent)
		dev := discoveryEvent.Device
		checkDevice(*dev, beacon)
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

func checkDevice(dev api.Device, beacon string) bool {
	props, err := dev.GetProperties()
	if err != nil {
		logErrorf("%s: Failed to get properties: %s", dev.Path, err.Error())
		return false
	}
	log.Debugf("name=%s addr=%s rssi=%d trusted=%t tx=%d connected=%t",
		props.Name, props.Address, props.RSSI, props.Trusted, props.TxPower, props.Connected)

	if props.Address != beacon {
		return false
	}

	if props.Connected || props.RSSI != 0 {
		beaconCh <- props.Address
		return true
	}

	watchDevice(dev, beacon)
	return false
}

func watchDevice(dev api.Device, beacon string) {
	err := dev.On("changed", emitter.NewCallback(func(ev emitter.Event) {
		changed := ev.GetData().(api.PropertyChangedEvent)
		// spew.Dump(changed.Properties)
		// spew.Dump(changed.Device.GetAllServicesAndUUID())
		if changed.Properties.Address != beacon {
			return
		}
		log.Debugf("%s: %d %s %+v", changed.Properties.Name, changed.Properties.RSSI, changed.Field, changed.Value)
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

	if findDevice(addrs[0]) {
		return C.PAM_SUCCESS
	}
	return C.PAM_AUTH_ERR

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
	if !findDevice(os.Args[1]) {
		os.Exit(1)
	}
}
