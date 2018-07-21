package main

import (
	"bufio"
	"os"
	"os/user"
	"path/filepath"

	log "github.com/sirupsen/logrus"
)

const configFile = ".authorized_beacons"

func homeDir(username string) (string, error) {
	u, err := user.Lookup(username)
	if err != nil {
		return "", err
	}

	return u.HomeDir, nil
}

func readAddresses(f string) ([]string, error) {
	log.Debugf("Parsing MAC addresses from %s", f)

	var ss []string
	file, err := os.Open(f)
	if err != nil {
		return ss, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		ss = append(ss, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return ss, err
	}

	return ss, nil
}

func readUserConfig(username string) ([]string, error) {
	hd, err := homeDir(username)
	if err != nil {
		return []string{}, err
	}

	return readAddresses(filepath.Join(hd, configFile))
}
