package context

import (
	"github.com/reogac/nas"
)

func FillRegistrationAccept(msg *nas.RegistrationAccept, isGpp bool) (err error) {
	msg.RegistrationResult.SetSmsAllowed(false)
	// 5gs network feature support
	if Get5gsNwFeatSuppEnable() {
		features := new(nas.NetworkFeatureSupport)
		if isGpp {
			features.SetIMSVoPS3GPP(Get5gsNwFeatSuppImsVoPS() == 1)
		} else {
			features.SetIMSVoPSN3GPP(Get5gsNwFeatSuppImsVoPS() == 1)
		}
		features.SetEMC(Get5gsNwFeatSuppEmc())
		features.SetEMF(Get5gsNwFeatSuppEmf())
		features.SetIWKN26(Get5gsNwFeatSuppIwkN26() == 1)
		features.SetMPSI(Get5gsNwFeatSuppMpsi() == 1)
		features.SetEMCN(Get5gsNwFeatSuppEmcN3() == 1)
		features.SetMCSI(Get5gsNwFeatSuppMcsi() == 1)
		msg.NetworkFeatureSupport = features
	}
	//set timer values
	if isGpp {
		msg.T3512Value = &nas.GprsTimer3{
			Value: T3512(),
		}
	} else {
		msg.Non3gppDeRegistrationTimerValue = &nas.GprsTimer2{
			Value: GetNon3GppDeregistrationTimerValue(),
		}
	}
	msg.T3502Value = &nas.GprsTimer2{
		Value: T3502(),
	}
	/*
		//TODO: build the list
			msg.EquivalentPlmns = nas.NewPlmnList(GetEquivalentPlmnList())
	*/
	return
}
