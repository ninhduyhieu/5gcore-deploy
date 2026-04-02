package ue

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"etrib5gc/logctx"
	"fmt"
	"github.com/reogac/sbi/models"
	"github.com/reogac/utils"

	"github.com/bronze1man/radius"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

// Attribute Types for EAP-AKA'
const (
	AT_RAND_ATTRIBUTE                   = 1
	AT_AUTN_ATTRIBUTE                   = 2
	AT_RES_ATTRIBUTE                    = 3
	AT_AUTS_ATTRIBUTE                   = 4
	AT_MAC_ATTRIBUTE                    = 11
	AT_NOTIFICATION_ATTRIBUTE           = 12
	AT_IDENTITY_ATTRIBUTE               = 14
	AT_KDF_INPUT_ATTRIBUTE              = 23
	AT_CLIENT_ERROR_CODE_ATTRIBUTE      = 22
	AT_KDF_ATTRIBUTE                    = 24
	EAP_AKA_PRIME_TYPENUM               = 50
	AKA_CHALLENGE_SUBTYPE               = 1
	AKA_AUTHENTICATION_REJECT_SUBTYPE   = 2
	AKA_SYNCHRONIZATION_FAILURE_SUBTYPE = 4
	AKA_NOTIFICATION_SUBTYPE            = 12
	AKA_CLIENT_ERROR_SUBTYPE            = 14
	RES_LENGTH                          = 8
)

type EapAkaPrimeAttribute struct {
	Type   uint8
	Length uint8
	Value  []byte
}

type EapAkaPrimePkt struct {
	Subtype    uint8
	Attributes map[uint8]EapAkaPrimeAttribute
	MACInput   []byte
}

func (ue *UeContext) HandleEapSession(body *models.EapSession) (output *models.EapSession, err error) {
	var paylod []byte
	if paylod, err = base64.StdEncoding.DecodeString(body.EapPayload); err != nil {
		err = utils.WrapError("EAP Payload decode", err)
		return
	}

	eapGoPkt := gopacket.NewPacket(paylod, layers.LayerTypeEAP, gopacket.Default)
	eapLayer := eapGoPkt.Layer(layers.LayerTypeEAP)

	var eapContent *layers.EAP
	var ok bool
	if eapContent, ok = eapLayer.(*layers.EAP); !ok {
		err = fmt.Errorf("Failed to extract EAP layer")
		return
	}
	var decodedPkt *EapAkaPrimePkt
	if decodedPkt = ue.DecodeEapAkaPrime(eapContent.Contents); err != nil {
		err = utils.WrapError("EAP-AKA' decode", err)
		return
	}

	if AKA_CHALLENGE_SUBTYPE != decodedPkt.Subtype {
		err = fmt.Errorf("Invalid eap subtype: expected CHALLENGE")
		return
	}

	K_autStr := ue.Kaut()
	var K_aut []byte
	if K_aut, err = hex.DecodeString(K_autStr); err != nil {
		err = utils.WrapError("K_aut decode", err)
		return
	} else {

		XMAC := ue.calculateAtMAC(K_aut, decodedPkt.MACInput)
		MAC := decodedPkt.Attributes[AT_MAC_ATTRIBUTE].Value
		XRES := ue.Xres()
		RES := hex.EncodeToString(decodedPkt.Attributes[AT_RES_ATTRIBUTE].Value)
		if !(string(XMAC) == string(MAC)) {
			err = fmt.Errorf("EAP-AKA' integrity check fail")
			return
		} else if XRES == RES {
			ue.Info("UE is authenticated")
			output = &models.EapSession{
				AuthResult: models.AUTHRESULT_AUTHENTICATION_SUCCESS,
				Supi:       ue.Supi(),
				KSeaf:      ue.Kseaf(),
				AmData:     ue.AmData(),
				EapPayload: ue.ConstructFailEapAkaNotification(eapContent.Id), //2DUC: it seems not right, you should return a success packet
			}

			//confirm authentication result to UDM
			err = ue.confirmAuth2Udm(true)
		}
	}
	return
}

func (ue *UeContext) ConstructFailEapAkaNotification(oldPktId uint8) string {
	var eapPkt radius.EapPacket
	eapPkt.Code = radius.EapCodeRequest
	eapPkt.Identifier = oldPktId + 1
	eapPkt.Type = EAP_AKA_PRIME_TYPENUM
	attrNum := fmt.Sprintf("%02x", AT_NOTIFICATION_ATTRIBUTE)
	attribute := attrNum + "01" + "4000"
	var attrHex []byte
	if attrHexTmp, err := hex.DecodeString(attribute); err != nil {
		fmt.Errorf("Decode attribute failed: %+v", err)
	} else {
		attrHex = attrHexTmp
	}
	eapPkt.Data = attrHex
	eapPktEncode := eapPkt.Encode()
	return base64.StdEncoding.EncodeToString(eapPktEncode)
}

func intToByteArray(i int) []byte {
	r := make([]byte, 2)
	binary.BigEndian.PutUint16(r, uint16(i))
	return r
}

