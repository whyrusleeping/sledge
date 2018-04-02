package main

import (
	"encoding/binary"
	"io"
	"fmt"
	"bytes"
)

/* References:
- http://www.tml.tkk.fi/Studies/T-110.497/2003/lecture4.pdf
- https://ledgerhq.github.io/btchip-doc/bitcoin-technical.html
*/
const packetSize = 64

func (l *ledger) Exchange(cmd []byte) ([]byte, error) {
	_, err := l.odev.Write(APDUEncode(cmd))
	if err != nil {
		return nil, err
	}

	return APDUDecode(l.odev)
}

func APDUEncode(data []byte) []byte {
	if len(data) > 59 {
		panic("no bueno")
		// TODO: implement message chunking
	}
	buf := make([]byte, 64)

	// the channel
	binary.BigEndian.PutUint16(buf, 0x0101)

	// "its five, okay?"
	buf[2] = 5

	// sequence ID (uint16)
	binary.BigEndian.PutUint16(buf[3:], 0)

	// command length (uint16)
	binary.BigEndian.PutUint16(buf[5:], uint16(len(data)))

	copy(buf[7:], data)

	return buf
}



func readAPDUPrefix(data []byte) (uint16, byte, uint16) {
	channel := binary.BigEndian.Uint16(data[:2])
	tag := data[2]
	seq := binary.BigEndian.Uint16(data[3:])
	return channel, tag, seq
}

func APDUDecode(r io.Reader) ([]byte, error) {
	data := make([]byte, packetSize)
	_, err := io.ReadFull(r, data)
	if err != nil {
		return nil, err
	}

	ch, tag, seq := readAPDUPrefix(data)
	if ch != 0x0101 {
		return nil, fmt.Errorf("expected channel to be 0x0101")
	}
	if tag != 5 {
		return nil,fmt.Errorf("expected tag to be 5")
	}
	if seq == 0xbf {
		// sometimes i get this. To the best of my knowledge, it just means "you suck, try again".
		// fmt.Println("encountered the weird seqno error.")
		// fmt.Println("i suspect this is because your OS is showing multiple ledgers,\nyou only have one plugged in, and we randomly picked the 'wrong' one")
		return nil, fmt.Errorf("sequence 0xbf. Abort.")
	}
	if seq != 0 {
		return nil, fmt.Errorf("got non-zero sequence number initially: %d", seq)
	}

	rlen := binary.BigEndian.Uint16(data[5:])

	blockSize := rlen
	if blockSize > packetSize - 7 {
		blockSize = packetSize - 7
	}

	out := new(bytes.Buffer)
	out.Write(data[7:7+blockSize])

	for uint16(out.Len()) != rlen {
		_, err := io.ReadFull(r, data)
		if err != nil {
			return nil, err
		}

		seq++
		ch, tag, seqn := readAPDUPrefix(data)
		if ch != 0x0101 {
		return nil, fmt.Errorf("expected channel to be 0x0101")
	}
	if tag != 5 {
		return nil,fmt.Errorf("expected tag to be 5")
		}
		if seqn != seq {
			return nil, fmt.Errorf("invalid sequence number")
		}

		var blockSize uint16
		if rlen - uint16(out.Len()) > packetSize - 5 {
			blockSize = packetSize - 5
		} else {
			blockSize = rlen - uint16(out.Len())
		}

		out.Write(data[5:5+blockSize])
	}

	return out.Bytes(), nil
}

