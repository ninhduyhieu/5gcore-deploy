package sepp

import (
	"github.com/reogac/sbi/models"
	"strings"
)

func parsePlmnId(serviceId string) *models.PlmnId {
	parts := strings.Split(serviceId, "-")
	if l := len(parts); l >= 3 {
		return &models.PlmnId{
			Mnc: parts[2],
			Mcc: parts[1],
		}
	}
	return nil
}
