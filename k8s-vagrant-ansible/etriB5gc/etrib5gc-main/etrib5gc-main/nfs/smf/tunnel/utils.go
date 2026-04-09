package tunnel

import (
	"github.com/lvdund/ngap/ies"
	"github.com/reogac/sbi/models"
)

func canBeDefault(pcc *models.PccRule) bool {
	//a PccRule withouth traffic control data and qos flow can be selected as the
	//default path
	if len(pcc.RefTcData) == 0 && len(pcc.RefQosData) == 0 {
		return true
	}
	return false
}

func buildNgapArp(arp *models.Arp) ies.AllocationAndRetentionPriority {
	if arp == nil {
		return ies.AllocationAndRetentionPriority{
			PriorityLevelARP: 15,
			PreemptionCapability: ies.PreemptionCapability{
				Value: ies.PreemptionCapabilityShallnottriggerpreemption,
			},
			PreemptionVulnerability: ies.PreemptionVulnerability{
				Value: ies.PreemptionVulnerabilityNotpreemptable,
			},
		}
	} else {
		//TODO: build from ARP
		return ies.AllocationAndRetentionPriority{
			PriorityLevelARP: 15,
			PreemptionCapability: ies.PreemptionCapability{
				Value: ies.PreemptionCapabilityShallnottriggerpreemption,
			},
			PreemptionVulnerability: ies.PreemptionVulnerability{
				Value: ies.PreemptionVulnerabilityNotpreemptable,
			},
		}
		/*
			if arp == nil {
				return 0, 0, 0
			}
			var arpPriorityLevel int64
			var arpPreEmptionCapability aper.Enumerated
			var arpPreEmptionVulnerability aper.Enumerated

			arpPriorityLevel = int64(arp.PriorityLevel)
			switch arp.PreemptCap {
			case models.PreemptionCapability_NOT_PREEMPT:
				arpPreEmptionCapability = ngapType.PreEmptionCapabilityPresentShallNotTriggerPreEmption
			case models.PreemptionCapability_MAY_PREEMPT:
				arpPreEmptionCapability = ngapType.PreEmptionCapabilityPresentMayTriggerPreEmption
			default:
				arpPreEmptionCapability = ngapType.PreEmptionCapabilityPresentShallNotTriggerPreEmption
			}
			switch arp.PreemptVuln {
			case models.PreemptionVulnerability_NOT_PREEMPTABLE:
				arpPreEmptionVulnerability = ngapType.PreEmptionVulnerabilityPresentNotPreEmptable
			case models.PreemptionVulnerability_PREEMPTABLE:
				arpPreEmptionVulnerability = ngapType.PreEmptionVulnerabilityPresentPreEmptable
			default:
				arpPreEmptionVulnerability = ngapType.PreEmptionVulnerabilityPresentNotPreEmptable
			}

			return arpPriorityLevel, arpPreEmptionCapability, arpPreEmptionVulnerability
		*/
	}
}
