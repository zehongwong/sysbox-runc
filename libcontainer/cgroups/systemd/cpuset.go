package systemd

import (
	"encoding/binary"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/willf/bitset"
)

// rangeToBits converts a text representation of a CPU mask (as written to
// or read from cgroups' cpuset.* files, e.g. "1,3-5") to a slice of bytes
// with the corresponding bits set (as consumed by systemd over dbus as
// AllowedCPUs/AllowedMemoryNodes unit property value).
func rangeToBits(str string) ([]byte, error) {
	bits := &bitset.BitSet{}

	for _, r := range strings.Split(str, ",") {
		// allow extra spaces around
		r = strings.TrimSpace(r)
		// allow empty elements (extra commas)
		if r == "" {
			continue
		}
		ranges := strings.SplitN(r, "-", 2)
		if len(ranges) > 1 {
			start, err := strconv.ParseUint(ranges[0], 10, 32)
			if err != nil {
				return nil, err
			}
			end, err := strconv.ParseUint(ranges[1], 10, 32)
			if err != nil {
				return nil, err
			}
			if start > end {
				return nil, errors.New("invalid range: " + r)
			}
			for i := uint(start); i <= uint(end); i++ {
				bits.Set(i)
			}
		} else {
			val, err := strconv.ParseUint(ranges[0], 10, 32)
			if err != nil {
				return nil, err
			}
			bits.Set(uint(val))
		}
	}

	words := bits.Bytes() // []uint64, word 0 holds bits 0..63
	if len(words) == 0 {
		return nil, errors.New("empty value")
	}

	// Little-endian byte stream
	// ref: https://github.com/systemd/systemd/blob/v259/src/shared/cpu-set-util.c#L363-L398
	//
	// For example: CPUs: 6-9
	// words[0] = 00000000_00000000_00000000_00000000_00000000_00000000_00000011_11000000
	// Bytes:   [0]       [1]       [2]       [3]       [4]       [5]       [6]       [7]
	//          11000000  00000011  00000000  00000000  00000000  00000000  00000000  00000000
	// Return:  [0xC0, 0x03]
	ret := make([]byte, len(words)*8)
	for i := range words {
		binary.LittleEndian.PutUint64(ret[i*8:], words[i])
	}

	// Trim trailing zero bytes
	for len(ret) > 0 && ret[len(ret)-1] == 0 {
		ret = ret[:len(ret)-1]
	}

	return ret, nil
}
