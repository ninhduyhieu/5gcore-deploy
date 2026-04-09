package sm

import (
	"etrib5gc/nfs/smf/context"
	"github.com/reogac/nas"
)

func (smCtx *SmContext) processPco(pco *nas.ExtendedProtocolConfigurationOptions) {
	if pco == nil {
		return
	}
	newPco := &nas.ExtendedProtocolConfigurationOptions{}
	dnsV4, dnsV6 := context.GetDns(smCtx.dnn)
	pcscfV4, pcscfV6 := context.GetPcscf(smCtx.dnn)
	//var err error
	for _, unit := range pco.Units() {
		switch unit.Id {
		case nas.DNSServerIPv4AddressRequestUL:
			if len(dnsV4) > 0 {
				newPco.AddUnit(nas.PcoUnit{
					Id:      unit.Id,
					Content: dnsV4.To4(),
				})
			}
		case nas.DNSServerIPv6AddressRequestUL:
			if len(dnsV6) > 0 {
				newPco.AddUnit(nas.PcoUnit{
					Id:      unit.Id,
					Content: dnsV6.To16(),
				})
			}

		case nas.PCSCFIPv4AddressRequestUL:
			if len(pcscfV4) > 0 {
				newPco.AddUnit(nas.PcoUnit{
					Id:      unit.Id,
					Content: pcscfV4.To4(),
				})
			}

		case nas.PCSCFIPv6AddressRequestUL:
			if len(pcscfV6) > 0 {
				newPco.AddUnit(nas.PcoUnit{
					Id:      unit.Id,
					Content: pcscfV6.To16(),
				})
			}

		case nas.IPv4LinkMTURequestUL:
			mtu := context.Ipv4LinkMtu()
			newPco.AddUnit(nas.PcoUnit{
				Id:      unit.Id,
				Content: []byte{uint8(mtu >> 8), uint8(mtu & 0xff)},
			})

		case nas.IMCNSubsystemSignalingFlagUL:
			smCtx.Trace("Didn't Implement container type IMCNSubsystemSignalingFlagUL")
		case nas.NotSupportedUL:
			smCtx.Trace("Didn't Implement container type NotSupportedUL")
		case nas.MSSupportOfNetworkRequestedBearerControlIndicatorUL:
			smCtx.Trace("Didn't Implement container type MSSupportOfNetworkRequestedBearerControlIndicatorUL")
		case nas.DSMIPv6HomeAgentAddressRequestUL:
			smCtx.Trace("Didn't Implement container type DSMIPv6HomeAgentAddressRequestUL")
		case nas.DSMIPv6HomeNetworkPrefixRequestUL:
			smCtx.Trace("Didn't Implement container type DSMIPv6HomeNetworkPrefixRequestUL")
		case nas.DSMIPv6IPv4HomeAgentAddressRequestUL:
			smCtx.Trace("Didn't Implement container type DSMIPv6IPv4HomeAgentAddressRequestUL")
		case nas.IPAddressAllocationViaNASSignallingUL:
			smCtx.Trace("Didn't Implement container type IPAddressAllocationViaNASSignallingUL")
		case nas.IPv4AddressAllocationViaDHCPv4UL:
			smCtx.Trace("Didn't Implement container type IPv4AddressAllocationViaDHCPv4UL")
		case nas.MSISDNRequestUL:
			smCtx.Trace("Didn't Implement container type MSISDNRequestUL")
		case nas.IFOMSupportRequestUL:
			smCtx.Trace("Didn't Implement container type IFOMSupportRequestUL")
		case nas.MSSupportOfLocalAddressInTFTIndicatorUL:
			smCtx.Trace("Didn't Implement container type MSSupportOfLocalAddressInTFTIndicatorUL")
		case nas.PCSCFReSelectionSupportUL:
			smCtx.Trace("Didn't Implement container type PCSCFReSelectionSupportUL")
		case nas.NBIFOMRequestIndicatorUL:
			smCtx.Trace("Didn't Implement container type NBIFOMRequestIndicatorUL")
		case nas.NBIFOMModeUL:
			smCtx.Trace("Didn't Implement container type NBIFOMModeUL")
		case nas.NonIPLinkMTURequestUL:
			smCtx.Trace("Didn't Implement container type NonIPLinkMTURequestUL")
		case nas.APNRateControlSupportIndicatorUL:
			smCtx.Trace("Didn't Implement container type APNRateControlSupportIndicatorUL")
		case nas.UEStatus3GPPPSDataOffUL:
			smCtx.Trace("Didn't Implement container type UEStatus3GPPPSDataOffUL")
		case nas.ReliableDataServiceRequestIndicatorUL:
			smCtx.Trace("Didn't Implement container type ReliableDataServiceRequestIndicatorUL")
		case nas.AdditionalAPNRateControlForExceptionDataSupportIndicatorUL:
			smCtx.Trace("Didn't Implement container type AdditionalAPNRateControlForExceptionDataSupportIndicatorUL")
		case nas.PDUSessionIDUL:
			smCtx.Trace("Didn't Implement container type PDUSessionIDUL")
		case nas.EthernetFramePayloadMTURequestUL:
			smCtx.Trace("Didn't Implement container type EthernetFramePayloadMTURequestUL")
		case nas.UnstructuredLinkMTURequestUL:
			smCtx.Trace("Didn't Implement container type UnstructuredLinkMTURequestUL")
		case nas.I5GSMCauseValueUL:
			smCtx.Trace("Didn't Implement container type 5GSMCauseValueUL")
		case nas.QoSRulesWithTheLengthOfTwoOctetsSupportIndicatorUL:
			smCtx.Trace("Didn't Implement container type QoSRulesWithTheLengthOfTwoOctetsSupportIndicatorUL")
		case nas.QoSFlowDescriptionsWithTheLengthOfTwoOctetsSupportIndicatorUL:
			smCtx.Trace("Didn't Implement container type QoSFlowDescriptionsWithTheLengthOfTwoOctetsSupportIndicatorUL")
		case nas.LinkControlProtocolUL:
			smCtx.Trace("Didn't Implement container type LinkControlProtocolUL")
		case nas.PushAccessControlProtocolUL:
			smCtx.Trace("Didn't Implement container type PushAccessControlProtocolUL")
		case nas.ChallengeHandshakeAuthenticationProtocolUL:
			smCtx.Trace("Didn't Implement container type ChallengeHandshakeAuthenticationProtocolUL")
		case nas.InternetProtocolControlProtocolUL:
			smCtx.Trace("Didn't Implement container type InternetProtocolControlProtocolUL")
		default:
			smCtx.Trace("Unknown Container ID [%d]", unit.Id)
		}
	}
	if len(newPco.Units()) > 0 {
		smCtx.pco = newPco
	}
}
