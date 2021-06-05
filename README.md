# pam-beacon

[![Latest Release](https://img.shields.io/github/release/muesli/pam-beacon.svg)](https://github.com/muesli/pam-beacon/releases)
[![Build Status](https://github.com/muesli/pam-beacon/workflows/build/badge.svg)](https://github.com/muesli/pam-beacon/actions)
[![Go ReportCard](https://goreportcard.com/badge/muesli/pam-beacon)](https://goreportcard.com/report/muesli/pam-beacon)
[![GoDoc](https://godoc.org/github.com/golang/gddo?status.svg)](https://pkg.go.dev/github.com/muesli/pam-beacon)

PAM module for (multi-factor) authentication with Bluetooth Devices & Beacons

## Installation

### Packages

ArchLinux: install the AUR package `pam_beacon-git`.

### Manually

Make sure you have a working Go environment (Go 1.11 or higher is required).
See the [install instructions](http://golang.org/doc/install.html).
`libpam` and its development headers are also required.

```
$ make
$ sudo make install
```

## Configuration

Create a file named `.authorized_beacons` in your home directory. You can put
the MAC addresses of as many beacons in it as you like, one per line. The auth
will succeed as long as `pam-beacon` can find at least one of the beacons.

Example:

```
00:11:22:AA:BB:CC
FF:EE:DD:99:88:77
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
