package sm

import (
	"fmt"
	"github.com/lvdund/ngap/ies"
	"github.com/reogac/nas"
	"github.com/reogac/sbi/models"
	"net"
	"strings"
)

func Nas2SbiPduSessionType(t uint8) models.PduSessionType {
	switch t {
	case nas.PduSessionTypeIpv4:
		return models.PDUSESSIONTYPE_IPV4
	case nas.PduSessionTypeIpv6:
		return models.PDUSESSIONTYPE_IPV6
	case nas.PduSessionTypeIpv4Ipv6:
		return models.PDUSESSIONTYPE_IPV4V6
	case nas.PduSessionTypeUnstructured:
		return models.PDUSESSIONTYPE_UNSTRUCTURED
	case nas.PduSessionTypeEthernet:
		return models.PDUSESSIONTYPE_ETHERNET
	}
	return models.PDUSESSIONTYPE_IPV4
}

// utility functions
func ueIpBytes(ip net.IP, sessionType uint8) (addr []byte) {
	switch sessionType {
	case nas.PduSessionTypeIpv6:
		fallthrough
	case nas.PduSessionTypeIpv4Ipv6:
		addr = make([]byte, len(ip))
		copy(addr[:], ip)
	default: //default is IpV4
		//case nasMessage.PduSessionTypeIpv4:
		ipv4 := ip.To4()
		addr = make([]byte, len(ipv4))
		copy(addr[:], ipv4)
	}
	return
}

func newUint8(v uint8) *uint8 {
	return &v
}

func upSecurityToNgap(input *models.UpSecurity) (output *ies.SecurityIndication) {
	/* TODO
	obj.SecurityIndication = new(ies.SecurityIndication)
				securityIndication := obj.SecurityIndication
				upSecurity := ctx.UpSecurity
				maximumDataRatePerUEForUserPlaneIntegrityProtectionForUpLink := ctx.
					MaximumDataRatePerUEForUserPlaneIntegrityProtectionForUpLink

				switch upSecurity.UpIntegr {
				case models.UpIntegrity_REQUIRED:
					securityIndication.IntegrityProtectionIndication.Value = ies.
						IntegrityProtectionIndicationPresentRequired
				case models.UpIntegrity_PREFERRED:
					securityIndication.IntegrityProtectionIndication.Value = ies.
						IntegrityProtectionIndicationPresentPreferred
				case models.UpIntegrity_NOT_NEEDED:
					securityIndication.IntegrityProtectionIndication.Value = ies.
						IntegrityProtectionIndicationPresentNotNeeded
				}
				switch upSecurity.UpConfid {
				case models.UpConfidentiality_REQUIRED:
					securityIndication.ConfidentialityProtectionIndication.Value = ies.
						ConfidentialityProtectionIndicationPresentRequired
				case models.UpConfidentiality_PREFERRED:
					securityIndication.ConfidentialityProtectionIndication.Value = ies.
						ConfidentialityProtectionIndicationPresentPreferred
				case models.UpConfidentiality_NOT_NEEDED:
					securityIndication.ConfidentialityProtectionIndication.Value = ies.
						ConfidentialityProtectionIndicationPresentNotNeeded
				}

				// Present only when Integrity Indication within the Security
				// Indication is set to "required" or "preferred"
				integrityProtectionInd := securityIndication.IntegrityProtectionIndication.Value
				if integrityProtectionInd == ies.IntegrityProtectionIndicationPresentRequired ||
					integrityProtectionInd == ies.IntegrityProtectionIndicationPresentPreferred {
					securityIndication.MaximumIntegrityProtectedDataRateUL = new(ies.MaximumIntegrityProtectedDataRate)
					switch maximumDataRatePerUEForUserPlaneIntegrityProtectionForUpLink {
					case models.MaxIntegrityProtectedDataRate_MAX_UE_RATE:
						securityIndication.MaximumIntegrityProtectedDataRateUL.Value = ies.
							MaximumIntegrityProtectedDataRatePresentMaximumUERate
					case models.MaxIntegrityProtectedDataRate__64_KBPS:
						securityIndication.MaximumIntegrityProtectedDataRateUL.Value = ies.
							MaximumIntegrityProtectedDataRatePresentBitrate64kbs
					}
				}
	*/
	return
}

func extractIdFromLocation(headers map[string]string) string {
	if v, ok := headers["Location"]; ok {
		if parts := strings.Split(v, "/"); len(parts) > 0 {
			return parts[len(parts)-1]
		}
	}
	return ""
}
func smContextRef(supi string, sid uint32) string {
	return fmt.Sprintf("supi=%s-sid=%d", supi, sid)
}

func plmnIdFromSupi(supi string) (*models.PlmnId, error) {
	//NOTE: temporarily solution, need to fix SUPI format from AMF
	parts := strings.Split(supi, "-")
	if len(parts) <= 1 {
		return nil, fmt.Errorf("Invalid SUPI %s", supi)
	}
	return &models.PlmnId{
		Mcc: parts[1][0:3],
		Mnc: parts[1][3:5],
	}, nil
	/*
		var suci nas.Suci
		if err := suci.Parse(supi); err != nil {
			return nil, fmt.Errorf("Fail to parse SUPI %s: %+v", supi, err)
		}
		if suci.GetSupiFormat() != nas.SupiFormatImsi {
			return nil, fmt.Errorf("Invalid SUPI %s", supi)
		}
		nasSupi, _ := suci.Content.(*nas.SupiImsi)
		mcc, mnc := nasSupi.PlmnId.Get()
		return &models.PlmnId{
			Mcc: mcc,
			Mnc: mnc,
		}, nil
	*/
}
