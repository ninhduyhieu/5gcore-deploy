package ngapconv

import (
	"encoding/hex"
	"fmt"
	"github.com/lvdund/ngap/aper"
)

func BitStringToHex(bitString *aper.BitString) (hexString string) {
	hexString = hex.EncodeToString(bitString.Bytes)
	hexLen := (bitString.NumBits + 3) / 4
	hexString = hexString[:hexLen]
	return
}

func HexToBitString(hexString string, bitLength int) (bitString aper.BitString, err error) {
	hexLen := len(hexString)
	if hexLen != (bitLength+3)/4 {
		err = fmt.Errorf("hexLen[%d] doesn't match bitLength[%d]", hexLen, bitLength)
		return
	}
	if hexLen%2 == 1 {
		hexString += "0"
	}
	var byteTmp []byte
	if byteTmp, err = hex.DecodeString(hexString); err != nil {
		return
	} else {
		bitString.Bytes = byteTmp
	}
	bitString.NumBits = uint64(bitLength)
	mask := byte(0xff)
	mask = mask << uint(8-bitLength%8)
	if mask != 0 {
		bitString.Bytes[len(bitString.Bytes)-1] &= mask
	}
	return
}

func ByteToBitString(byteArray []byte, bitLength int) (bitString aper.BitString, err error) {
	byteLen := (bitLength + 7) / 8
	if byteLen > len(byteArray) {
		err = fmt.Errorf("bitLength[%d] is beyond byteArray size[%d]", bitLength, len(byteArray))
		return
	}
	bitString.Bytes = byteArray
	bitString.NumBits = uint64(bitLength)
	return
}
