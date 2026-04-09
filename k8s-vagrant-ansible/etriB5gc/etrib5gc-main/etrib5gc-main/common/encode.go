package common

import (
	"encoding/hex"
	"fmt"
)

const (
	MAX_12_BITS uint16 = 1 << 12
	MAX_10_BITS uint16 = 1 << 10
	MAX_6_BITS  uint8  = 1 << 6
)

// ShiftLeft performs a left bit shift operation on the provided bytes.
// If the bits count is negative, a right bit shift is performed.
func ShiftLeft(data []byte, bits int) (ret []byte) {
	n := len(data)
	ret = make([]byte, n)
	if bits < 0 {
		bits = -bits
		for i := n - 1; i > 0; i-- {
			ret[i] = data[i]>>bits | data[i-1]<<(8-bits)
		}
		ret[0] = data[0] >> bits
	} else {
		for i := 0; i < n-1; i++ {
			ret[i] = data[i]<<bits | data[i+1]>>(8-bits)
		}
		ret[n-1] = data[n-1] << bits
	}
	return
}

func loadHex(v []byte, str string) (err error) {
	var tmp []byte
	if tmp, err = hex.DecodeString(str); err != nil {
		return
	}
	if len(tmp) != len(v) {
		err = fmt.Errorf("input(len=%d) must be %d octets length", len(tmp), len(v))
		return
	}
	copy(v[:], tmp)
	return
}

/*
type _PlmnId [3]byte //mcc=12bits, mnc=12bits

func (id *_PlmnId) SetHex(str string) error {
	return loadHex(id[:], str)
}

// mcc, mnc should not be bigger than 12-bit number
func (id *_PlmnId) Set(mcc, mnc uint16) (err error) {
	if mcc >= MAX_12_BITS {
		err = fmt.Errorf("MCC must be a 12-bit number")
		return
	}
	if mnc >= MAX_12_BITS {
		err = fmt.Errorf("MNC must be a 12-bit number")
		return
	}

	var tmp [4]byte
	binary.BigEndian.PutUint16(tmp[:], mcc)
	binary.BigEndian.PutUint16(tmp[2:], mnc)
	copy(tmp[2:], ShiftLeft(tmp[2:], 4))
	copy(id[:], ShiftLeft(tmp[:], 4)[:3])
	return
}
func (id _PlmnId) Mcc() uint16 {
	mcc := ShiftLeft(id[:2], -4)
	return binary.BigEndian.Uint16(mcc)
}
func (id _PlmnId) Mnc() uint16 {
	mnc := ShiftLeft(id[1:], 4)
	return binary.BigEndian.Uint16(mnc) / 16
}

type _Guami [6]byte //plmnid+amfid
func (id *_Guami) Set(plmnid _PlmnId, amfid _AmfId) {
	copy(id[:], plmnid[:])
	copy(id[3:], amfid[:])
}
func (id *_Guami) SetHex(str string) error {
	return loadHex(id[:], str)
}

func (id _Guami) PlmnId() (plmnid _PlmnId) {
	copy(plmnid[:], id[:3])
	return
}

func (id _Guami) AmfId() (amfid _AmfId) {
	copy(amfid[:], id[3:])
	return
}
func (id _Guami) String() string {
	return hex.EncodeToString(id[:])
}

type _Guti [10]byte //guami+tmsi

func (id *_Guti) Set(guami _Guami, tmsi Tmsi) {
	copy(id[:], guami[:])
	copy(id[6:], tmsi[:])
	return
}
func (id *_Guti) SetHex(str string) error {
	return loadHex(id[:], str)
}
func (id _Guti) String() string {
	return hex.EncodeToString(id[:])
}
type _Snssai [4]byte //sst=8bits, sd=24bits

func (id _Snssai) Sst() uint8 {
	return id[0]
}

func (id _Snssai) Sd() uint32 {
	var tmp [4]byte = id
	id[0] = 0
	return binary.BigEndian.Uint32(tmp[:])
}

func (id *_Snssai) Set(sst uint8, sd uint32) {
	binary.BigEndian.PutUint32(id[:], sd)
	id[0] = sst
}

func (id *_Snssai) SetHex(str string) error {
	return loadHex(id[:], str)
}

type Tac [3]byte
type Tai [6]byte //plmnid+tac
*/
