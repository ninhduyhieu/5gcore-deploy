package suci

const (
	FORMAT_IMSI byte = 0
	FORMAT_NAI  byte = 1
)

func Nai2String(buf []byte) string {
	panic("Nai2String is not implemented")
	return ""
}

// TS 24.501 9.11.3.4
// suci(imsi) =
// "suci-0-${mcc}-${mnc}-${routingIndentifier}-${protectionScheme}-${homeNetworkPublicKeyIdentifier}-${schemeOutput}"
// suci(nai) = "nai-${naiString}"
func Suci2String(buf []byte) (suci string, plmnid string, err error) {
	/*
		var mcc, mnc, routingInd, protectionScheme, homeNetworkPublicKeyIdentifier, schemeOutput string

		format := (buf[0] & 0xf0) >> 4
		if format == FORMAT_NAI {
			suci = Nai2String(buf)
			return
		}
		//IMSI format
		var plmnid common.PlmnId
		if err = plmnid.SetBytes(buf[1:4]); err != nil {
			return
		}
		// Routing
		// Indicator
		var routingIndBytes []byte
		routingIndBytes = append(routingIndBytes, bits.RotateLeft8(buf[4], 4))
		routingIndBytes = append(routingIndBytes, bits.RotateLeft8(buf[5], 4))
		routingInd = hex.EncodeToString(routingIndBytes)

		if idx := strings.Index(routingInd, "f"); idx != -1 {
			routingInd = routingInd[0:idx]
		}

		scheme: = fmt.Sprintf("%x", buf[6]) // convert byte to hex string without leading 0s
		keyind = fmt.Sprintf("%d", buf[7])

		// output
		var ciphertext string
		if scheme == NULL_SCHEME {
			// MSIN
			var msinBytes []byte
			for i := 8; i < len(buf); i++ {
				msinBytes = append(msinBytes, bits.RotateLeft8(buf[i], 4))
			}
			schemeOutput = hex.EncodeToString(msinBytes)
			if schemeOutput[len(schemeOutput)-1] == 'f' {
				schemeOutput = schemeOutput[:len(schemeOutput)-1]
			}
		} else {
			ciphertext = hex.EncodeToString(buf[8:])
		}

		suci = strings.Join([]string{
			"suci", "0", plmnid.Mcc, plmnid.Mnc, rid, scheme, keyind, cihpertext,
		}, "-")
	*/
	return
}
