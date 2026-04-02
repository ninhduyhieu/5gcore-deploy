package ngapconv

import (
	"encoding/hex"
	"github.com/lvdund/ngap/aper"
)

func AmfIdToNgap(amfId string) (regionId, setId, ptrId aper.BitString, err error) {
	if regionId, err = HexToBitString(amfId[:2], 8); err != nil {
		return
	}
	if setId, err = HexToBitString(amfId[2:5], 10); err != nil {
		return
	}

	var tmpByte []byte
	if tmpByte, err = hex.DecodeString(amfId[4:]); err != nil {
		return
	}
	var shiftByte []byte
	if shiftByte, err = aper.GetBitString(tmpByte, 2, 6); err != nil {
		return
	}
	ptrId.NumBits = 6
	ptrId.Bytes = shiftByte
	return
}

func AmfIdToString(regionId, setId, ptrId aper.BitString) (amfId string) {
	regionHex := BitStringToHex(&regionId)
	tmpByte := []byte{setId.Bytes[0], (setId.Bytes[1] & 0xc0) | (ptrId.Bytes[0] >> 2)}
	restHex := hex.EncodeToString(tmpByte)
	amfId = regionHex + restHex
	return
}
