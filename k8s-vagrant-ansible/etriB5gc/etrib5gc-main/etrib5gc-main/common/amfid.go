package common

import (
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
)

type AmfSet struct {
	Region uint8  `json:"region"`
	Set    uint16 `json:"set"`
}

func (s *AmfSet) String() string {
	return fmt.Sprintf("%d-%d", s.Region, s.Set)
}

func AmfPointerString(pointer uint8) string {
	return fmt.Sprintf("%d", pointer)
}

func AmfSetFromString(s string) (id *AmfSet, err error) {
	tokens := strings.Split(s, "-")
	if len(tokens) != 2 {
		err = fmt.Errorf("Invalid AmfSet format: %s", s)
		return
	}
	value := AmfSet{}
	var v uint64
	if v, err = strconv.ParseUint(tokens[0], 10, 8); err != nil {
		return
	}
	value.Region = uint8(v)

	if v, err = strconv.ParseUint(tokens[1], 10, 16); err != nil {
		return
	}
	if v >= uint64(MAX_10_BITS) {
		err = fmt.Errorf("AmfSet must be 10bits")
		return
	}
	value.Set = uint16(v)
	id = &value
	return
}

// write AmfId in hex string
func AmfIdString(amfSet AmfSet, pointer uint8) (amfId string) {
	id := AmfId{
		amfSet:     amfSet,
		amfPointer: pointer,
	}
	return id.String()
}

type AmfId struct {
	amfSet     AmfSet
	amfPointer uint8
}

func NewAmfId(amfSet AmfSet, amfPointer uint8) AmfId {
	return AmfId{
		amfSet:     amfSet,
		amfPointer: amfPointer,
	}
}

func (id *AmfId) Set() *AmfSet {
	return &id.amfSet
}

func (id *AmfId) Pointer() uint8 {
	return id.amfPointer
}

func ParseAmfId(amfIdStr string) (amfId *AmfId, err error) {
	var buf []byte
	if buf, err = hex.DecodeString(amfIdStr); err != nil {
		return
	}
	if len(buf) != 3 {
		err = fmt.Errorf("AmfId length must be 3 bytes")
		return
	}
	amfId = &AmfId{
		amfSet: AmfSet{
			Region: buf[0],
			Set:    uint16(buf[1])<<2 + (uint16(buf[2])&0x00c0)>>6,
		},
		amfPointer: buf[2] & 0x3f,
	}
	return
}

func (id *AmfId) Bytes() (b []byte, err error) {
	var buf [3]byte
	if id.amfSet.Set >= MAX_10_BITS {
		err = fmt.Errorf("AmfSet must be a 10-bit number")
		return
	}

	if id.amfPointer >= MAX_6_BITS {
		err = fmt.Errorf("AmfPointer must be a 6-bit number")
		return
	}
	buf[0] = id.amfSet.Region
	buf[1] = uint8(id.amfSet.Set>>2) & 0xff
	buf[2] = uint8(id.amfSet.Set&0x03)<<6 + id.amfPointer&0x3f
	b = buf[:]
	return
}

func (id *AmfId) String() string {
	if b, err := id.Bytes(); err == nil {
		return hex.EncodeToString(b)
	}
	return ""
}
