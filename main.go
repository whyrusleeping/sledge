package main

import (
	"fmt"
	"encoding/binary"
	"bytes"

	hid "github.com/karalabe/hid"
)

const ledgerVendorID = 11415

func APDUEncode(data []byte) []byte {
	if len(data) > 59 {
		panic("no bueno")
		// TODO: implement message chunking
	}
	buf := make([]byte, 64)

	// the channel
	binary.BigEndian.PutUint16(buf, 0x0101)

	// 
	buf[2] = 5

	// sequence ID (uint16)
	binary.BigEndian.PutUint16(buf[3:], 0)

	// command length (uint16)
	binary.BigEndian.PutUint16(buf[5:], uint16(len(data)))

	copy(buf[7:], data)


	return buf
}

const packetSize = 64


func readAPDUPrefix(data []byte) (uint16, byte, uint16) {
	channel := binary.BigEndian.Uint16(data[:2])
	tag := data[2]
	seq := binary.BigEndian.Uint16(data[3:])
	return channel, tag, seq
}

func APDUDecode(data []byte) []byte {
	fmt.Printf("DATA: %x\n", data)
	ch, tag, seq := readAPDUPrefix(data)
	if ch != 0x0101 {
		panic("expected channel to be 0x0101")
	}
	if tag != 5 {
		panic("expected tag to be 5")
	}
	if seq != 0 {
		fmt.Println(seq)
		panic("expected sequence number to be zero")
	}

	rlen := binary.BigEndian.Uint16(data[5:])
	fmt.Println("rlen:",  rlen)
	fmt.Printf("DATAS: %x\n", data[7:])

	blockSize := rlen
	if blockSize > packetSize - 7 {
		blockSize = packetSize - 7
	}

	out := new(bytes.Buffer)

	out.Write(data[7:7+blockSize])

	data = data[packetSize:]

	for uint16(out.Len()) != rlen {
		fmt.Println("looping", out.Len(), rlen, len(data))
		seq++
		ch, tag, seqn := readAPDUPrefix(data)
		if ch != 0x0101 {
			panic("bad channel! bad!")
		}
		if tag != 5 {
			panic("that fuckin tag wasnt five. Don't know why its bad tho")
		}
		if seqn != seq {
			panic("sequence number mismatch")
		}

		var blockSize uint16
		if rlen - uint16(out.Len()) > packetSize - 5 {
			blockSize = packetSize - 5
		} else {
			blockSize = rlen - uint16(out.Len())
		}

		out.Write(data[5:5+blockSize])
	}

	return out.Bytes()
}

func main() {
	devs := hid.Enumerate(ledgerVendorID,0)

	if len(devs) == 0 {
		fmt.Println("no ledger found")
		return
	}
	if len(devs) > 1 {
		fmt.Printf("%d ledgers found, using first\n", len(devs))
		//fmt.Println(devs)
		//return
	}

	dev := devs[0]

	odev, err := dev.Open()
	if err != nil {
		panic(err)
	}
	defer odev.Close()

	l := &ledger{odev}

	pubk, addr, _ := l.getPublicKey()
	fmt.Println("pubkey: ", pubk)
	fmt.Println("addr: ", addr)
}

type ledger struct {
	odev *hid.Device
}

func (l *ledger) getPublicKey(odevpath ...uint32) ([]byte, string, []byte) {
	cmd := []byte{0xE0, 0x40, 0, 0, 5, 1, 0,0,0,1}
	//cmd := []byte{0xE0, 0x24, 0, 0, 0, 1}
	//cmd := []byte{ 0x80, 0x02, 0xF0, 0x0D}

	_, err := l.odev.Write(APDUEncode(cmd))
	if err != nil {
		panic(err)
	}


	out := make([]byte, 256)
	n, err := l.odev.Read(out)
	if err != nil {
		panic(err)
	}
	fmt.Println("read this many bytes: ", n)

	resp := APDUDecode(out[:n])
	fmt.Println("resp len: ", len(resp))

	pubklen := resp[0]
	pubk := resp[1:1+pubklen]

	resp = resp[1+pubklen:]

	addrlen := resp[0]
	addr := resp[1:1+addrlen]

	resp = resp[1:1+addrlen]
	fmt.Println("expect this to be 32: ", len(resp))
	return pubk, string(addr), nil
}