func padZeros(byteArray []byte, size int) []byte {
	l := len(byteArray)
	if l == size {
		return byteArray
	}
	r := make([]byte, size)
	copy(r[size-l:], byteArray)
	return r
}

func (ue *UeContext) calculateAtMAC(key []byte, input []byte) []byte {
	h := hmac.New(sha256.New, key)
	if _, err := h.Write(input); err != nil {
		ue.Errorln(err.Error())
	}
	sum := h.Sum(nil)
	return sum[:16]
}

func EapEncodeAttribute(attributeType string, data string) (string, error) {
	log := logctx.WithFields(logctx.Fields{
		"mod": "eap",
	})
	var attribute string
	var length int
	switch attributeType {
	case "AT_RAND":
		length = len(data)/8 + 1
		if length != 5 {
			return "", fmt.Errorf("[eapEncodeAttribute] AT_RAND Length Error")
		}
		attrNum := fmt.Sprintf("%02x", AT_RAND_ATTRIBUTE)
		attribute = attrNum + "050010" + data

	case "AT_AUTN":
		length = len(data)/8 + 1
		if length != 5 {
			return "", fmt.Errorf("[eapEncodeAttribute] AT_AUTN Length Error")
		}
		attrNum := fmt.Sprintf("%02x", AT_AUTN_ATTRIBUTE)
		attribute = attrNum + "050010" + data

	case "AT_KDF_INPUT":
		var byteName []byte
		nLength := len(data)
		length := (nLength+3)/4 + 1
		b := make([]byte, length*4)
		byteNameLength := intToByteArray(nLength)
		byteName = []byte(data)
		pad := padZeros(byteName, (length-1)*4)
		b[0] = 23
		b[1] = byte(length)
		copy(b[2:4], byteNameLength)
		copy(b[4:], pad)
		return string(b[:]), nil

	case "AT_KDF":
		// Value 1 default key derivation function for EAP-AKA'
		attrNum := fmt.Sprintf("%02x", AT_KDF_ATTRIBUTE)
		attribute = attrNum + "01" + "0001"

	case "AT_MAC":
		// Pad MAC value with 16 bytes of 0 since this is just for the calculation of MAC
		attrNum := fmt.Sprintf("%02x", AT_MAC_ATTRIBUTE)
		attribute = attrNum + "05" + "0000" + "00000000000000000000000000000000"
	case "MAC_LEN":
		attrNum := fmt.Sprintf("%02x", AT_MAC_ATTRIBUTE)
		attribute = attrNum + "050010"

	case "AT_RES":
		var byteName []byte
		nLength := len(data)
		length := (nLength+3)/4 + 1
		b := make([]byte, length*4)
		byteNameLength := intToByteArray(nLength)
		byteName = []byte(data)
		pad := padZeros(byteName, (length-1)*4)
		b[0] = 3
		b[1] = byte(length)
		copy(b[2:4], byteNameLength)
		copy(b[4:], pad)
		return string(b[:]), nil

	default:
		log.Errorf("UNKNOWN attributeType %s\n", attributeType)
		return "", nil
	}

	if r, err := hex.DecodeString(attribute); err != nil {
		return "", err
	} else {
		return string(r), nil
	}
}

func eapAkaPrimePrf(ikPrime string, ckPrime string, identity string) ([]byte, []byte, []byte, []byte, []byte) {
	log := logctx.WithFields(logctx.Fields{
		"mod": "eap",
	})
	keyAp := ikPrime + ckPrime
	var key []byte
	if keyTmp, err := hex.DecodeString(keyAp); err != nil {
		log.Errorf("Decode key AP failed: %+v", err)
	} else {
		key = keyTmp
	}
	sBase := []byte("EAP-AKA'" + identity)
	MK := []byte("")
	prev := []byte("")
	//_ = prev
	prfRounds := 208/32 + 1
	for i := 0; i < prfRounds; i++ {
		// Create a new HMAC by defining the hash type and the key (as byte array)
		h := hmac.New(sha256.New, key)
		hexNum := (byte)(i + 1)
		ap := append(sBase, hexNum)
		s := append(prev, ap...)
		// Write Data to it
		if _, err := h.Write(s); err != nil {
			log.Errorf(err.Error())
		}
		// Get result
		sha := h.Sum(nil)
		MK = append(MK, sha...)
		prev = sha
	}
	K_encr := MK[0:16]  // 0..127
	K_aut := MK[16:48]  // 128..383
	K_re := MK[48:80]   // 384..639
	MSK := MK[80:144]   // 640..1151
	EMSK := MK[144:208] // 1152..1663
	return K_encr, K_aut, K_re, MSK, EMSK
}

