package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"strconv"
	"strings"

	hid "github.com/karalabe/hid"
)

const ledgerVendorID = 11415

func main() {
	devs := hid.Enumerate(ledgerVendorID, 0)

	if len(devs) == 0 {
		fmt.Println("no ledger found")
		return
	}
	if len(devs) > 1 {
		fmt.Printf("warning: %d ledgers found, using first\n", len(devs))
		fmt.Println("===========\nif you don't actually have two ledgers plugged in,\nthen you might get random errors. No idea why this happens yet\n===========")
	}

	dev := devs[0]

	odev, err := dev.Open()
	if err != nil {
		panic(err)
	}
	defer odev.Close()

	l := &ledger{odev}

	switch os.Args[1] {
	case "btcaddr":
		if len(os.Args) < 3 {
			fmt.Println("must pass a path to derive")
		}
		path, err := parseHDPath(os.Args[2])
		if err != nil {
			fmt.Println(err)
			return
		}
		_, addr, err := l.getBitcoinAddress(path)
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Println(addr)
	case "ethaddr":
		if len(os.Args) < 3 {
			fmt.Println("must pass a path to derive")
		}
		path, err := parseHDPath(os.Args[2])
		if err != nil {
			fmt.Println(err)
			return
		}
		_, addr, err := l.getEthereumAddress(path)
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Println(addr)
	}
}

func parseHDPath(s string) ([]uint32, error) {
	if len(s) < 3 {
		return nil, fmt.Errorf("paths must be at least three characters long ('m/1')")
	}
	if s[0] != 'm' {
		return nil, fmt.Errorf("bip32 paths must start with m")
	}

	parts := strings.Split(s[2:], "/")
	var out []uint32
	for _, p := range parts {
		var hardened bool
		if strings.HasSuffix(p, "'") {
			hardened = true
			p = p[:len(p)-1]
		}
		val, err := strconv.ParseUint(p, 10, 32)
		if err != nil {
			return nil, err
		}

		if hardened {
			val = val | 0x80000000
		}
		out = append(out, uint32(val))
	}
	return out, nil
}

type ledger struct {
	odev *hid.Device
}

func (l *ledger) readKeyRoutine(t byte, path []uint32) ([]byte, string, error) {
	cmd := []byte{0xe0, 0x02, 1, 1}

	params := encodePath(path)
	cmd = append(cmd, byte(len(params)))
	cmd = append(cmd, params...)

	resp, err := l.Exchange(cmd)
	if err != nil {
		return nil, "", err
	}

	if len(resp) <= 2 {
		msg := fmt.Sprintf("error code 0x%x", resp)
		if bytes.Equal(resp, []byte{0x69, 0x82}) {
			msg += ": device is locked"
		}
		return nil, "", fmt.Errorf(msg)
	}

	pubklen := resp[0]
	pubk := resp[1 : 1+pubklen]

	resp = resp[1+pubklen:]

	addrlen := resp[0]
	addr := resp[1 : 1+addrlen]

	return pubk, string(addr), nil
}

func encodePath(p []uint32) []byte {
	enc := make([]byte, 1+(4*len(p)))
	enc[0] = byte(len(p))
	for i, p := range p {
		binary.BigEndian.PutUint32(enc[1+(4*i):], p)
	}

	return enc
}

func (l *ledger) getBitcoinAddress(odevpath []uint32) ([]byte, string, error) {
	return l.readKeyRoutine(0x40, odevpath)
}

// TODO: doesnt work for some reason...
func (l *ledger) getEthereumAddress(path []uint32) ([]byte, string, error) {
	return l.readKeyRoutine(0x02, path)
}
