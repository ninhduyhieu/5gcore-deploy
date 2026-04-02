package suci

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/elliptic"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"github.com/reogac/nas"
	"math"
	"math/big"
	"math/bits"

	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/curve25519"
)

var log *logrus.Entry

func init() {
	log = logrus.WithFields(logrus.Fields{"mod": "suci"})
}

// suci-mcc-mnc-routingIndentifier-protectionScheme-homeNetworkPublicKeyIdentifier-schemeOutput.

//const from nas
//ProtectionSchemeNullScheme    uint8 = 0
//ProtectionSchemeECIESProfileA uint8 = 1
//ProtectionSchemeECIESProfileB uint8 = 2

type Profile struct {
	ProtectionScheme int    `json:"scheme,omitempty"`
	PrivateKey       string `json:"prvKey,omitempty"`
	PublicKey        string `json:"pubKey,omitempty"`
}

// profile A.
const (
	A_MAC_K_LEN = 32 // octets
	A_ENC_K_LEN = 16 // octets
	A_ICB_LEN   = 16 // octets
	A_MAC_LEN   = 8  // octets
	A_HASH_LEN  = 32 // octets
)

// profile B.
const (
	B_MAC_K_LEN = 32 // octets
	B_ENC_K_LEN = 16 // octets
	B_ICB_LEN   = 16 // octets
	B_MAC_LEN   = 8  // octets
	B_HASH_LEN  = 32 // octets
)

func CompressKey(uncompressed []byte, y *big.Int) []byte {
	compressed := uncompressed[0:33]
	if y.Bit(0) == 1 { // 0x03
		compressed[0] = 0x03
	} else { // 0x02
		compressed[0] = 0x02
	}
	return compressed
}

// modified from https://stackoverflow.com/questions/46283760/
// how-to-uncompress-a-single-x9-62-compressed-point-on-an-ecdh-p256-curve-in-go.
func uncompressKey(compressedBytes []byte, priv []byte) (*big.Int, *big.Int) {
	// Split the sign byte from the rest
	signByte := uint(compressedBytes[0])
	xBytes := compressedBytes[1:]

	x := new(big.Int).SetBytes(xBytes)
	three := big.NewInt(3)

	// The params for P256
	c := elliptic.P256().Params()

	// The equation is y^2 = x^3 - 3x + b
	// x^3, mod P
	xCubed := new(big.Int).Exp(x, three, c.P)

	// 3x, mod P
	threeX := new(big.Int).Mul(x, three)
	threeX.Mod(threeX, c.P)

	// x^3 - 3x + b mod P
	ySquared := new(big.Int).Sub(xCubed, threeX)
	ySquared.Add(ySquared, c.B)
	ySquared.Mod(ySquared, c.P)

	// find the square root mod P
	y := new(big.Int).ModSqrt(ySquared, c.P)
	if y == nil {
		// If this happens then you're dealing with an invalid point.
		log.Error("Uncompressed key with invalid point")
		return nil, nil
	}

	// Finally, check if you have the correct root. If not you want -y mod P
	if y.Bit(0) != signByte&1 {
		y.Neg(y)
		y.Mod(y, c.P)
	}
	return x, y
}

func HmacSha256(input, macKey []byte, macLen int) (tag []byte, err error) {
	h := hmac.New(sha256.New, macKey)
	if _, err = h.Write(input); err != nil {
		log.Errorf("HMAC SHA256 error %+v", err)
		return
	}
	mac := h.Sum(nil)
	tag = mac[:macLen]
	return
}

func Aes128ctr(input, encKey, icb []byte) (output []byte, err error) {
	output = make([]byte, len(input))
	var block cipher.Block
	if block, err = aes.NewCipher(encKey); err != nil {
		log.Errorf("AES128 CTR error %+v", err)
		return
	}
	stream := cipher.NewCTR(block, icb)
	stream.XORKeyStream(output, input)
	return
}

func AnsiX963KDF(sharedKey, pubKey []byte, encKeylen, macKeylen, hashlen int) (kdfKey []byte) {
	var counter uint32 = 0x00000001
	numrounds := int(math.Ceil(float64(encKeylen+macKeylen) / float64(hashlen)))
	for i := 1; i <= numrounds; i++ {
		counterbytes := make([]byte, 4)
		binary.BigEndian.PutUint32(counterbytes, counter)
		tmpk := sha256.Sum256(append(append(sharedKey, counterbytes...), pubKey...))
		slicek := tmpk[:]
		kdfKey = append(kdfKey, slicek...)
		counter++
	}
	return
}

func swapnibbles(input []byte) []byte {
	output := make([]byte, len(input))
	for i, b := range input {
		output[i] = bits.RotateLeft8(b, 4)
	}
	return output
}

func decompose(input []byte, keyLen int, macLen int) (mac []byte, pubKey []byte, ciphertext []byte, err error) {
	if len(input) < keyLen+macLen {
		log.Error("len of input data is too short!")
		err = fmt.Errorf("suci input too short\n")
		return
	}
	pubKey = input[:keyLen]
	mac = input[len(input)-macLen:]
	ciphertext = input[keyLen : len(input)-macLen]
	return
}

