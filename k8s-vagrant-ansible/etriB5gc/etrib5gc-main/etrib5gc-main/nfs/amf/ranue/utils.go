package ranue

import (
	"github.com/reogac/sbi/models"
)

func isGpp(access models.AccessType) bool {
	return access == models.ACCESSTYPE_3GPP_ACCESS
}

func getAccessType(isGpp bool) models.AccessType {
	if isGpp {
		return models.ACCESSTYPE_3GPP_ACCESS
	}
	return models.ACCESSTYPE_NON_3GPP_ACCESS
}