func (ue *UeContext) DecodeEapAkaPrime(eapPkt []byte) *EapAkaPrimePkt {
	log := logctx.WithFields(logctx.Fields{
		"mod": "eap",
	})

	var decodePkt EapAkaPrimePkt
	var attrLen int
	var decodeAttr EapAkaPrimeAttribute
	attributes := make(map[uint8]EapAkaPrimeAttribute)
	data := eapPkt[5:]
	decodePkt.Subtype = data[0]
	dataLen := len(data)
	// decode attributes
	for i := 3; i < dataLen; i += attrLen {
		attrType := data[i]
		attrLen = int(data[i+1]) * 4
		if attrLen == 0 {
			ue.Errorf("attribute length equal to zero")
			return nil
		}
		if i+attrLen > dataLen {
			ue.Errorf("packet length out of range")
			return nil
		}
		switch attrType {
		case AT_RES_ATTRIBUTE:
			accLen := RES_LENGTH
			decodeAttr.Type = attrType
			decodeAttr.Length = data[i+1]
			decodeAttr.Value = data[i+4 : i+4+accLen]
			//ue.Infof("value res :%s",hex.EncodeToString(decodeAttr.Value))
			attributes[attrType] = decodeAttr
		case AT_MAC_ATTRIBUTE:
			if attrLen != 20 {
				ue.Errorf("attribute AT_MAC decode err")
				return nil
			}
			decodeAttr.Type = attrType
			decodeAttr.Length = data[i+1]
			Mac := make([]byte, attrLen-4)
			copy(Mac, data[i+4:i+attrLen])
			decodeAttr.Value = Mac
			attributes[attrType] = decodeAttr
			// clean AT_MAC value for integrity check later
			zeros := make([]byte, attrLen-4)
			copy(data[i+4:i+attrLen], zeros)
			decodePkt.MACInput = eapPkt
		case AT_KDF_ATTRIBUTE:
			if attrLen != 4 {
				fmt.Errorf("attribute AT_KDF decode err")
				return nil
			}
			decodeAttr.Type = attrType
			decodeAttr.Length = data[i+1]
			decodeAttr.Value = data[i+2 : i+attrLen]
			attributes[attrType] = decodeAttr
		case AT_AUTS_ATTRIBUTE:
			log.Infof("Decoding AT_AUTS\n")
			if attrLen != 16 {
				ue.Errorf("attribute AT_AUTS decode err")
				return nil
			}
			decodeAttr.Type = attrType
			decodeAttr.Length = data[i+1]
			decodeAttr.Value = data[i+2 : i+attrLen]
			attributes[attrType] = decodeAttr
		case AT_CLIENT_ERROR_CODE_ATTRIBUTE:
			log.Infof("Decoding AT_CLIENT_ERROR_CODE\n")
			if attrLen != 4 {
				ue.Errorf("attribute AT_CLIENT_ERROR_CODE decode err")
				return nil
			}
			decodeAttr.Type = attrType
			decodeAttr.Length = data[i+1]
			decodeAttr.Value = data[i+2 : i+attrLen]
			attributes[attrType] = decodeAttr
		default:
			log.Infof("attribute type %x skipped\n", attrType)
		}
	}

	switch decodePkt.Subtype {
	case AKA_CHALLENGE_SUBTYPE:
		log.Infof("Subtype AKA-Challenge\n")
		if _, ok := attributes[AT_RES_ATTRIBUTE]; !ok {
			ue.Errorf("AKA-Challenge attributes error")
			return nil
		} else if _, ok := attributes[AT_MAC_ATTRIBUTE]; !ok {
			ue.Errorf("AKA-Challenge attributes error")
			return nil
		}
	case AKA_AUTHENTICATION_REJECT_SUBTYPE:
		log.Infof("Subtype AKA-Authentication-Reject\n")
		if len(attributes) != 0 {
			ue.Errorf("AKA-Authentication-Reject attributes error")
			return nil
		}
	case AKA_SYNCHRONIZATION_FAILURE_SUBTYPE:
		log.Infof("Subtype AKA-Synchronization-Failure\n")
		if len(attributes) != 2 {
			ue.Errorf("AKA-Synchornization-Failure attributes error")
			return nil
		} else if _, ok := attributes[AT_AUTS_ATTRIBUTE]; !ok {
			ue.Errorf("AKA-Synchornization-Failure attributes error")
			return nil
		} else if _, ok := attributes[AT_KDF_ATTRIBUTE]; !ok {
			ue.Errorf("AKA-Synchornization-Failure attributes error")
			return nil
		} else if kdfVal := attributes[AT_KDF_ATTRIBUTE].Value; !(kdfVal[0] == 0 && kdfVal[1] == 1) {
			ue.Errorf("AKA-Synchornization-Failure attributes error")
			return nil
		}
	case AKA_NOTIFICATION_SUBTYPE:
		log.Infof("Subtype AKA-Notification\n")
	case AKA_CLIENT_ERROR_SUBTYPE:
		log.Infof("Subtype AKA-Client-Error\n")
		if len(attributes) != 1 {
			ue.Errorf("AKA-Client-Error attributes error")
			return nil
		} else if _, ok := attributes[AT_CLIENT_ERROR_CODE_ATTRIBUTE]; !ok {
			ue.Errorf("AKA-Client-Error attributes error")
			return nil
		}
	default:
		log.Infof("subtype %x skipped\n", decodePkt.Subtype)
	}

	decodePkt.Attributes = attributes

	return &decodePkt
}