func RecoverSupi(suciStr string, profiles []Profile) (supi string, err error) {
	//parse suci
	suci := new(nas.Suci)
	if err = suci.Parse(suciStr); err != nil {
		err = fmt.Errorf("Fail to parse suci: %+v", err)
		return
	}

	if suci.GetSupiFormat() != nas.SupiFormatImsi { //not concealed
		supi = suci.String()
		return
	}

	//it is a imsi typed suci, let's process
	concealedSupi := suci.Content.(*nas.SupiImsi)
	if concealedSupi.ProtectionScheme == nas.ProtectionSchemeNullScheme { //not concealed
		supi = "imsi-" + nas.GetSupiString(concealedSupi)
		return
	}

	var keyIndex int
	keyIndex = int(concealedSupi.HomeNetworkPublicKeyId)

	if keyIndex > len(profiles) {
		err = fmt.Errorf("keyIndex(%d) out of range(%d)", keyIndex, len(profiles))
		return
	}

	profile := profiles[keyIndex-1]

	scheme := concealedSupi.ProtectionScheme
	if scheme != uint8(profile.ProtectionScheme) {
		err = fmt.Errorf("Protect Scheme mismatch [%d:%d]", scheme, profile.ProtectionScheme)
		return
	}

	var prvKey []byte
	if prvKey, err = hex.DecodeString(profile.PrivateKey); err != nil {
		log.Errorf("Decode private key error: %+v", err)
		return
	}
	schemeOutput := concealedSupi.SchemeOutput

	//get key length (scheme dependent)
	keyLen := 32 //A_SCHEME
	macLen := A_MAC_LEN
	uncompressed := false

	if scheme == nas.ProtectionSchemeECIESProfileB {
		macLen = B_MAC_LEN
		if schemeOutput[0] == 0x02 || schemeOutput[0] == 0x03 {
			keyLen = 33 // ceil(log(2, q)/8) + 1 = 33
			uncompressed = false
		} else if schemeOutput[0] == 0x04 {
			keyLen = 65 // 2*ceil(log(2, q)/8) + 1 = 65
			uncompressed = true
		} else {
			log.Error("input error")
			err = fmt.Errorf("suci input error\n")
			return
		}
	} else if scheme != nas.ProtectionSchemeECIESProfileA {
		err = fmt.Errorf("Unknown scheme")
		return
	}
	var mac, pubKey, ciphertext []byte
	if mac, pubKey, ciphertext, err = decompose(schemeOutput, keyLen, macLen); err != nil {
		return
	}

	var sharedKey, kdfKey, encKey, icb, macKey, macTag []byte
	if scheme == nas.ProtectionSchemeECIESProfileA {
		if sharedKey, err = curve25519.X25519(prvKey, pubKey); err != nil {
			log.Errorf("X25519 error: %+v", err)
			return
		}
		kdfKey = AnsiX963KDF(sharedKey, pubKey, A_ENC_K_LEN, A_MAC_K_LEN, A_HASH_LEN)
		encKey = kdfKey[:A_ENC_K_LEN]
		icb = kdfKey[A_ENC_K_LEN : A_ENC_K_LEN+A_ICB_LEN]
		macKey = kdfKey[len(kdfKey)-A_MAC_K_LEN:]
		macTag, err = HmacSha256(ciphertext, macKey, A_MAC_LEN)

	} else {
		var x, y *big.Int
		if uncompressed {
			x = new(big.Int).SetBytes(pubKey[1:(keyLen/2 + 1)])
			y = new(big.Int).SetBytes(pubKey[(keyLen/2 + 1):])
		} else {
			x, y = uncompressKey(pubKey, prvKey)
			if x == nil || y == nil {
				log.Error("Uncompressed key has invalid point")
				err = fmt.Errorf("Key uncompression error\n")
				return
			}
		}

		// x-coordinate is the shared key
		tmp, _ := elliptic.P256().ScalarMult(x, y, prvKey)
		sharedKey = tmp.Bytes()
		if uncompressed {
			pubKey = CompressKey(pubKey, y)
		}
		kdfKey = AnsiX963KDF(sharedKey, pubKey, B_ENC_K_LEN, B_MAC_K_LEN, B_HASH_LEN)
		encKey = kdfKey[:B_ENC_K_LEN]
		icb = kdfKey[B_ENC_K_LEN : B_ENC_K_LEN+B_ICB_LEN]
		macKey = kdfKey[len(kdfKey)-B_MAC_K_LEN:]
		macTag, err = HmacSha256(ciphertext, macKey, B_MAC_LEN)
	}

	if !bytes.Equal(macTag, mac) {
		log.Error("MAC unmatches")
		err = fmt.Errorf("MAC failed\n")
		return
	}
	var plaintext []byte
	if plaintext, err = Aes128ctr(ciphertext, encKey, icb); err != nil {
		return
	}
	var tmpStr string

	tmpStr = hex.EncodeToString(swapnibbles(plaintext))
	if tmpStr[len(tmpStr)-1] == 'f' {
		tmpStr = tmpStr[:len(tmpStr)-1]
	}

	plmn := concealedSupi.PlmnId.String()
	supi = "imsi-" + plmn + tmpStr
	return
}
