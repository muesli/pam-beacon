# pam-beacon

PAM module for (multi-factor) authentication with Bluetooth Devices & Beacons

## Installation

### Packages

ArchLinux: install the AUR package `pam_beacon-git`.

### Manually

Make sure you have a working Go environment (Go 1.9 or higher is required).
See the [install instructions](http://golang.org/doc/install.html).
`libpam` and its development headers are also required.

```
$ make deps
$ make
$ sudo make install
```

## Configuration

Create a file named `.authorized_beacons` in your home directory. Put a single
line with the beacon's MAC address in it. For example:

```
00:11:22:AA:BB:CC
```

Copy `config/pam.d/system-auth-beacon` to `/etc/pam.d`. Include this PAM module
wherever you want to require it for authentication, e.g. in
`/etc/pam.d/system-login`:

```
auth    include     system-auth-beacon
```

Careful: if your bluetooth beacon isn't discoverable, you will lock yourself out
of your system! It's probably a good idea to keep a root-shell open during
installation & testing of `pam-beacon`.

## Development

[![GoDoc](https://godoc.org/github.com/golang/gddo?status.svg)](https://godoc.org/github.com/muesli/pam-beacon)
[![Build Status](https://travis-ci.org/muesli/pam-beacon.svg?branch=master)](https://travis-ci.org/muesli/pam-beacon)
[![Go ReportCard](http://goreportcard.com/badge/muesli/pam-beacon)](http://goreportcard.com/report/muesli/pam-beacon)
